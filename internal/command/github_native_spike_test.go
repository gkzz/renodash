package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNativeSpikeIssue(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GH_TOKEN", "primary")
	t.Setenv("GITHUB_TOKEN", "secondary")
	t.Setenv("GH_HOST", "other.example")
	if _, err := exec.LookPath("gh"); !errors.Is(err, exec.ErrNotFound) {
		t.Fatal("expected gh to be absent")
	}
	const issue = `{"number":5,"body":"Dashboard","html_url":"https://github.com/Owner/Repo/issues/5","updated_at":"2026-11-25T00:00:00Z"}`
	client := offlineNativeSpike(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/Owner/Repo/issues/5" || r.Method != http.MethodGet {
			t.Error("unexpected Issue request")
		}
		if r.Header.Get("Authorization") != "Bearer primary" {
			t.Error("unexpected authorization")
		}
		_, _ = w.Write([]byte(issue))
	})
	got, err := client.FetchIssue(context.Background(), "Owner/Repo", 5)
	if err != nil || string(got) != issue {
		t.Fatalf("Issue contract failed: %v", err)
	}
}

func offlineNativeSpike(t *testing.T, handler http.HandlerFunc) nativeSpikeClient {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	client := newNativeSpike()
	// Only the test transport remaps the fixed API host to the local TLS server.
	client.http.Transport = spikeRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Scheme != "https" || r.URL.Host != "api.github.com" || r.URL.User != nil {
			t.Error("request escaped fixed API origin")
			return nil, errors.New("unexpected destination")
		}
		r = r.Clone(r.Context())
		r.URL.Host = server.Listener.Addr().String()
		return server.Client().Transport.RoundTrip(r)
	})
	return client
}

type spikeRoundTrip func(*http.Request) (*http.Response, error)

func (f spikeRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNativeSpikeContextBeforeAuthentication(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	client := newNativeSpike()
	client.http.Transport = spikeRoundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("request reached transport")
		return nil, nil
	})
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, release := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer release()
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"canceled", canceled, context.Canceled},
		{"expired", expired, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, fetch := range []func(context.Context, string, int) ([]byte, error){client.FetchIssue, client.FetchPR} {
				data, err := fetch(tc.ctx, "Owner/Repo", 4)
				if data != nil || !errors.Is(err, tc.want) {
					t.Errorf("data=%q err=%v, want no data and %v", data, err, tc.want)
				}
			}
		})
	}
}

func TestNativeSpikeAuthenticationAndRedirects(t *testing.T) {
	for _, name := range []string{"unset", "empty", "fallback", "same host redirect", "cross host redirect", "wrapped transport error"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GH_TOKEN", "")
			t.Setenv("GITHUB_TOKEN", "")
			if name == "unset" {
				_ = os.Unsetenv("GH_TOKEN")
				_ = os.Unsetenv("GITHUB_TOKEN")
			}
			var calls atomic.Int32
			client := offlineNativeSpike(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("Authorization") != "Bearer fallback" {
					t.Error("unexpected fallback token")
				}
				switch name {
				case "same host redirect":
					w.Header().Set("Location", "/another-path")
				case "cross host redirect":
					w.Header().Set("Location", "https://other.example/credential-marker")
				default:
					return
				}
				w.WriteHeader(http.StatusTemporaryRedirect)
			})
			if name != "unset" && name != "empty" {
				t.Setenv("GITHUB_TOKEN", "fallback")
			}
			if name == "wrapped transport error" {
				client.http.Transport = spikeRoundTrip(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, fmt.Errorf("wrapper: %w", errors.New("credential-marker"))
				})
			}
			data, err := client.FetchIssue(context.Background(), "Owner/Repo", 5)
			switch name {
			case "fallback":
				if err != nil || calls.Load() != 1 {
					t.Fatalf("fallback: calls=%d err=%v", calls.Load(), err)
				}
			case "unset", "empty":
				assertSpikeError(t, data, err, "authentication", false, 0)
				if calls.Load() != 0 {
					t.Fatal("missing token triggered a request")
				}
			default:
				code := "redirect"
				if name == "wrapped transport error" {
					code = "fetch_failed"
				}
				assertSpikeError(t, data, err, code, false, 0)
				if calls.Load() != 1 {
					t.Fatal("expected a single request")
				}
			}
		})
	}
}

