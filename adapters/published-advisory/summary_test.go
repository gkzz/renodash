package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSummaryMode(t *testing.T) {
	for _, scenario := range []string{"matching", "failed-ci", "source-failure", "no-match", "incomplete", "other-pr", "old-head", "version", "distribution", "files", "reordered-files", "missing", "malformed", "oversized"} {
		t.Run(scenario, func(t *testing.T) {
			input := loadInput(t, "pr53-cve.json")
			context := map[string]any{
				"pr_url": input.PRURL, "pr_head_sha": input.HeadSHA,
				"inspected_files": append([]string(nil), input.Files...), "target": input.Target,
				"ci_sha": strings.Repeat("a", 40), "code_sha": strings.Repeat("b", 40), "ci_outcome": "success",
			}
			wantMatch := scenario == "matching" || scenario == "failed-ci" || scenario == "reordered-files" || scenario == "source-failure"
			switch scenario {
			case "source-failure":
				failed := input.Sources[0]
				failed.Payload, failed.Failure = nil, "Synthetic retrieval failure"
				input.Sources = append(input.Sources, failed)
			case "no-match":
				input.Target.Old, input.Target.Proposed = "24.22.3", "26.13.1"
				context["target"] = input.Target
			case "incomplete":
				input = loadInput(t, "pr53.json")
			case "failed-ci":
				context["ci_outcome"] = "failure"
			case "other-pr":
				context["pr_url"] = "https://github.com/gkzz/renodash/pull/999"
			case "old-head":
				context["pr_head_sha"] = strings.Repeat("c", 40)
			case "version":
				input.Target.Proposed = "26.13.1"
			case "distribution":
				input.Target.Distribution = "unofficial"
			case "files":
				input.Files = []string{"other.lock"}
			case "reordered-files":
				input.Files[0], input.Files[1] = input.Files[1], input.Files[0]
			case "oversized":
				input.Sources[0].Coverage = strings.Repeat("<", 200000)
			}
			dir := t.TempDir()
			contextPath, snapshotPath := filepath.Join(dir, "context.json"), filepath.Join(dir, "snapshot.json")
			writeJSON(t, contextPath, context)
			if scenario != "missing" {
				writeJSON(t, snapshotPath, input)
			}
			if scenario == "malformed" {
				if err := os.WriteFile(snapshotPath, []byte("{"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			err := run([]string{"-summary-context", contextPath, "-snapshot", snapshotPath, "-format", "markdown"}, strings.NewReader(""), &out)
			if err != nil {
				t.Fatal(err)
			}
			got := out.String()
			for _, text := range []string{"Offline fixture experiment", "CI SHA: " + strings.Repeat("a", 40), "Executed code SHA: " + strings.Repeat("b", 40), "CI outcome: " + context["ci_outcome"].(string)} {
				if !strings.Contains(got, text) {
					t.Fatalf("missing %q: %s", text, got)
				}
			}
			if wantMatch {
				if !strings.Contains(got, "Affected") || !strings.Contains(got, "CVE-2026-123863") {
					t.Fatal(got)
				}
			} else if scenario == "no-match" {
				if !strings.Contains(got, "NoMatchInCheckedSources") || strings.Contains(got, "Affected") {
					t.Fatal(got)
				}
			} else if !strings.Contains(got, "Unknown") || strings.Contains(got, "CVE-2026-123863") || strings.Contains(got, "Affected") {
				t.Fatal(got)
			}
			if scenario == "missing" || scenario == "other-pr" || scenario == "old-head" || scenario == "malformed" {
				if strings.Contains(got, "no sources supplied") {
					t.Error("snapshot rejection obscured by generic missing-source reason")
				}
			}
			if scenario == "incomplete" && !strings.Contains(got, "<details>") {
				t.Error("large checked-range list is not folded")
			}
			if scenario == "source-failure" && !strings.Contains(got, "Synthetic retrieval failure") {
				t.Fatal(got)
			}
			if len(out.Bytes()) > 512<<10 {
				t.Fatal("summary exceeds bound")
			}
		})
	}
}

func TestSummaryContextValidationAndWriteFailure(t *testing.T) {
	original, err := os.ReadFile("testdata/summary-context.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"null": []byte("null"), "trailing": append(append([]byte(nil), original...), []byte(" {}")...),
		"unknown-field": bytes.Replace(original, []byte(`"ci_sha"`), []byte(`"unexpected"`), 1),
		"bad-sha":       bytes.Replace(original, []byte(strings.Repeat("a", 40)), []byte("bad"), 1),
		"bad-outcome":   bytes.Replace(original, []byte(`"unknown"`), []byte(`"green"`), 1),
		"too-large":     bytes.Repeat([]byte(" "), (16<<20)+1),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "context.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := run([]string{"-summary-context", path, "-snapshot", "testdata/pr53-cve.json", "-format", "markdown"}, strings.NewReader(""), &out); err == nil || out.Len() != 0 {
				t.Fatalf("err=%v output=%s", err, &out)
			}
		})
	}
	err = run([]string{"-summary-context", "testdata/summary-context.json", "-snapshot", "testdata/pr53-cve.json", "-format", "markdown"}, strings.NewReader(""), shortWriter{})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
}

func TestConflictingRangeEvidence(t *testing.T) {
	in := loadInput(t, "pr53-cve.json")
	in.Target.Old, in.Target.Proposed = "24.1.0", "26.1.0"
	other := in.Sources[0]
	other.URL = "https://example.invalid/synthetic-narrative"
	other.Revision = "synthetic"
	other.Payload = json.RawMessage(`{"synthetic":{"cve":["CVE-2026-123863"],"vulnerable":">=24.22.0 <24.22.3 || >=26.9.0 <26.13.1","patched":">=24.22.3 <25.0.0 || >=26.13.1","ref":"https://example.invalid/synthetic-narrative","affectedEnvironments":["all"]}}`)
	in.Sources = append(in.Sources, other)
	r := buildReport(in, time.Now())
	got := string(markdown(r))
	for _, text := range []string{"different published ranges for CVE-2026-123863", "&gt;=24.22.0", "&gt;=24.0.0", "synthetic-narrative"} {
		if !strings.Contains(got, text) {
			t.Errorf("missing %q: %s", text, got)
		}
	}
	if r.Old.State != "Affected" || r.Proposed.State != "Affected" || r.Sources[1].Proposed.State != "NoMatchInCheckedSources" {
		t.Fatal(r)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
