package command

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

func graphPage(head, commit, cursor string, more bool, checks string) string {
	page := `{"data":{"repository":{"pullRequest":{"number":4,"url":"https://github.com/Owner/Repo/pull/4","state":"OPEN","title":"Update","headRefOid":"` + head + `","updatedAt":"2026-11-25T00:00:00Z","commits":{"nodes":[{"commit":{"oid":"` + commit + `","statusCheckRollup":{"contexts":{"nodes":` + checks + `,"pageInfo":{"hasNextPage":` + map[bool]string{true: "true", false: "false"}[more] + `,"endCursor":"` + cursor + `"}}}}}]}}}}}`
	return "HTTP/2.0 200 OK\r\nContent-Type: application/json\r\n\r\n" + page
}
func prTestClient(t *testing.T, pages ...string) (*githubHTTPClient, *[]graphQLVariables) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GH_TOKEN", "fixture-token")
	var calls []graphQLVariables
	client := newGitHubHTTPClient()
	client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "https://api.github.com/graphql" ||
			req.Header.Get("Authorization") != "Bearer fixture-token" || req.Header.Get("Content-Type") != "application/json" {
			t.Fatal("unexpected GraphQL request")
		}
		defer req.Body.Close()
		var request graphQLRequest
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil || request.Query != prQuery {
			t.Fatalf("invalid GraphQL payload: %v", err)
		}
		calls = append(calls, request.Variables)
		if len(calls) > len(pages) {
			t.Fatal("unexpected extra GraphQL request")
		}
		return http.ReadResponse(bufio.NewReader(strings.NewReader(pages[len(calls)-1])), req)
	})
	return client, &calls
}

