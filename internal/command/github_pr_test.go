package command

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

func graphPage(head, commit, cursor string, more bool, checks string) string {
	page := `{"data":{"repository":{"pullRequest":{"number":4,"url":"https://github.com/Owner/Repo/pull/4","state":"OPEN","title":"Update","headRefOid":"` + head + `","updatedAt":"2026-11-25T00:00:00Z","commits":{"nodes":[{"commit":{"oid":"` + commit + `","statusCheckRollup":{"contexts":{"nodes":` + checks + `,"pageInfo":{"hasNextPage":` + map[bool]string{true: "true", false: "false"}[more] + `,"endCursor":"` + cursor + `"}}}}}]}}}}}`
	return "HTTP/2.0 200 OK\r\nContent-Type: application/json\r\n\r\n" + page
}
func TestGHPRPaginationAndArguments(t *testing.T) {
	pages := []string{graphPage("abc", "abc", "next", true, `[{"__typename":"CheckRun","name":"test"}]`), graphPage("abc", "abc", "", false, `[{"__typename":"StatusContext","context":"external"}]`)}
	payload, _ := json.Marshal(pages)
	installFakeGH(t, "pages", string(payload))
	path := filepath.Join(t.TempDir(), "calls.jsonl")
	t.Setenv("COMMAND_FAKE_GH_LOG", path)
	data, err := (ghClient{}).FetchPR(context.Background(), "Owner/Repo", 4)
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
	log, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	calls := strings.Split(strings.TrimSpace(string(log)), "\n")
	if len(calls) != 2 {
		t.Fatal(string(log))
	}
	for i, line := range calls {
		var args []string
		if err := json.Unmarshal([]byte(line), &args); err != nil {
			t.Fatal(err)
		}
		want := []string{"api", "graphql", "--hostname", "github.com", "--include", "-f", "query=" + prQuery, "-f", "owner=Owner", "-f", "repo=Repo", "-F", "number=4"}
		if i == 1 {
			want = append(want, "-f", "cursor=next")
		}
		if !slices.Equal(args, want) {
			t.Fatalf("args=%q want=%q", args, want)
		}
	}
}
func TestGHPRRejectsIncompleteEvidence(t *testing.T) {
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
		{"head changed", []string{graphPage("a", "a", "next", true, "[]"), graphPage("b", "b", "", false, "[]")}, "head_changed"},
		{"different commit", []string{graphPage("a", "b", "", false, "[]")}, "head_changed"},
		{"repeated cursor", []string{graphPage("a", "a", "next", true, "[]"), graphPage("a", "a", "next", true, "[]")}, "pagination"},
		{"missing cursor", []string{graphPage("a", "a", "", true, "[]")}, "pagination"},
		{"graphql errors", []string{"HTTP/2.0 200 OK\r\n\r\n" + `{"data":{"repository":null},"errors":[{"type":"FORBIDDEN","message":"secret"}]}`}, "graphql_error"},
		{"rate limit", []string{"HTTP/2.0 200 OK\r\nRetry-After: 5\r\n\r\n" + `{"errors":[{"type":"RATE_LIMITED","message":"secret"}]}`}, "rate_limit"},
		{"missing response", []string{"HTTP/2.0 200 OK\r\n\r\n" + `{"data":{"repository":null}}`}, "not_found_or_inaccessible"},
		{"no headers", []string{"secret"}, "HTTP status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(tc.pages)
			installFakeGH(t, "pages", string(payload))
			data, err := (ghClient{}).FetchPR(context.Background(), "Owner/Repo", 4)
			if data != nil || err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "secret") {
				t.Fatalf("data=%s err=%v", data, err)
			}
		})
	}
}

