package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkzz/renodash/internal/summary"
)

func renderSummary(t *testing.T, data []byte) string {
	t.Helper()
	var out bytes.Buffer
	if err := summary.RenderOffline(bytes.NewReader(data), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func summaryData(t *testing.T, items, prs []any) []byte {
	t.Helper()
	if items == nil {
		items = []any{}
	}
	if prs == nil {
		prs = []any{}
	}
	data, err := json.Marshal(map[string]any{
		"source": map[string]any{"issue_number": 5, "issue_url": "https://github.com/gkzz/renodash/issues/5", "fetched_at": "2026-11-13T00:00:00Z"},
		"items":  items, "pull_requests": prs, "bulk_actions": []any{}, "unknown_controls": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func assertContains(t *testing.T, output string, expected ...string) {
	t.Helper()
	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("missing %q in %s", s, output)
		}
	}
}
func TestSummaryEmpty(t *testing.T) {
	output := renderSummary(t, summaryData(t, nil, nil))
	assertContains(t, output, "**Issue:** [#5]", "**Candidates:** 0", "**Bulk actions:** 0", "**Unknown controls:** 0", "**Linked PRs:** 0", "No candidates", "No linked pull requests", "verification was skipped")
}
func TestSummaryMissingChecks(t *testing.T) {
	for _, mode := range []string{"missing", "empty", "null"} {
		t.Run(mode, func(t *testing.T) {
			pr := map[string]any{"number": 7, "url": "https://github.com/gkzz/renodash/pull/7", "state": "OPEN", "is_draft": true, "title": "Update"}
			if mode == "empty" {
				pr["checks"] = []any{}
			} else if mode == "null" {
				pr["checks"] = nil
			}
			output := renderSummary(t, summaryData(t, nil, []any{pr}))
			assertContains(t, output, "| OPEN | true | Update | Unknown (no checks returned) |", "Check results are observations, not merge approval.")
		})
	}
}
func TestSummaryEscapingAndChecks(t *testing.T) {
	title := "A & <b> \\ *x* _y_ [z](url) | `code`\r\nnext"
	escaped := "A &amp; &lt;b&gt; &#92; &#42;x&#42; &#95;y&#95; &#91;z&#93;(url) &#124; &#96;code&#96; next"
	output := renderSummary(t, summaryData(t,
		[]any{map[string]any{"section": title, "action": title, "title": title}},
		[]any{map[string]any{"number": 7, "url": "https://github.com/gkzz/renodash/pull/7", "state": "CLOSED", "is_draft": false, "title": title,
			"checks": []any{map[string]any{"name": title, "conclusion": "SUCCESS"}, map[string]any{"context": "legacy", "state": "PENDING"}, map[string]any{"name": "failed", "conclusion": "FAILURE"}, map[string]any{"name": "running", "status": "IN_PROGRESS"}, map[string]any{}}}},
	))
	if strings.Count(output, escaped) != 5 || strings.Contains(output, title) {
		t.Fatal(output)
	}
	assertContains(t, output, "| CLOSED | false |", "real GitHub responses captured and saved in advance", "Dashboard body is constructed for testing", "not current GitHub state", ": SUCCESS<br>legacy: PENDING<br>failed: FAILURE<br>running: IN&#95;PROGRESS<br>Unnamed check: Unknown")
}

func TestOfflineSummaryAdapter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake-actions-summary")
	build := exec.Command("go", "build", "-o", path, "./internal/tools/fake-actions-summary")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build adapter: %v: %s", err, out)
	}
	data, stderr, err := newHarness(t).run("")
	if err != nil {
		t.Fatalf("offline CLI: %v: %s", err, stderr)
	}
	for _, input := range [][]byte{data, []byte("{"), nil, append(append([]byte{}, data...), []byte(" {}")...)} {
		cmd := exec.Command(path)
		cmd.Env = []string{"PATH=" + t.TempDir()}
		cmd.Stdin = bytes.NewReader(input)
		var errors bytes.Buffer
		cmd.Stderr = &errors
		out, err := cmd.Output()
		if bytes.Equal(input, data) {
			if err != nil || string(out) != renderSummary(t, input) || errors.Len() != 0 {
				t.Fatalf("adapter: %v stdout=%s stderr=%s", err, out, &errors)
			}
		} else if err == nil || len(out) != 0 || errors.Len() == 0 {
			t.Fatalf("invalid input: %v stdout=%s stderr=%s", err, out, &errors)
		}
	}
	cmd := exec.Command(path, "unexpected")
	if out, err := cmd.CombinedOutput(); err == nil || !bytes.Contains(out, []byte("usage:")) {
		t.Fatalf("arguments: %v: %s", err, out)
	}
}

func TestOfflineSummaryGolden(t *testing.T) {
	data, stderr, err := newHarness(t).run("")
	if err != nil {
		t.Fatalf("offline CLI: %v: %s", err, stderr)
	}
	result := decode(t, data)
	result["source"].(map[string]any)["fetched_at"] = "2026-11-13T00:00:00Z"
	data, err = json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "internal/summary/testdata/offline.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if got := renderSummary(t, data); got != string(want) {
		t.Fatalf("offline summary differs from jq golden:\n%s", got)
	}
}
