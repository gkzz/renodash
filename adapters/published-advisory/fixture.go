package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// prepareFixture adapts saved experiment inputs, never live PR or advisory data.
func prepareFixture(args []string) error {
	fs := flag.NewFlagSet("prepare-fixture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scenario := fs.String("scenario", "", "offline experiment scenario")
	fixtureDir := fs.String("fixture-dir", "testdata", "saved fixture directory")
	outputDir := fs.String("output-dir", "", "generated input directory")
	ciSHA := fs.String("ci-sha", "", "SHA tested by CI")
	codeSHA := fs.String("code-sha", "", "executed code SHA")
	outcome := fs.String("ci-outcome", "", "observed CI outcome")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || strings.TrimSpace(*outputDir) == "" {
		return fmt.Errorf("prepare-fixture requires an output directory and no positional arguments")
	}
	switch *scenario {
	case "affected", "no-match", "incomplete", "mismatch", "missing", "source-failure", "conflicting", "oversized", "failed-ci":
	default:
		return fmt.Errorf("unsupported fixture scenario: %s", *scenario)
	}
	c, err := readSummaryContext(filepath.Join(*fixtureDir, "summary-context.json"))
	if err != nil {
		return err
	}
	c.CISHA, c.CodeSHA, c.CIOutcome = *ciSHA, *codeSHA, *outcome
	if err := validateSummaryContext(c); err != nil {
		return err
	}
	fixture := "pr53-cve.json"
	if *scenario == "incomplete" {
		fixture = "pr53.json"
	}
	f, err := os.Open(filepath.Join(*fixtureDir, fixture))
	if err != nil {
		return fmt.Errorf("open fixture: %w", err)
	}
	defer f.Close()
	snapshot, err := parseInput(f)
	if err != nil {
		return fmt.Errorf("parse fixture: %w", err)
	}
	if len(snapshot.Sources) == 0 {
		return fmt.Errorf("fixture requires at least one saved source")
	}
	switch *scenario {
	case "no-match":
		c.Target.Old, c.Target.Proposed = "24.22.3", "26.13.1"
		snapshot.Target = c.Target
	case "mismatch":
		c.HeadSHA = strings.Repeat("c", 40)
	case "source-failure":
		failed := snapshot.Sources[0]
		failed.Payload, failed.Failure = nil, "Synthetic partial retrieval failure"
		snapshot.Sources = append(snapshot.Sources, failed)
	case "conflicting":
		alternative := snapshot.Sources[0]
		alternative.URL = "https://example.invalid/synthetic-conflicting-source"
		alternative.Revision = "synthetic-narrative-range"
		alternative.Coverage = "Synthetic narrative interpretation; conflicts with structured range. Manual review required."
		alternative.Payload = json.RawMessage(`{"CVE-2026-123863":{"cve":["CVE-2026-123863"],"vulnerable":">=24.22.0 <24.22.3 || >=26.9.0 <26.13.1","patched":">=24.22.3 <25.0.0 || >=26.13.1","ref":"https://nodejs.org/en/blog/vulnerability/november-2026-security-releases/","affectedEnvironments":["all"]}}`)
		snapshot.Sources = append(snapshot.Sources, alternative)
		c.Target.Old, c.Target.Proposed = "24.1.0", "26.1.0"
		snapshot.Target = c.Target
	case "oversized":
		snapshot.Sources[0].Coverage = strings.Repeat("<", 200000)
	}
	contextData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	snapshotData, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*outputDir, 0700); err != nil {
		return err
	}
	snapshotPath := filepath.Join(*outputDir, "advisory-snapshot.json")
	if *scenario == "missing" {
		if err := os.Remove(snapshotPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if err := os.WriteFile(snapshotPath, append(snapshotData, '\n'), 0600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*outputDir, "advisory-context.json"), append(contextData, '\n'), 0600)
}