func TestHTTPPRPaginationAndArguments(t *testing.T) {
	pages := []string{graphPage("abc", "abc", "next", true, `[{"__typename":"CheckRun","name":"test"}]`), graphPage("abc", "abc", "", false, `[{"__typename":"StatusContext","context":"external"}]`)}
	client, calls := prTestClient(t, pages...)
	data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
	if err != nil {
		t.Fatal(err)
	}
	var pr struct {
		Number int
		Checks []json.RawMessage `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal(data, &pr); err != nil || pr.Number != 4 || len(pr.Checks) != 2 {
		t.Fatalf("data=%s err=%v", data, err)
	}
	want := []graphQLVariables{{Owner: "Owner", Repo: "Repo", Number: 4}, {Owner: "Owner", Repo: "Repo", Number: 4, Cursor: "next"}}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("variables=%+v want=%+v", *calls, want)
	}
}
func TestHTTPPRRetriesFromFirstPage(t *testing.T) {
	first := graphPage("abc", "abc", "next", true, `[{"__typename":"CheckRun","name":"test"}]`)
	last := graphPage("abc", "abc", "", false, "[]")
	// The second page fails with a truncated body. The HTTP status still makes it retryable.
	failed := "HTTP/1.1 503 Service Unavailable\r\nContent-Length: 80\r\n\r\ntruncated"
	client, calls := prTestClient(t, first, failed, first, last)
	waits := 0
	policy := retryPolicy{
		delay: func(int) time.Duration { return 0 },
		wait: func(context.Context, time.Duration) error {
			waits++
			return nil
		},
	}
	data, attempts, err := retryPR(context.Background(), client, "Owner/Repo", 4, policy)
	if err != nil || attempts != 2 || waits != 1 {
		t.Fatalf("attempts=%d waits=%d err=%v", attempts, waits, err)
	}
	var pr struct {
		Checks []json.RawMessage `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal(data, &pr); err != nil || len(pr.Checks) != 1 {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for i, cursor := range []string{"", "next", "", "next"} {
		if len(*calls) != 4 || (*calls)[i].Cursor != cursor {
			t.Fatalf("unexpected GraphQL calls: %+v", *calls)
		}
	}
}

func TestHTTPPRRejectsIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		pages []string
		want  string
	}{
		{"missing initial commit", []string{"HTTP/2.0 200 OK\r\n\r\n" + `{"data":{"repository":{"pullRequest":{"headRefOid":"a","commits":{"nodes":[]}}}}}`}, "invalid_metadata"},
		{"missing continuation commit", []string{graphPage("a", "a", "next", true, "[]"), strings.Replace(graphPage("a", "a", "", false, "[]"), `"nodes":[{"commit":{"oid":"a","statusCheckRollup":{"contexts":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}]`, `"nodes":[]`, 1)}, "invalid_metadata"},
		{"missing continuation rollup", []string{graphPage("a", "a", "next", true, "[]"), strings.Replace(graphPage("a", "a", "", false, "[]"), `"statusCheckRollup":{"contexts":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}`, `"statusCheckRollup":null`, 1)}, "invalid_pagination"},
		{"missing pageInfo", []string{strings.Replace(graphPage("a", "a", "", false, "[]"), `,"pageInfo":{"hasNextPage":false,"endCursor":""}`, "", 1)}, "invalid_pagination"},
		{"null pageInfo", []string{strings.Replace(graphPage("a", "a", "", false, "[]"), `"pageInfo":{"hasNextPage":false,"endCursor":""}`, `"pageInfo":null`, 1)}, "invalid_pagination"},
		{"missing hasNextPage", []string{strings.Replace(graphPage("a", "a", "next", true, "[]"), `"hasNextPage":true,`, "", 1)}, "invalid_pagination"},
		{"null hasNextPage", []string{strings.Replace(graphPage("a", "a", "next", true, "[]"), `"hasNextPage":true`, `"hasNextPage":null`, 1)}, "invalid_pagination"},
		{"missing check nodes", []string{strings.Replace(graphPage("a", "a", "", false, "[]"), `"contexts":{"nodes":[],`, `"contexts":{`, 1)}, "invalid_metadata"},
		{"null check nodes", []string{graphPage("a", "a", "", false, "null")}, "invalid_metadata"},
		{"unknown check type", []string{graphPage("a", "a", "", false, `[{"__typename":"Unknown","context":"secret"}]`)}, "invalid_metadata"},
		{"missing check type", []string{graphPage("a", "a", "", false, `[{"context":"secret"}]`)}, "invalid_metadata"},
		{"null check", []string{graphPage("a", "a", "", false, `[null]`)}, "invalid_metadata"},
		{"head changed", []string{graphPage("a", "a", "next", true, "[]"), graphPage("b", "b", "", false, "[]")}, "head_changed"},
		{"different commit", []string{graphPage("a", "b", "", false, "[]")}, "head_changed"},
		{"repeated cursor", []string{graphPage("a", "a", "next", true, "[]"), graphPage("a", "a", "next", true, "[]")}, "pagination"},
		{"missing cursor", []string{graphPage("a", "a", "", true, "[]")}, "pagination"},
		{"graphql errors", []string{"HTTP/2.0 200 OK\r\n\r\n" + `{"data":{"repository":null},"errors":[{"type":"FORBIDDEN","message":"secret"}]}`}, "graphql_error"},
		{"rate limit", []string{"HTTP/2.0 200 OK\r\nRetry-After: 5\r\n\r\n" + `{"errors":[{"type":"RATE_LIMITED","message":"secret"}]}`}, "rate_limit"},
		{"missing response", []string{"HTTP/2.0 200 OK\r\n\r\n" + `{"data":{"repository":null}}`}, "not_found_or_inaccessible"},
		{"no headers", []string{"secret"}, "GitHub PR request failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := prTestClient(t, tc.pages...)
			data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
			if data != nil || err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "secret") {
				t.Fatalf("data=%s err=%v", data, err)
			}
		})
	}
}

