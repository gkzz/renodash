package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const root = "../.."

var order = []int{53, 51, 30, 52}

type offlineHarness struct {
	gh, calls string
	client    *issueHTTPClient
}

func newHarness(t *testing.T) offlineHarness {
	t.Helper()
	installFakeGH(t, "offline", "")
	gh, err := exec.LookPath("gh")
	if err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(t.TempDir(), "calls")
	fixtures := filepath.Join(root, "testdata/pr-metadata")
	t.Setenv("OFFLINE_CALLS", calls)
	t.Setenv("OFFLINE_FIXTURES", fixtures)
	t.Setenv("GH_TOKEN", "fixture-token")
	t.Setenv("GITHUB_TOKEN", "")
	issue, err := os.ReadFile(filepath.Join(fixtures, "dashboard-issue.json"))
	if err != nil {
		t.Fatal(err)
	}
	client := newIssueHTTPClient()
	client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != "https://api.github.com/repos/gkzz/renodash/issues/5" {
			return nil, fmt.Errorf("unexpected offline Issue request")
		}
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("unexpected Issue token")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(issue))}, nil
	})
	return offlineHarness{gh, calls, client}
}

func (h offlineHarness) run(t *testing.T, mode string, stdin io.Reader, args ...string) ([]byte, string, error) {
	t.Helper()
	t.Setenv("OFFLINE_MODE", mode)
	if len(args) == 0 {
		args = []string{"get", "--with-pr-metadata", "gkzz/renodash#5"}
	}
	var stdout, stderr bytes.Buffer
	code := runWithClient(h.client, args, stdin, &stdout, &stderr)
	if code != 0 {
		return stdout.Bytes(), stderr.String(), fmt.Errorf("exit status %d", code)
	}
	return stdout.Bytes(), stderr.String(), nil
}
func decode(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func TestOfflineCapturedMetadata(t *testing.T) {
	h := newHarness(t)
	out, stderr, err := h.run(t, "", nil)
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	result := decode(t, out)
	if len(result["items"].([]any)) != 5 {
		t.Fatal("expected five candidates")
	}
	prs := result["pull_requests"].([]any)
	if len(prs) != len(order) {
		t.Fatalf("PRs: %v", prs)
	}
	states := map[string]bool{}
	var wantCalls [][]string
	for i, n := range order {
		pr := prs[i].(map[string]any)
		if pr["number"] != float64(n) {
			t.Fatalf("first-seen order changed: %v", prs)
		}
		data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("testdata/pr-metadata/pr-%d.json", n)))
		if err != nil {
			t.Fatal(err)
		}
		captured := decode(t, data)
		if pr["state"] != captured["state"] {
			t.Fatalf("captured state changed: got %v, want %v", pr["state"], captured["state"])
		}
		if diff := cmp.Diff(captured["statusCheckRollup"], pr["checks"]); diff != "" {
			t.Fatalf("captured checks mismatch (-want +got):\n%s", diff)
		}
		if pr["mergeable"] != "UNKNOWN" || pr["merge_state_status"] != "UNKNOWN" {
			t.Fatalf("historical fixture has no merge observations: %v", pr)
		}
		timestamp, ok := pr["fetched_at"].(string)
		if !ok {
			t.Fatalf("missing PR fetch time: %v", pr)
		}
		if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
			t.Fatal(err)
		}
		states[pr["state"].(string)] = true
		wantCalls = append(wantCalls, []string{"api", "graphql", "number=" + fmt.Sprint(n)})
	}
	if diff := cmp.Diff(map[string]bool{"OPEN": true, "CLOSED": true, "MERGED": true}, states); diff != "" {
		t.Fatalf("states mismatch (-want +got):\n%s", diff)
	}
	calls, err := os.ReadFile(h.calls)
	if err != nil {
		t.Fatal(err)
	}
	var gotCalls [][]string
	for _, line := range bytes.Split(bytes.TrimSpace(calls), []byte("\n")) {
		var args []string
		if err = json.Unmarshal(line, &args); err != nil {
			t.Fatal(err)
		}
		if len(args) == 13 && args[1] == "graphql" {
			args = []string{args[0], args[1], args[12]}
		}
		gotCalls = append(gotCalls, args)
	}
	if diff := cmp.Diff(wantCalls, gotCalls); diff != "" {
		t.Fatalf("lookups mismatch (-want +got):\n%s", diff)
	}
	if path := os.Getenv("RENODASH_OUTPUT"); path != "" {
		if err = os.WriteFile(path, out, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func TestOfflineLookupErrors(t *testing.T) {
	for mode, message := range map[string]string{"failure": "fetch PR", "malformed": "decode_error", "missing": "head_changed"} {
		t.Run(mode, func(t *testing.T) {
			out, stderr, err := newHarness(t).run(t, mode, nil)
			if err == nil || len(out) != 0 || !strings.Contains(stderr, "51") || !strings.Contains(stderr, message) {
				t.Fatalf("err=%v stdout=%s stderr=%s", err, out, stderr)
			}
		})
	}
}
func TestOfflineEmptyChecks(t *testing.T) {
	out, stderr, err := newHarness(t).run(t, "empty", nil)
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	checks := decode(t, out)["pull_requests"].([]any)[1].(map[string]any)["checks"]
	if values, ok := checks.([]any); !ok || len(values) != 0 {
		t.Fatal(checks)
	}
	if summary := renderSummary(t, out); !strings.Contains(summary, "Unknown (no checks returned)") {
		t.Fatal(summary)
	}
}
func TestOfflineUnknownRequests(t *testing.T) {
	for _, args := range [][]string{{"auth", "status"}, {"pr", "view", "999", "--repo", "gkzz/renodash", "--json", fields}} {
		h := newHarness(t)
		cmd := exec.Command(h.gh, args...)
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "unsupported offline gh request") {
			t.Fatalf("%v: %s", err, out)
		}
	}
}

func TestOfflineMergeObservations(t *testing.T) {
	out, stderr, err := newHarness(t).run(t, "merge", nil)
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	pr := decode(t, out)["pull_requests"].([]any)[1].(map[string]any)
	if pr["mergeable"] != "MERGEABLE" || pr["merge_state_status"] != "BLOCKED" {
		t.Fatalf("merge observations lost: %v", pr)
	}
}

func TestOfflineStdinPartial(t *testing.T) {
	expected, err := os.ReadFile(filepath.Join(root, "internal/command/testdata/renodash.json"))
	if err != nil {
		t.Fatal(err)
	}
	wantItems := decode(t, expected)["items"]
	for _, mode := range []string{"", "failure", "malformed", "all-prs-fail"} {
		t.Run(mode, func(t *testing.T) {
			h := newHarness(t)
			data, diagnostic, err := h.run(t, mode, strings.NewReader("gkzz/renodash#5\r\n"),
				"get", "--with-pr-metadata", "--allow-partial", "--stdin")
			if err != nil {
				t.Fatalf("%v: %s", err, diagnostic)
			}
			result := decode(t, data)
			r := result["retrieval"].(map[string]any)
			wantStatus, wantPRs := "complete", order
			var failedPRs []int
			wantCode := "fetch_failed"
			if mode != "" {
				wantStatus, wantPRs = "partial", []int{53, 30, 52}
				failedPRs = []int{51}
			}
			if mode == "all-prs-fail" {
				wantPRs, failedPRs = nil, order
			}
			if mode == "malformed" {
				wantCode = "decode_error"
			}
			prs := result["pull_requests"].([]any)
			errors := r["errors"].([]any)
			if r["status"] != wantStatus || len(prs) != len(wantPRs) || len(errors) != len(failedPRs) {
				t.Fatalf("result=%s", data)
			}
			if diff := cmp.Diff(wantItems, result["items"]); diff != "" {
				t.Fatalf("Dashboard candidates changed (-want +got):\n%s", diff)
			}
			for i, number := range wantPRs {
				pr := prs[i].(map[string]any)
				data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("testdata/pr-metadata/pr-%d.json", number)))
				if err != nil {
					t.Fatal(err)
				}
				captured := decode(t, data)
				if pr["number"] != float64(number) || pr["head_sha"] != captured["headRefOid"] {
					t.Fatalf("unexpected PR at index %d: %v", i, pr)
				}
				if diff := cmp.Diff(captured["statusCheckRollup"], pr["checks"]); diff != "" {
					t.Fatalf("PR #%d checks changed (-want +got):\n%s", number, diff)
				}
			}
			for i, entry := range errors {
				e := entry.(map[string]any)
				if e["resource"] != "pull_request" || e["repository"] != "gkzz/renodash" ||
					e["code"] != wantCode || e["number"] != float64(failedPRs[i]) ||
					e["attempts"] != float64(1) || e["retryable"] != false {
					t.Fatalf("error=%v", e)
				}
			}
		})
	}
}

func TestOfflineSharedCredentials(t *testing.T) {
	for _, tc := range []struct{ name, gh, github string }{
		{"secondary token", " ", " fixture-token "},
		{"saved login", " ", "\t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			t.Setenv("GH_TOKEN", tc.gh)
			t.Setenv("GITHUB_TOKEN", tc.github)
			data, diagnostic, err := h.run(t, "", nil)
			if err != nil || diagnostic != "" {
				t.Fatalf("err=%v stderr=%q", err, diagnostic)
			}
			if got := len(decode(t, data)["pull_requests"].([]any)); got != len(order) {
				t.Fatalf("got %d PRs, want %d", got, len(order))
			}
		})
	}
}
