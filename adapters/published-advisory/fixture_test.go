package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrepareFixture(t *testing.T) {
	for _, scenario := range []string{"affected", "no-match", "incomplete", "mismatch", "missing", "source-failure", "conflicting", "oversized", "failed-ci"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			outcome := "success"
			if scenario == "failed-ci" {
				outcome = "failure"
			}
			args := []string{"prepare-fixture", "-scenario", scenario, "-ci-sha", strings.Repeat("a", 40), "-code-sha", strings.Repeat("b", 40), "-ci-outcome", outcome, "-output-dir", dir}
			var stdout bytes.Buffer
			if err := run(args, strings.NewReader(""), &stdout); err != nil {
				t.Fatal(err)
			}
			if stdout.Len() != 0 {
				t.Fatal("unexpected stdout", &stdout)
			}
			contextPath, snapshotPath := filepath.Join(dir, "advisory-context.json"), filepath.Join(dir, "advisory-snapshot.json")
			context, err := readSummaryContext(contextPath)
			if err != nil {
				t.Fatal(err)
			}
			if context.PRURL != "https://github.com/gkzz/renodash/pull/53" || context.CISHA != strings.Repeat("a", 40) || context.CodeSHA != strings.Repeat("b", 40) || context.CIOutcome != outcome {
				t.Fatal(context)
			}
			if scenario != "mismatch" && context.HeadSHA != "cc4ccda827937e4d7b9ce164899d3781acd04085" {
				t.Fatal(context.HeadSHA)
			}
			r, notice := summarySnapshot(context, snapshotPath, time.Now())
			want := "Affected"
			switch scenario {
			case "no-match":
				want = "NoMatchInCheckedSources"
			case "incomplete", "mismatch", "missing":
				want = "Unknown"
			}
			if r.Proposed.State != want {
				t.Fatalf("got %s want %s: %s", r.Proposed.State, want, notice)
			}
			if scenario == "incomplete" && len(r.Proposed.Limitations) != 8 {
				t.Fatal(r.Proposed)
			}
			if scenario == "source-failure" && !strings.Contains(strings.Join(r.Proposed.Limitations, " "), "Synthetic partial retrieval failure") {
				t.Fatal(r.Proposed)
			}
			if scenario == "conflicting" && (len(r.Sources) != 2 || !strings.Contains(strings.Join(r.Proposed.Limitations, " "), "different published ranges for CVE-2026-123863")) {
				t.Fatal(r)
			}
			if err := runSummary(contextPath, snapshotPath, &stdout, time.Now()); err != nil {
				t.Fatal(err)
			}
			if scenario == "oversized" && !strings.Contains(stdout.String(), "summary exceeds 512 KiB") {
				t.Fatal(stdout.String())
			}
			// Reuse the directory to prove that missing never consumes a previous snapshot.
			args[2] = "missing"
			if err := run(args, strings.NewReader(""), &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(snapshotPath); !os.IsNotExist(err) {
				t.Fatalf("snapshot remains: %v", err)
			}
		})
	}
}

func TestPrepareFixtureRejectsInvalidInputs(t *testing.T) {
	for _, flags := range [][]string{
		{"-scenario", "unknown"}, {"-ci-sha", "bad"}, {"-ci-outcome", "green"}, {"-fixture-dir", "absent"}, {"extra"},
	} {
		dir := t.TempDir()
		args := []string{"prepare-fixture", "-scenario", "affected", "-ci-sha", strings.Repeat("a", 40), "-code-sha", strings.Repeat("b", 40), "-ci-outcome", "success", "-output-dir", dir}
		args = append(args, flags...)
		if err := run(args, strings.NewReader(""), &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", flags)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("partial output: %v %v", entries, err)
		}
	}
}

func TestPrepareFixtureRejectsInvalidSavedData(t *testing.T) {
	for _, file := range []string{"summary-context.json", "pr53-cve.json"} {
		t.Run(file, func(t *testing.T) {
			fixtures, output := t.TempDir(), t.TempDir()
			for _, name := range []string{"summary-context.json", "pr53-cve.json"} {
				data, err := os.ReadFile(filepath.Join("testdata", name))
				if err != nil {
					t.Fatal(err)
				}
				if name == file {
					data = []byte(`{"invalid":true}`)
				}
				if err := os.WriteFile(filepath.Join(fixtures, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := prepareFixture([]string{"-scenario", "affected", "-ci-sha", strings.Repeat("a", 40), "-code-sha", strings.Repeat("b", 40), "-ci-outcome", "success", "-fixture-dir", fixtures, "-output-dir", output})
			if err == nil {
				t.Fatal("invalid saved data accepted")
			}
			entries, err := os.ReadDir(output)
			if err != nil || len(entries) != 0 {
				t.Fatalf("partial output: %v %v", entries, err)
			}
		})
	}
}
