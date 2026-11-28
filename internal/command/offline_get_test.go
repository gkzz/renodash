package command

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const root = "../.."

var order = []int{53, 51, 30, 52}

type offlineHarness struct {
	calls  string
	client *githubHTTPClient
}

func newHarness(t *testing.T) offlineHarness {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("OFFLINE_CALLS", calls)
	t.Setenv("GH_TOKEN", "fixture-token")
	t.Setenv("GITHUB_TOKEN", "")
	handler := offlineGitHubHandler(t)
	client := newGitHubHTTPClient()
	client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		if req.Body != nil {
			defer req.Body.Close()
		}
		handler.ServeHTTP(response, req)
		return response.Result(), nil
	})
	return offlineHarness{calls, client}
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
	}
	if diff := cmp.Diff(map[string]bool{"OPEN": true, "CLOSED": true, "MERGED": true}, states); diff != "" {
		t.Fatalf("states mismatch (-want +got):\n%s", diff)
	}
	calls, err := os.ReadFile(h.calls)
	if err != nil {
		t.Fatal(err)
	}
	var gotCalls []int
	for _, line := range bytes.Split(bytes.TrimSpace(calls), []byte("\n")) {
		var number int
		if err = json.Unmarshal(line, &number); err != nil {
			t.Fatal(err)
		}
		gotCalls = append(gotCalls, number)
	}
	if diff := cmp.Diff(order, gotCalls); diff != "" {
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
			wantCode := "not_found_or_inaccessible"
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

func offlineGitHubHandler(t *testing.T) http.Handler {
	t.Helper()
	issue, err := os.ReadFile(filepath.Join(root, "testdata/pr-metadata/dashboard-issue.json"))
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Host != "api.github.com" || req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("unexpected offline GitHub host or credentials")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch req.URL.Path {
		case "/graphql":
			if err := offlinePR(w, req); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
			}
		case "/repos/gkzz/renodash/issues/5":
			if req.Method != http.MethodGet {
				t.Error("unexpected Issue request method")
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			_, _ = w.Write(issue)
		default:
			t.Error("unexpected offline GitHub request")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func serveOfflineGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(offlineGitHubHandler(t))
	t.Cleanup(server.Close)
	return server
}

func TestRunWithOfflineSharedCredentials(t *testing.T) {
	routeIssueServer(t, serveOfflineGitHub(t))
	for _, tc := range []struct{ name, gh, github string }{
		{"secondary token", " ", " fixture-token "},
		{"saved login", " ", "\t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installFakeGH(t, "auth-token", "fixture-token")
			t.Setenv("OFFLINE_MODE", "")
			t.Setenv("GH_TOKEN", tc.gh)
			t.Setenv("GITHUB_TOKEN", tc.github)
			var stdout, stderr bytes.Buffer
			code := Run([]string{"get", "--with-pr-metadata", "gkzz/renodash#5"}, nil, &stdout, &stderr)
			if code != 0 || stderr.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr.String())
			}
			if got := len(decode(t, stdout.Bytes())["pull_requests"].([]any)); got != len(order) {
				t.Fatalf("got %d PRs, want %d", got, len(order))
			}
		})
	}
}

func TestOfflineCLIFetchesIssueAndPRs(t *testing.T) {
	dir := t.TempDir()
	repo, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := json.Marshal(map[string]map[string]string{
		"Replace": {filepath.Join(repo, "offline_cli_transport.go"): filepath.Join(repo, "testdata/offline-cli-transport.go")},
	})
	if err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "renodash")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-overlay", overlayPath, "-o", binary, ".")
	build.Dir = repo
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build offline CLI: %v: %s", err, output)
	}

	server := serveOfflineGitHub(t)
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	certPath := filepath.Join(dir, "issue-cert.pem")
	if err := os.WriteFile(certPath, cert, 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir)
	t.Setenv("OFFLINE_ISSUE_CERT", certPath)
	t.Setenv("OFFLINE_ISSUE_ADDR", server.Listener.Addr().String())
	t.Setenv("GH_TOKEN", " ")
	t.Setenv("GITHUB_TOKEN", " fixture-token ")

	cmd := exec.Command(binary, "get", "--with-pr-metadata", "gkzz/renodash#5")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil || stderr.Len() != 0 {
		t.Fatalf("offline CLI: %v: %s", err, &stderr)
	}
	result := decode(t, output)
	source := result["source"].(map[string]any)
	if source["repository"] != "gkzz/renodash" || source["issue_number"] != float64(5) {
		t.Fatalf("unexpected Issue source: %v", source)
	}
	prs := result["pull_requests"].([]any)
	if len(prs) != len(order) {
		t.Fatalf("got %d PRs, want %d", len(prs), len(order))
	}
	for i, pr := range prs {
		if got := pr.(map[string]any)["number"]; got != float64(order[i]) {
			t.Fatalf("PR at index %d: got %v, want %d", i, got, order[i])
		}
	}
}
