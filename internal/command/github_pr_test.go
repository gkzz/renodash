package command

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
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