func TestHTTPPRClassifiesHTTPFailures(t *testing.T) {
	for _, tc := range []struct {
		name, response, code string
		retryable            bool
	}{
		{"server", "HTTP/2.0 503 Service Unavailable\r\n\r\nsecret body", "http_5xx", true},
		{"rate limit", "HTTP/2.0 429 Too Many Requests\r\nRetry-After: 5\r\n\r\nsecret body", "rate_limit", true},
		{"secondary limit", "HTTP/2.0 403 Forbidden\r\nX-RateLimit-Remaining: 42\r\n\r\n" + `{"message":"You have exceeded a secondary rate limit. secret"}`, "rate_limit", false},
		{"graphql secondary message", "HTTP/2.0 403 Forbidden\r\nX-RateLimit-Remaining: 42\r\n\r\n" + `{"errors":[{"message":"You have exceeded a secondary rate limit. secret"}]}`, "rate_limit", false},
		{"typed limit", "HTTP/2.0 403 Forbidden\r\nX-RateLimit-Remaining: 42\r\n\r\n" + `{"errors":[{"type":"RATE_LIMITED","message":"secret"}]}`, "rate_limit", false},
		{"structured permission", "HTTP/2.0 403 Forbidden\r\nX-RateLimit-Remaining: 42\r\n\r\n" + `{"message":"Resource not accessible by integration. secret"}`, "permission_denied", false},
		{"permission", "HTTP/2.0 403 Forbidden\r\n\r\nsecret body", "permission_denied", false},
		{"authentication", "HTTP/2.0 401 Unauthorized\r\n\r\nsecret body", "authentication", false},
		{"redirect", "HTTP/1.1 302 Found\r\nLocation: https://example.com/\r\n\r\n", "http_error", false},
		{"truncated rate limit", "HTTP/1.1 429 Too Many Requests\r\nRetry-After: 5\r\nContent-Length: 30\r\n\r\nsecret body", "rate_limit", true},
		{"truncated success", "HTTP/1.1 200 OK\r\nContent-Length: 30\r\n\r\n{}", "decode_error", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := prTestClient(t, tc.response)
			data, err := client.FetchPR(context.Background(), "owner/repo", 4)
			if err == nil || data != nil {
				t.Fatalf("data=%s err=%v", data, err)
			}
			code, retryable, wait := requestFailure(err)
			if code != tc.code || retryable != tc.retryable || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "sensitive") {
				t.Fatalf("code=%s retryable=%t error=%v", code, retryable, err)
			}
			if tc.name == "rate limit" && wait != 5*time.Second {
				t.Fatalf("wait=%s", wait)
			}
		})
	}
}

func TestHTTPPRCancellation(t *testing.T) {
	t.Setenv("GH_TOKEN", "fixture-token")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := newGitHubHTTPClient()
	client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
		cancel()
		return nil, req.Context().Err()
	})
	data, err := client.FetchPR(ctx, "owner/repo", 4)
	if data != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("data=%s err=%v", data, err)
	}
}

func TestHTTPPRCheckCompatibility(t *testing.T) {
	pages := []string{graphPage("a", "a", "", false, `[{"__typename":"CheckRun","name":"test","checkSuite":{"workflowRun":{"workflow":{"name":"CI"}}},"status":"COMPLETED","conclusion":"SUCCESS","startedAt":"2026-11-25T00:00:00Z","completedAt":"2026-11-25T00:01:00Z","detailsUrl":"https://example.com/check"},{"__typename":"StatusContext","context":"external","state":"SUCCESS","targetUrl":"https://example.com/status","createdAt":"2026-11-25T00:00:00Z","description":"passed"}]`)}
	client, _ := prTestClient(t, pages...)
	data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
	if err != nil {
		t.Fatal(err)
	}
	var pr map[string]json.RawMessage
	if err := json.Unmarshal(data, &pr); err != nil {
		t.Fatal(err)
	}
	var got any
	json.Unmarshal(pr["statusCheckRollup"], &got)
	var want any
	json.Unmarshal([]byte(`[{"__typename":"CheckRun","name":"test","workflowName":"CI","status":"COMPLETED","conclusion":"SUCCESS","startedAt":"2026-11-25T00:00:00Z","completedAt":"2026-11-25T00:01:00Z","detailsUrl":"https://example.com/check"},{"__typename":"StatusContext","context":"external","state":"SUCCESS","targetUrl":"https://example.com/status","startedAt":"2026-11-25T00:00:00Z"}]`), &want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("checks=%s", pr["statusCheckRollup"])
	}
}

func TestHTTPPRPendingCheckTimes(t *testing.T) {
	for _, tc := range []struct {
		name, status, started, wantStarted string
	}{
		{"queued", "QUEUED", "null", "0001-01-01T00:00:00Z"},
		{"running", "IN_PROGRESS", `"2026-11-25T00:00:00Z"`, "2026-11-25T00:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks := `[{"__typename":"CheckRun","name":"test","status":"` + tc.status + `","startedAt":` + tc.started + `,"completedAt":null,"conclusion":null,"checkSuite":{"workflowRun":null}}]`
			client, _ := prTestClient(t, graphPage("a", "a", "", false, checks))
			data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
			if err != nil {
				t.Fatal(err)
			}
			var pr struct {
				Checks []map[string]any `json:"statusCheckRollup"`
			}
			if err := json.Unmarshal(data, &pr); err != nil {
				t.Fatal(err)
			}
			want := []map[string]any{{"__typename": "CheckRun", "name": "test", "workflowName": "", "status": tc.status, "conclusion": "", "startedAt": tc.wantStarted, "completedAt": "0001-01-01T00:00:00Z", "detailsUrl": ""}}
			if !reflect.DeepEqual(pr.Checks, want) {
				t.Fatalf("checks=%v want=%v", pr.Checks, want)
			}
		})
	}
}