func assertSpikeError(t *testing.T, data []byte, err error, want string, retryable bool, wait time.Duration) {
	t.Helper()
	code, retry, delay := requestFailure(err)
	if data != nil || err == nil || code != want || retry != retryable || delay != wait {
		t.Fatalf("got data=%t code=%s retryable=%t wait=%s, want %s/%t/%s", data != nil, code, retry, delay, want, retryable, wait)
	}
	for _, marker := range []string{"credential-marker", "Bearer", "fallback"} {
		if strings.Contains(err.Error(), marker) {
			t.Fatal("error exposes request or response diagnostics")
		}
	}
}

func TestNativeSpikeHTTPFailures(t *testing.T) {
	t.Setenv("GH_TOKEN", "credential-marker")
	for _, tc := range []struct {
		name, body, code string
		status           int
		headers          map[string]string
		retry            bool
		truncated        bool
		wait             time.Duration
	}{
		{name: "invalid token", status: 401, code: "authentication"},
		{name: "permission", status: 403, code: "permission_denied"},
		{name: "not found", status: 404, code: "not_found_or_inaccessible"},
		{name: "unknown wait", status: 429, code: "rate_limit"},
		{name: "server error", status: 503, code: "http_5xx", retry: true},
		{name: "truncated server error", status: 503, code: "http_5xx", retry: true, truncated: true},
		{name: "truncated rate limit", status: 429, code: "rate_limit", headers: map[string]string{"Retry-After": "7"}, retry: true, wait: 7 * time.Second, truncated: true},
		{name: "truncated authentication error", status: 401, code: "authentication", truncated: true},
		{name: "truncated success", status: 200, code: "fetch_failed", truncated: true, body: "{}"},
		{name: "secondary", status: 403, code: "rate_limit", headers: map[string]string{"Retry-After": "7"}, retry: true, wait: 7 * time.Second},
		{
			name:      "secondary with body read error",
			status:    403,
			code:      "rate_limit",
			headers:   map[string]string{"X-RateLimit-Remaining": "42"},
			body:      `{"message":"You have exceeded a secondary rate limit. credential-marker"}`,
			truncated: true,
		},
		{name: "primary", status: 403, code: "rate_limit", headers: map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1"}, retry: true},
		{name: "GraphQL limit", status: 200, code: "rate_limit", headers: map[string]string{"Retry-After": "7"}, retry: true, wait: 7 * time.Second, body: `{"errors":[{"type":"RATE_LIMITED","message":"credential-marker"}]}`},
		{name: "GraphQL limit with body read error", status: 200, code: "rate_limit", headers: map[string]string{"Retry-After": "7"}, retry: true, wait: 7 * time.Second, body: `{"errors":[{"type":"RATE_LIMITED","message":"credential-marker"}]}`, truncated: true},
		{name: "partial GraphQL data", status: 200, code: "graphql_error", body: `{"data":{"repository":{"pullRequest":{"number":4}}},"errors":[{"message":"credential-marker"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := offlineNativeSpike(t, func(w http.ResponseWriter, r *http.Request) {
				for key, value := range tc.headers {
					w.Header().Set(key, value)
				}
				w.Header().Set("X-Renodash-Status", "credential-marker")
				if tc.truncated {
					w.Header().Set("Content-Length", strconv.Itoa(len(tc.body)+1))
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
			assertSpikeError(t, data, err, tc.code, tc.retry, tc.wait)
			if tc.status != 200 || tc.code == "fetch_failed" {
				data, err = client.FetchIssue(context.Background(), "Owner/Repo", 5)
				assertSpikeError(t, data, err, tc.code, tc.retry, tc.wait)
			}
		})
	}
}

func TestNativeSpikeRateLimitSurvivesAttemptDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		client := newNativeSpike()
		client.token = "credential-marker"
		client.http.Transport = spikeRoundTrip(func(r *http.Request) (*http.Response, error) {
			return spikeRateLimitUntilDone(r.Context()), nil
		})
		data, err := client.FetchPR(ctx, "Owner/Repo", 4)
		assertSpikeError(t, data, err, "rate_limit", true, time.Minute)
	})
}

func spikeRateLimitUntilDone(ctx context.Context) *http.Response {
	reader, writer := io.Pipe()
	go func() {
		<-ctx.Done()
		_ = writer.CloseWithError(ctx.Err())
	}()
	return &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": {"60"}},
		Body:       reader,
	}
}

func TestNativeSpikeRetryAfterAttemptDeadline(t *testing.T) {
	issue := spikeFixture(t, "dashboard-issue.json")
	pages := map[int][]byte{}
	for _, number := range []int{53, 51, 30, 52} {
		pages[number] = spikeFixture(t, fmt.Sprintf("pr-%d.graphql.json", number))
	}
	for _, tc := range []struct {
		name     string
		budget   time.Duration
		cancelAt time.Duration
		elapsed  time.Duration
		calls    []int
		failures []retrievalError
	}{
		{
			name: "recovers", budget: 5 * time.Minute, elapsed: 90 * time.Second,
			calls: []int{53, 51, 51, 30, 52},
		},
		{
			name: "insufficient budget", budget: 89 * time.Second, elapsed: 30 * time.Second,
			calls:    []int{53, 51},
			failures: spikeRateLimitFailures(1),
		},
		{
			name: "exhausts retries", budget: 5 * time.Minute, elapsed: 210 * time.Second,
			calls:    []int{53, 51, 51, 51},
			failures: spikeRateLimitFailures(3),
		},
		{
			name: "canceled during wait", budget: 5 * time.Minute, cancelAt: 31 * time.Second, elapsed: 31 * time.Second,
			calls: []int{53, 51},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), tc.budget)
				defer cancel()
				start := time.Now()
				if tc.cancelAt != 0 {
					timer := time.AfterFunc(tc.cancelAt, cancel)
					defer timer.Stop()
				}
				var calls []int
				var attempts int
				client := newNativeSpike()
				client.token = "fixture-token"
				client.http.Transport = spikeRoundTrip(func(r *http.Request) (*http.Response, error) {
					body := issue
					if r.Method != http.MethodGet {
						number, _ := spikeGraphQL(t, r)
						calls = append(calls, number)
						if number == 51 {
							attempts++
							wantStart := time.Duration(attempts-1) * 90 * time.Second
							if elapsed := time.Since(start); elapsed != wantStart {
								t.Errorf("PR #51 attempt %d started at %s, want %s", attempts, elapsed, wantStart)
							}
							if attempts == 1 || len(tc.failures) != 0 {
								return spikeRateLimitUntilDone(r.Context()), nil
							}
						}
						body = pages[number]
					}
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
				})
				var stdout, stderr bytes.Buffer
				err := writeGetResult(ctx, client, "gkzz/renodash", 5, "gkzz/renodash#5", true, true, &stdout, &stderr)
				if elapsed := time.Since(start); elapsed != tc.elapsed || !slices.Equal(calls, tc.calls) {
					t.Fatalf("elapsed=%s calls=%v, want %s and %v", elapsed, calls, tc.elapsed, tc.calls)
				}
				if tc.cancelAt != 0 {
					if !errors.Is(err, context.Canceled) || stdout.Len() != 0 {
						t.Fatalf("cancellation: err=%v stdout=%q", err, stdout.String())
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				var result getResult
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.PullRequests == nil || result.Retrieval == nil {
					t.Fatalf("invalid get result: %v", err)
				}
				var numbers []int
				for _, pr := range *result.PullRequests {
					numbers = append(numbers, pr.Number)
				}
				wantNumbers, wantStatus := []int{53, 51, 30, 52}, "complete"
				if len(tc.failures) != 0 {
					wantNumbers, wantStatus = []int{53}, "partial"
				}
				if !slices.Equal(numbers, wantNumbers) || result.Retrieval.Status != wantStatus ||
					!slices.Equal(result.Retrieval.Errors, tc.failures) {
					t.Fatalf("PRs=%v retrieval=%+v, want %v, %s and %+v", numbers, result.Retrieval, wantNumbers, wantStatus, tc.failures)
				}
			})
		})
	}
}

func spikeRateLimitFailures(attempts int) []retrievalError {
	return []retrievalError{
		{Resource: "pull_request", Repository: "gkzz/renodash", Number: 51, Code: "rate_limit", Attempts: attempts, Retryable: true},
		{Resource: "pull_request", Repository: "gkzz/renodash", Number: 30, Code: "rate_limit", Retryable: true},
		{Resource: "pull_request", Repository: "gkzz/renodash", Number: 52, Code: "rate_limit", Retryable: true},
	}
}

func spikePage(head, commit, cursor string, more bool, checks string) string {
	_, body, _ := strings.Cut(graphPage(head, commit, cursor, more, checks), "\r\n\r\n")
	return body
}

func spikeGraphQL(t *testing.T, r *http.Request) (number int, cursor string) {
	t.Helper()
	var payload struct {
		Query     string
		Variables struct {
			Owner, Repo, Cursor string
			Number              int
		}
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Query != prQuery ||
		r.Method != http.MethodPost || r.URL.Path != "/graphql" {
		t.Error("unexpected GraphQL request")
	}
	return payload.Variables.Number, payload.Variables.Cursor
}

func TestNativeSpikePagination(t *testing.T) {
	t.Setenv("GH_TOKEN", "fixture-token")
	checkPage := make([]string, 100)
	wantTypes := make([]string, 0, 101)
	for i := range checkPage {
		checkPage[i] = fmt.Sprintf(`{"__typename":"CheckRun","name":"build-%d"}`, i)
		wantTypes = append(wantTypes, "CheckRun")
	}
	wantTypes = append(wantTypes, "StatusContext")
	first := spikePage("a", "a", "next", true, "["+strings.Join(checkPage, ",")+"]")
	last := spikePage("a", "a", "", false, `[{"__typename":"StatusContext","context":"external"}]`)
	empty := spikePage("a", "a", "", false, "[]")
	null := strings.Replace(empty, `{"contexts":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}`, "null", 1)
	for _, tc := range []struct {
		name, first, last, code string
		want                    []string
	}{
		{name: "mixed pages", first: first, last: last, want: wantTypes},
		{name: "empty", first: empty, want: []string{}},
		{name: "null", first: null, want: []string{}},
		{name: "missing metadata", first: `{"data":{"repository":{"pullRequest":{"number":4}}}}`, code: "invalid_metadata"},
		{name: "malformed JSON", first: "{", code: "decode_error"},
		{name: "changed head", first: first, last: spikePage("b", "b", "", false, "[]"), code: "head_changed"},
		{name: "mismatching commit", first: spikePage("a", "b", "", false, "[]"), code: "head_changed"},
		{name: "missing cursor", first: spikePage("a", "a", "", true, "[]"), code: "invalid_pagination"},
		{name: "repeated cursor", first: first, last: first, code: "invalid_pagination"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			client := offlineNativeSpike(t, func(w http.ResponseWriter, r *http.Request) {
				number, cursor := spikeGraphQL(t, r)
				if number != 4 || (calls.Add(1) == 1 && cursor != "") {
					t.Error("unexpected GraphQL variables")
				}
				if cursor == "" {
					_, _ = io.WriteString(w, tc.first)
				} else {
					if cursor != "next" {
						t.Error("unexpected continuation")
					}
					_, _ = io.WriteString(w, tc.last)
				}
			})
			data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
			wantCalls := int32(1)
			if tc.last != "" {
				wantCalls = 2
			}
			if calls.Load() != wantCalls {
				t.Fatalf("requests=%d want=%d", calls.Load(), wantCalls)
			}
			if tc.code != "" {
				assertSpikeError(t, data, err, tc.code, false, 0)
				return
			}
			var result struct {
				Checks []struct {
					Type string `json:"__typename"`
				} `json:"statusCheckRollup"`
			}
			if err != nil || json.Unmarshal(data, &result) != nil || result.Checks == nil {
				t.Fatalf("invalid result: %v", err)
			}
			got := make([]string, 0, len(result.Checks))
			for _, check := range result.Checks {
				got = append(got, check.Type)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("check types=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestNativeSpikeMergeMetadata(t *testing.T) {
	t.Setenv("GH_TOKEN", "fixture-token")
	for _, tc := range []struct{ name, fields, want string }{
		{"missing", "", "UNKNOWN"},
		{"null", `"mergeable":null,"mergeStateStatus":null,`, "UNKNOWN"},
		{"unknown", `"mergeable":"UNKNOWN","mergeStateStatus":"UNKNOWN",`, "UNKNOWN"},
		{"future value", `"mergeable":"FUTURE","mergeStateStatus":"FUTURE",`, "FUTURE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := strings.Replace(spikePage("a", "a", "", false, "[]"), `"number":4,`, `"number":4,`+tc.fields, 1)
			client := offlineNativeSpike(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, page)
			})
			data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
			if err != nil {
				t.Fatal(err)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatal(err)
			}
			wantRaw := map[string]string{"missing": "", "null": "null", "unknown": `"UNKNOWN"`, "future value": `"FUTURE"`}[tc.name]
			if string(raw["mergeable"]) != wantRaw || string(raw["mergeStateStatus"]) != wantRaw {
				t.Fatal("transport changed raw merge fields")
			}
			metadata, err := decodePullRequest(data, "Owner/Repo", 4, "")
			if err != nil || metadata.Mergeable != tc.want || metadata.MergeStateStatus != tc.want {
				t.Fatalf("unexpected normalized merge fields: %v", err)
			}
		})
	}
}

func TestNativeSpikeCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GH_TOKEN", "fixture-token")
	const repo = "gkzz/renodash"
	issue := spikeFixture(t, "dashboard-issue.json")
	pages := map[int][]byte{}
	for _, number := range []int{53, 51, 30, 52} {
		pages[number] = spikeFixture(t, fmt.Sprintf("pr-%d.graphql.json", number))
	}

	for _, scenario := range []string{"complete", "partial", "all failed", "strict", "invalid stdin", "unauthenticated commands"} {
		t.Run(scenario, func(t *testing.T) {
			calls := make(chan int, 10)
			client := offlineNativeSpike(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					calls <- 5
					if r.URL.Path != "/repos/gkzz/renodash/issues/5" {
						t.Error("unexpected Dashboard path")
					}
					_, _ = w.Write(issue)
					return
				}
				number, cursor := spikeGraphQL(t, r)
				calls <- number
				if cursor != "" {
					t.Error("unexpected fixture pagination")
				}
				if scenario == "all failed" || number == 51 && (scenario == "partial" || scenario == "strict") {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				_, _ = w.Write(pages[number])
			})
			var stdout, stderr bytes.Buffer
			if scenario == "unauthenticated commands" {
				t.Setenv("GH_TOKEN", "")
				t.Setenv("GITHUB_TOKEN", "")
				for _, args := range [][]string{{"parse"}, {"version"}, {"--help"}, {"get", "--help"}} {
					if runWithClient(client, args, strings.NewReader(""), &stdout, &stderr) != 0 {
						t.Fatalf("command requires authentication: %v", args)
					}
				}
				if len(calls) != 0 {
					t.Fatal("non-fetching command made an HTTP request")
				}
				return
			}
			args := []string{"get", "--with-pr-metadata", "--stdin"}
			if scenario != "strict" {
				args = append(args, "--allow-partial")
			}
			input := " " + repo + "#5\r\n"
			if scenario == "invalid stdin" {
				input = repo + "#5" + strings.Repeat(" ", 4096)
			}
			exit := runWithClient(client, args, strings.NewReader(input), &stdout, &stderr)
			close(calls)
			var order []int
			for number := range calls {
				order = append(order, number)
			}
			switch scenario {
			case "invalid stdin":
				if exit != 1 || stdout.Len() != 0 || len(order) != 0 || !strings.Contains(stderr.String(), "4096") {
					t.Fatal("invalid stdin did not fail before HTTP")
				}
				return
			case "strict":
				if exit != 1 || stdout.Len() != 0 || !slices.Equal(order, []int{5, 53, 51}) {
					t.Fatalf("strict failure: exit=%d calls=%v", exit, order)
				}
				return
			}
			if exit != 0 || !slices.Equal(order, []int{5, 53, 51, 30, 52}) {
				t.Fatalf("exit=%d calls=%v", exit, order)
			}
			var result getResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.PullRequests == nil || result.Retrieval == nil {
				t.Fatalf("invalid CLI result: %v", err)
			}
			var source githubIssue
			if err := json.Unmarshal(issue, &source); err != nil || result.Source.IssueURL != source.HTMLURL || len(result.Items) != 5 {
				t.Fatalf("Dashboard evidence missing: %v", err)
			}
			wantPRs := []int{53, 51, 30, 52}
			wantFailures := []int{}
			wantStatus := "complete"
			if scenario == "all failed" {
				wantPRs, wantFailures, wantStatus = nil, wantPRs, "partial"
			} else if scenario == "partial" {
				wantPRs, wantFailures, wantStatus = []int{53, 30, 52}, []int{51}, "partial"
			}
			if *result.PullRequests == nil || result.Retrieval.Errors == nil ||
				len(*result.PullRequests) != len(wantPRs) || len(result.Retrieval.Errors) != len(wantFailures) ||
				result.Retrieval.Status != wantStatus {
				t.Fatalf("unexpected result shape or retrieval status: %v", result.Retrieval)
			}
			for i, number := range wantPRs {
				data := spikeFixture(t, fmt.Sprintf("pr-%d.json", number))
				want, err := decodePullRequest(data, repo, number, (*result.PullRequests)[i].FetchedAt)
				if err != nil {
					t.Fatalf("decode saved PR #%d: %v", number, err)
				}
				if diff := cmp.Diff(want, (*result.PullRequests)[i], spikeJSONValue); diff != "" {
					t.Errorf("PR #%d differs from saved baseline (-want +got):\n%s", number, diff)
				}
			}
			for i, number := range wantFailures {
				want := retrievalError{Resource: "pull_request", Repository: repo, Number: number, Code: "permission_denied", Attempts: 1}
				if result.Retrieval.Errors[i] != want {
					t.Fatalf("unexpected partial failure: %+v", result.Retrieval.Errors[i])
				}
			}
		})
	}
}

// Compare check objects as JSON values, independent of whitespace and key order.
var spikeJSONValue = cmpopts.AcyclicTransformer("JSONValue", func(raw json.RawMessage) any {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	return value
})

func spikeFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../testdata/pr-metadata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNativeSpikeRetryUnits(t *testing.T) {
	t.Setenv("GH_TOKEN", "fixture-token")
	type pageCall struct {
		number int
		cursor string
	}
	for _, scenario := range []string{"page failure recovers", "timeouts recover", "timeout exhaustion", "unresolved rate limit", "cancel retry wait"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := []pageCall{}
			waits := 0
			client := newNativeSpike()
			client.http.Transport = spikeRoundTrip(func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				if !ok || time.Until(deadline) > 30*time.Second {
					t.Error("missing per-attempt deadline")
				}
				number, cursor := spikeGraphQL(t, r)
				calls = append(calls, pageCall{number, cursor})
				status, body := 200, spikePage("a", "a", "", false, "[]")
				if number == 5 {
					switch scenario {
					case "page failure recovers":
						if cursor == "" {
							body = spikePage("a", "a", "next", true, "[]")
						} else if len(calls) == 3 {
							status, body = 503, ""
						}
					case "timeouts recover", "timeout exhaustion", "cancel retry wait":
						if len(calls) <= 3 || scenario == "timeout exhaustion" {
							return nil, context.DeadlineExceeded
						}
					case "unresolved rate limit":
						status, body = 429, ""
					}
					body = strings.NewReplacer(`"number":4`, `"number":5`, "/pull/4", "/pull/5").Replace(body)
				}
				if number == 6 {
					body = strings.NewReplacer(`"number":4`, `"number":6`, "/pull/4", "/pull/6").Replace(body)
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			policy := retryPolicy{delay: func(int) time.Duration { return 0 }, wait: func(ctx context.Context, _ time.Duration) error {
				waits++
				if scenario == "cancel retry wait" {
					cancel()
					return waitForRetry(ctx, time.Hour)
				}
				return nil
			}}
			items := []dashpkg.Item{{PRLink: "../pull/4"}, {PRLink: "../pull/5"}, {PRLink: "pull/005"}, {PRLink: "../pull/6"}}
			prs, retrieval, err := collectPullRequests(ctx, client, "Owner/Repo", items, true, policy)
			wantCalls := []pageCall{{4, ""}, {5, ""}}
			wantWaits := 0
			switch scenario {
			case "page failure recovers":
				wantCalls = append(wantCalls, pageCall{5, "next"}, pageCall{5, ""}, pageCall{5, "next"}, pageCall{6, ""})
				wantWaits = 1
			case "timeouts recover", "timeout exhaustion":
				wantCalls = append(wantCalls, pageCall{5, ""}, pageCall{5, ""}, pageCall{6, ""})
				wantWaits = 2
			case "cancel retry wait":
				wantWaits = 1
			}
			if !slices.Equal(calls, wantCalls) || waits != wantWaits {
				t.Fatalf("calls=%v waits=%d", calls, waits)
			}
			if scenario == "cancel retry wait" {
				if !errors.Is(err, context.Canceled) || prs != nil {
					t.Fatalf("cancellation returned a result: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "timeout exhaustion":
				if len(prs) != 2 || prs[1].Number != 6 || retrieval.Status != "partial" || len(retrieval.Errors) != 1 ||
					retrieval.Errors[0].Attempts != 3 || retrieval.Errors[0].Code != "timeout" {
					t.Fatalf("unexpected exhaustion result: %+v", retrieval)
				}
			case "unresolved rate limit":
				if len(prs) != 1 || retrieval.Status != "partial" || len(retrieval.Errors) != 2 ||
					retrieval.Errors[0].Attempts != 1 || retrieval.Errors[1].Attempts != 0 ||
					retrieval.Errors[1].Code != "rate_limit" {
					t.Fatalf("unexpected rate limit result: %+v", retrieval)
				}
			default:
				if len(prs) != 3 || prs[0].Number != 4 || prs[1].Number != 5 || prs[2].Number != 6 ||
					retrieval.Status != "complete" {
					t.Fatalf("unexpected recovery: %+v", retrieval)
				}
			}
		})
	}
}

func TestNativeSpikeCancelsHTTPRequest(t *testing.T) {
	t.Setenv("GH_TOKEN", "fixture-token")
	started := make(chan struct{})
	released := make(chan struct{})
	stop := make(chan struct{})
	defer close(stop)
	client := offlineNativeSpike(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(released)
		case <-stop:
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := client.FetchIssue(ctx, "Owner/Repo", 5)
		result <- err
	}()
	select {
	case <-started:
		cancel()
	case <-ctx.Done():
		t.Fatal("request did not reach the server")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("request returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("request ignored cancellation")
	}
	select {
	case <-released:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not observe disconnect")
	}
}
