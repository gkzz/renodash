package integration

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func renderSummary(t *testing.T, data []byte) string {
	t.Helper()
	// jq runs in the test driver, outside the CLI's isolated environment.
	cmd := exec.Command("jq", "-r", "-f", filepath.Join(root, ".github/scripts/renodash-summary.jq"))
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("jq: %v: %s", err, out)
	}
	return string(out)
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