func TestHTTPPRForbiddenPartialCollection(t *testing.T) {
	for _, tc := range []struct {
		name, status, body, code       string
		wantPRs, wantCalls, wantErrors int
	}{
		{"secondary limit", "403 Forbidden", `{"message":"You have exceeded a secondary rate limit. secret"}`, "rate_limit", 1, 2, 2},
		{"HTTP 200 secondary limit", "200 OK", `{"errors":[{"message":"You have exceeded a secondary rate limit. secret"}]}`, "rate_limit", 1, 2, 2},
		{"permission", "403 Forbidden", `{"message":"Resource not accessible by integration. secret"}`, "permission_denied", 2, 3, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := prTestClient(t, "HTTP/2.0 "+tc.status+"\r\nX-RateLimit-Remaining: 42\r\n\r\n"+tc.body)
			calls := 0
			collector := fakeGitHubClient{prFn: func(ctx context.Context, repo string, n int) ([]byte, error) {
				calls++
				if n == 2 {
					return client.FetchPR(ctx, repo, n)
				}
				return []byte(`{"number":` + strconv.Itoa(n) + `,"url":"https://github.com/owner/repo/pull/` + strconv.Itoa(n) + `","state":"OPEN","headRefOid":"a","updatedAt":"time"}`), nil
			}}
			prs, r, err := collectPullRequests(context.Background(), collector, "owner/repo", []dashpkg.Item{{PRLink: "pull/1"}, {PRLink: "pull/2"}, {PRLink: "pull/3"}}, true, defaultRetryPolicy())
			if err != nil || len(prs) != tc.wantPRs || prs[0].Number != 1 || calls != tc.wantCalls || r.Status != "partial" || len(r.Errors) != tc.wantErrors {
				t.Fatalf("prs=%+v retrieval=%+v calls=%d err=%v", prs, r, calls, err)
			}
			for i, e := range r.Errors {
				if e.Number != i+2 || e.Code != tc.code || e.Attempts != 1-i || e.Retryable {
					t.Fatalf("error=%+v", e)
				}
			}
			encoded, _ := json.Marshal(r)
			if strings.Contains(string(encoded), "secret") {
				t.Fatal(string(encoded))
			}
		})
	}
}

func TestHTTPPRInitialNullRollup(t *testing.T) {
	page := strings.Replace(graphPage("a", "a", "", false, "[]"), `"statusCheckRollup":{"contexts":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}`, `"statusCheckRollup":null`, 1)
	client, _ := prTestClient(t, page)
	data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
	if err != nil {
		t.Fatal(err)
	}
	var pr map[string]json.RawMessage
	if err := json.Unmarshal(data, &pr); err != nil || string(pr["statusCheckRollup"]) != "[]" {
		t.Fatalf("data=%s err=%v", data, err)
	}
}

func TestHTTPPRSecondaryRateLimit(t *testing.T) {
	for _, header := range []string{"", "Retry-After: 5\r\n"} {
		t.Run(header, func(t *testing.T) {
			client, _ := prTestClient(t, "HTTP/2.0 200 OK\r\n"+header+"\r\n"+`{"errors":[{"message":"You have exceeded a secondary rate limit. secret"}]}`)
			data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
			code, retryable, wait := requestFailure(err)
			wantWait := time.Duration(0)
			if header != "" {
				wantWait = 5 * time.Second
			}
			if data != nil || code != "rate_limit" || retryable != (header != "") || wait != wantWait || strings.Contains(err.Error(), "secret") {
				t.Fatalf("data=%s code=%s retryable=%t wait=%s err=%v", data, code, retryable, wait, err)
			}
		})
	}
}

func TestHTTPPRCommitErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name, fields, code string
	}{
		{"missing", "", "invalid_metadata"},
		{"null", `,"commits":null`, "invalid_metadata"},
		{"empty", `,"commits":{"nodes":[]}`, "invalid_metadata"},
		{"wrong type", `,"commits":"secret"`, "decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := "HTTP/2.0 200 OK\r\n\r\n" + `{"data":{"repository":{"pullRequest":{"headRefOid":"a"` + tc.fields + `}}}}`
			client, _ := prTestClient(t, page)
			data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
			code, retryable, _ := requestFailure(err)
			if data != nil || err == nil || code != tc.code || retryable || strings.Contains(err.Error(), "secret") {
				t.Fatalf("data=%s code=%s retryable=%t err=%v", data, code, retryable, err)
			}
		})
	}
}

