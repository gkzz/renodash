// Package integration tests the CLI through a fixture-only gh process adapter.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

var root, binary string
var order = []int{53, 51, 30, 52}

func TestMain(m *testing.M) {
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "gh" {
		if err := serveGH(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	var err error
	root, err = filepath.Abs("../..")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Actions supplies the binary installed by go install .; ordinary go test
	// builds the current source once for the same process-level integration tests.
	binary = os.Getenv("RENODASH_BINARY")
	cleanup := func() {}
	if binary == "" {
		dir, err := os.MkdirTemp("", "renodash-integration-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cleanup = func() { os.RemoveAll(dir) }
		name := "renodash"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		binary = filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", binary, ".")
		cmd.Dir = root
		if data, err := cmd.CombinedOutput(); err != nil {
			cleanup()
			fmt.Fprintf(os.Stderr, "build CLI: %v\n%s", err, data)
			os.Exit(1)
		}
	} else {
		binary, err = filepath.Abs(binary)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}

type harness struct {
	gh    string
	env   []string
	calls string
}

func newHarness(t *testing.T) harness {
	t.Helper()
	dir := t.TempDir()
	name := "gh"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	gh := filepath.Join(dir, name)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(gh, data, 0700); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(dir, "calls")
	// Do not inherit credentials, gh config, or the host's authenticated gh PATH.
	return harness{gh, []string{"PATH=" + dir, "HOME=" + dir, "OFFLINE_CALLS=" + calls,
		"OFFLINE_FIXTURES=" + filepath.Join(root, "testdata/pr-metadata")}, calls}
}
func (h harness) run(mode string) ([]byte, string, error) {
	cmd := exec.Command(binary, "get", "--with-pr-metadata", "gkzz/renodash#5")
	cmd.Env = append(append([]string{}, h.env...), "OFFLINE_MODE="+mode)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	return out, stderr.String(), err
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
	out, stderr, err := h.run("")
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
	wantCalls := [][]string{{"api", "repos/gkzz/renodash/issues/5"}}
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
		if pr["state"] != captured["state"] || !reflect.DeepEqual(pr["checks"], captured["statusCheckRollup"]) {
			t.Fatalf("captured metadata changed: %v", pr)
		}
		states[pr["state"].(string)] = true
		wantCalls = append(wantCalls, []string{"pr", "view", fmt.Sprint(n), "--repo", "gkzz/renodash", "--json", fields})
	}
	if !reflect.DeepEqual(states, map[string]bool{"OPEN": true, "CLOSED": true, "MERGED": true}) {
		t.Fatal(states)
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
		gotCalls = append(gotCalls, args)
	}
	if !reflect.DeepEqual(gotCalls, wantCalls) {
		t.Fatalf("duplicate or unexpected lookups: %v", gotCalls)
	}
	if path := os.Getenv("RENODASH_OUTPUT"); path != "" {
		if err = os.WriteFile(path, out, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func TestOfflineLookupErrors(t *testing.T) {
	for mode, message := range map[string]string{"failure": "fetch PR", "malformed": "decode PR", "missing": "missing or mismatched metadata"} {
		t.Run(mode, func(t *testing.T) {
			out, stderr, err := newHarness(t).run(mode)
			if err == nil || len(out) != 0 || !strings.Contains(stderr, "51") || !strings.Contains(stderr, message) {
				t.Fatalf("err=%v stdout=%s stderr=%s", err, out, stderr)
			}
		})
	}
}
func TestOfflineEmptyChecks(t *testing.T) {
	out, stderr, err := newHarness(t).run("empty")
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
		cmd.Env = h.env
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "unsupported offline gh request") {
			t.Fatalf("%v: %s", err, out)
		}
	}
}