func TestGHPRClassifiesFailedProcessHTTP(t *testing.T) {
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			installFakeGH(t, "http-failure", tc.response)
			data, err := (ghClient{}).FetchPR(context.Background(), "owner/repo", 4)
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

func TestGHPRCheckCompatibility(t *testing.T) {
	pages := []string{graphPage("a", "a", "", false, `[{"__typename":"CheckRun","name":"test","checkSuite":{"workflowRun":{"workflow":{"name":"CI"}}},"status":"COMPLETED","conclusion":"SUCCESS","startedAt":"2026-11-25T00:00:00Z","completedAt":"2026-11-25T00:01:00Z","detailsUrl":"https://example.com/check"},{"__typename":"StatusContext","context":"external","state":"SUCCESS","targetUrl":"https://example.com/status","createdAt":"2026-11-25T00:00:00Z","description":"passed"}]`)}
	payload, _ := json.Marshal(pages)
	installFakeGH(t, "pages", string(payload))
	data, err := (ghClient{}).FetchPR(context.Background(), "Owner/Repo", 4)
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

func TestGHPRPendingCheckTimes(t *testing.T) {
	for _, tc := range []struct {
		name, status, started, wantStarted string
	}{
		{"queued", "QUEUED", "null", "0001-01-01T00:00:00Z"},
		{"running", "IN_PROGRESS", `"2026-11-25T00:00:00Z"`, "2026-11-25T00:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks := `[{"__typename":"CheckRun","name":"test","status":"` + tc.status + `","startedAt":` + tc.started + `,"completedAt":null,"conclusion":null,"checkSuite":{"workflowRun":null}}]`
			payload, err := json.Marshal([]string{graphPage("a", "a", "", false, checks)})
			if err != nil {
				t.Fatal(err)
			}
			installFakeGH(t, "pages", string(payload))
			data, err := (ghClient{}).FetchPR(context.Background(), "Owner/Repo", 4)
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

func TestGHPRForbiddenPartialCollection(t *testing.T) {
	for _, tc := range []struct {
		name, status, body, code       string
		wantPRs, wantCalls, wantErrors int
	}{
		{"secondary limit", "403 Forbidden", `{"message":"You have exceeded a secondary rate limit. secret"}`, "rate_limit", 1, 2, 2},
		{"HTTP 200 secondary limit", "200 OK", `{"errors":[{"message":"You have exceeded a secondary rate limit. secret"}]}`, "rate_limit", 1, 2, 2},
		{"permission", "403 Forbidden", `{"message":"Resource not accessible by integration. secret"}`, "permission_denied", 2, 3, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installFakeGH(t, "http-failure", "HTTP/2.0 "+tc.status+"\r\nX-RateLimit-Remaining: 42\r\n\r\n"+tc.body)
			calls := 0
			client := fakeGitHubClient{prFn: func(ctx context.Context, repo string, n int) ([]byte, error) {
				calls++
				if n == 2 {
					return (ghClient{}).FetchPR(ctx, repo, n)
				}
				return []byte(`{"number":` + strconv.Itoa(n) + `,"url":"https://github.com/owner/repo/pull/` + strconv.Itoa(n) + `","state":"OPEN","headRefOid":"a","updatedAt":"time"}`), nil
			}}
			prs, r, err := collectPullRequests(context.Background(), client, "owner/repo", []dashpkg.Item{{PRLink: "pull/1"}, {PRLink: "pull/2"}, {PRLink: "pull/3"}}, true, defaultRetryPolicy())
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

func TestGHPRInitialNullRollup(t *testing.T) {
	page := strings.Replace(graphPage("a", "a", "", false, "[]"), `"statusCheckRollup":{"contexts":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}`, `"statusCheckRollup":null`, 1)
	installFakeGH(t, "output", page)
	data, err := (ghClient{}).FetchPR(context.Background(), "Owner/Repo", 4)
	if err != nil {
		t.Fatal(err)
	}
	var pr map[string]json.RawMessage
	if err := json.Unmarshal(data, &pr); err != nil || string(pr["statusCheckRollup"]) != "[]" {
		t.Fatalf("data=%s err=%v", data, err)
	}
}

func TestGHPRSecondaryRateLimit(t *testing.T) {
	for _, header := range []string{"", "Retry-After: 5\r\n"} {
		t.Run(header, func(t *testing.T) {
			installFakeGH(t, "output", "HTTP/2.0 200 OK\r\n"+header+"\r\n"+`{"errors":[{"message":"You have exceeded a secondary rate limit. secret"}]}`)
			data, err := (ghClient{}).FetchPR(context.Background(), "Owner/Repo", 4)
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