func TestHTTPPRRejectsMissingCommitOID(t *testing.T) {
	for _, tc := range []struct {
		name, commit string
	}{
		{"null commit", "null"},
		{"missing OID", "{}"},
		{"empty OID", `{"oid":""}`},
		{"null OID", `{"oid":null}`},
	} {
		for _, continuation := range []bool{false, true} {
			t.Run(tc.name+"/"+strconv.FormatBool(continuation), func(t *testing.T) {
				page := "HTTP/2.0 200 OK\r\n\r\n" + `{"data":{"repository":{"pullRequest":{"headRefOid":"a","commits":{"nodes":[{"commit":` + tc.commit + `}]}}}}}`
				pages := []string{page}
				if continuation {
					pages = append([]string{graphPage("a", "a", "next", true, "[]")}, pages...)
				}
				client, _ := prTestClient(t, pages...)
				data, err := client.FetchPR(context.Background(), "Owner/Repo", 4)
				code, retryable, _ := requestFailure(err)
				if data != nil || err == nil || code != "invalid_metadata" || retryable {
					t.Fatalf("data=%s code=%s retryable=%t err=%v", data, code, retryable, err)
				}
			})
		}
	}
}

func TestStructuredHTTPFailures(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		status  int
		headers string
		code    string
		retry   bool
	}{
		{200, "", "", false}, {401, "", "authentication", false}, {403, "", "permission_denied", false}, {404, "", "not_found_or_inaccessible", false}, {503, "", "http_5xx", true}, {429, "Retry-After: 5\r\n", "rate_limit", true}, {403, "X-RateLimit-Remaining: 0\r\nX-RateLimit-Reset: " + fmt.Sprint(now.Add(time.Minute).Unix()) + "\r\n", "rate_limit", true}, {429, "", "rate_limit", false}, {429, "Retry-After: nonsense\r\n", "rate_limit", false},
	} {
		t.Run(fmt.Sprint(tc.status, tc.headers), func(t *testing.T) {
			resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(fmt.Sprintf("HTTP/2.0 %d Status\r\n%s\r\n{\"secret\":true}", tc.status, tc.headers))), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			failure := httpFailure(resp.StatusCode, resp.Header, body)
			if tc.code == "" {
				if failure != nil {
					t.Fatal(failure)
				}
				return
			}
			code, retry, _ := requestFailure(failure)
			if code != tc.code || retry != tc.retry || strings.Contains(failure.Error(), "secret") {
				t.Fatalf("code=%s retry=%t err=%v", code, retry, failure)
			}
		})
	}
	h := http.Header{"Retry-After": []string{now.Add(10 * time.Second).UTC().Format(http.TimeFormat)}}
	if wait, ok := rateLimitWait(h, now); !ok || wait < 9*time.Second || wait > 10*time.Second {
		t.Fatalf("wait=%s ok=%t", wait, ok)
	}
}

type stalledPRBody struct {
	ctx    context.Context
	read   bool
	closed bool
}

func (b *stalledPRBody) Read([]byte) (int, error) {
	b.read = true
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *stalledPRBody) Close() error {
	b.closed = true
	return nil
}

func TestHTTPPRRateLimitDoesNotWaitForBody(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		header    http.Header
		wantRetry bool
	}{
		{"429", http.StatusTooManyRequests, http.Header{"Retry-After": {"5"}}, true},
		{"403", http.StatusForbidden, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GH_TOKEN", "fixture-token")
			client := newGitHubHTTPClient()
			var body *stalledPRBody
			client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
				body = &stalledPRBody{ctx: req.Context()}
				return &http.Response{StatusCode: tc.status, Header: tc.header, Body: body}, nil
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			data, err := client.FetchPR(ctx, "owner/repo", 4)
			code, retry, wait := requestFailure(err)
			if data != nil || code != "rate_limit" || retry != tc.wantRetry || wait <= 0 || body == nil || body.read || !body.closed {
				t.Fatalf("data=%s err=%v retry=%v wait=%v body=%+v", data, err, retry, wait, body)
			}
		})
	}
}

type observedPRBody struct {
	io.ReadCloser
	started chan struct{}
}

func (b *observedPRBody) Read(p []byte) (int, error) {
	if b.started != nil {
		close(b.started)
		b.started = nil
	}
	return b.ReadCloser.Read(p)
}

func TestHTTPPRCancellationOverTLS(t *testing.T) {
	for _, stage := range []string{"headers", "body", "pagination"} {
		t.Run(stage, func(t *testing.T) {
			t.Setenv("GH_TOKEN", "fixture-token")
			started := make(chan struct{})
			wantCalls := int32(1)
			if stage == "pagination" {
				wantCalls = 2
			}
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				if call > wantCalls {
					t.Error("unexpected request after cancellation point")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var request graphQLRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Variables.Number != 4 {
					t.Errorf("unexpected GraphQL request: %+v, %v", request.Variables, err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if stage == "pagination" && call == 1 {
					_, page, _ := strings.Cut(graphPage("a", "a", "next", true, "[]"), "\r\n\r\n")
					_, _ = io.WriteString(w, page)
					return
				}
				if stage == "pagination" && request.Variables.Cursor != "next" {
					t.Errorf("unexpected continuation cursor: %q", request.Variables.Cursor)
				}
				if stage == "body" {
					if _, err := io.WriteString(w, "{"); err != nil {
						t.Error(err)
						return
					}
					if err := http.NewResponseController(w).Flush(); err != nil {
						t.Error(err)
						return
					}
				} else {
					close(started)
				}
				<-r.Context().Done()
			}))
			t.Cleanup(server.Close)
			routeGitHubServer(t, server)
			client := newGitHubHTTPClient()
			if stage == "body" {
				transport := client.http.Transport
				client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
					resp, err := transport.RoundTrip(req)
					if err == nil {
						resp.Body = &observedPRBody{ReadCloser: resp.Body, started: started}
					}
					return resp, err
				})
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			type result struct {
				prs []pullRequestMetadata
				err error
			}
			done := make(chan result, 1)
			go func() {
				prs, _, err := collectPullRequests(ctx, client, "Owner/Repo",
					[]dashpkg.Item{{PRLink: "pull/4"}, {PRLink: "pull/5"}}, true, defaultRetryPolicy())
				done <- result{prs, err}
			}()
			select {
			case <-started:
				cancel()
			case got := <-done:
				t.Fatalf("collection returned before cancellation: %+v", got)
			case <-ctx.Done():
				t.Fatal("request did not reach cancellation point")
			}
			select {
			case got := <-done:
				if got.prs != nil || !errors.Is(got.err, context.Canceled) {
					t.Fatalf("got %+v; want no metadata and context.Canceled", got)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("collection did not return after cancellation")
			}
			if got := calls.Load(); got != wantCalls {
				t.Fatalf("made %d requests, want %d", got, wantCalls)
			}
		})
	}
}

func TestHTTPPRPartialStopsAfterExhaustedRateLimit(t *testing.T) {
	success := graphPage("a", "a", "", false, "[]")
	limited := "HTTP/1.1 429 Too Many Requests\r\nRetry-After: 5\r\nContent-Length: 100\r\n\r\ntruncated"
	client, calls := prTestClient(t, success, limited, limited, limited)
	var waits []time.Duration
	policy := retryPolicy{
		delay: func(int) time.Duration { return 0 },
		wait: func(_ context.Context, duration time.Duration) error {
			waits = append(waits, duration)
			return nil
		},
	}
	prs, retrieval, err := collectPullRequests(context.Background(), client, "Owner/Repo",
		[]dashpkg.Item{{PRLink: "pull/4"}, {PRLink: "pull/5"}, {PRLink: "pull/6"}}, true, policy)
	if err != nil || len(prs) != 1 || prs[0].Number != 4 || retrieval.Status != "partial" {
		t.Fatalf("prs=%+v retrieval=%+v err=%v", prs, retrieval, err)
	}
	want := []retrievalError{
		{Resource: "pull_request", Repository: "Owner/Repo", Code: "rate_limit", Number: 5, Attempts: 3, Retryable: true},
		{Resource: "pull_request", Repository: "Owner/Repo", Code: "rate_limit", Number: 6, Attempts: 0, Retryable: true},
	}
	if !reflect.DeepEqual(retrieval.Errors, want) || !reflect.DeepEqual(waits, []time.Duration{5 * time.Second, 5 * time.Second}) {
		t.Fatalf("errors=%+v waits=%v", retrieval.Errors, waits)
	}
	for i, number := range []int{4, 5, 5, 5} {
		if len(*calls) != 4 || (*calls)[i].Number != number {
			t.Fatalf("unexpected GraphQL calls: %+v", *calls)
		}
	}
}
