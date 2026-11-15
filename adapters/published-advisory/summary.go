package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"time"
)

const summaryLimit = 512 << 10

// SummaryContext is independently supplied comparison data, not authenticated metadata.
type SummaryContext struct {
	PRURL                string   `json:"pr_url"`
	HeadSHA              string   `json:"pr_head_sha"`
	Files                []string `json:"inspected_files"`
	Target               Target   `json:"target"`
	CISHA                string   `json:"ci_sha"`
	CIOutcome            string   `json:"ci_outcome"`
	CodeSHA              string   `json:"code_sha"`
	IntentionalCIFailure bool     `json:"intentional_ci_failure,omitempty"`
}

func readSummaryContext(path string) (SummaryContext, error) {
	var c SummaryContext
	f, err := os.Open(path)
	if err != nil {
		return c, fmt.Errorf("open summary context: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (16<<20)+1))
	if err != nil {
		return c, err
	}
	if len(data) > 16<<20 {
		return c, fmt.Errorf("summary context exceeds 16 MiB")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode summary context: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("expected one context JSON object")
	}
	return c, validateSummaryContext(c)
}

func validateSummaryContext(c SummaryContext) error {
	// Reuse the adapter's identity validation without trusting snapshot contents.
	identity, err := json.Marshal(Input{PRURL: c.PRURL, HeadSHA: c.HeadSHA, Files: c.Files, Target: c.Target})
	if err != nil {
		return err
	}
	if _, err := parseInput(bytes.NewReader(identity)); err != nil {
		return fmt.Errorf("invalid context identity: %w", err)
	}
	if !shaPattern.MatchString(c.CISHA) || !shaPattern.MatchString(c.CodeSHA) {
		return fmt.Errorf("40-character CI and executed code SHAs are required")
	}
	switch c.CIOutcome {
	case "success", "failure", "cancelled", "skipped", "unknown":
	default:
		return fmt.Errorf("invalid CI outcome")
	}
	return nil
}

func sameFiles(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func summarySnapshot(c SummaryContext, path string, now time.Time) (Report, string) {
	empty := buildReport(Input{PRURL: c.PRURL, HeadSHA: c.HeadSHA, Files: c.Files, Target: c.Target}, now)
	reject := func(reason string) (Report, string) {
		empty.Old, empty.Proposed = unknown(reason), unknown(reason)
		return empty, "Unknown: " + reason + "; no findings published."
	}
	f, err := os.Open(path)
	if err != nil {
		return reject("snapshot unavailable")
	}
	defer f.Close()
	in, err := parseInput(f)
	if err != nil {
		return reject("invalid snapshot")
	}
	if in.PRURL != c.PRURL || in.HeadSHA != c.HeadSHA || in.Target != c.Target || !sameFiles(in.Files, c.Files) {
		return reject("snapshot target mismatch")
	}
	return buildReport(in, now), "Snapshot matches the supplied context. Context is caller-supplied and not authenticated."
}

func runSummary(contextPath, snapshotPath string, out io.Writer, now time.Time) error {
	c, err := readSummaryContext(contextPath)
	if err != nil {
		return err
	}
	r, notice := summarySnapshot(c, snapshotPath, now)
	var b bytes.Buffer
	fmt.Fprintln(&b, "## Offline fixture experiment — NOT MERGE")
	fmt.Fprintln(&b, "\nThis is not a current PR assessment. No PR diff, live advisory acquisition, authenticity check, or automatic freshness enforcement is performed.")
	fmt.Fprintf(&b, "\nExpected PR head SHA: %s\n\nCI SHA: %s\n\nExecuted code SHA: %s\n\nCI outcome: %s\n\nAssessment time (UTC): %s\n", c.HeadSHA, c.CISHA, c.CodeSHA, c.CIOutcome, now.UTC().Format(time.RFC3339))
	if c.IntentionalCIFailure {
		fmt.Fprintln(&b, "\nIntentional failure probe: the workflow simulates a failing step with continue-on-error. CI outcome records that probe and adapter tests, not the overall job conclusion.")
	}
	if c.HeadSHA != c.CISHA {
		fmt.Fprintln(&b, "\nCI SHA differs from the expected PR head; CI does not establish tests of that exact head.")
	}
	fmt.Fprintln(&b, "\nSaved source retrieval times do not establish freshness or complete coverage. Conflicting source evidence requires manual review.")
	metadata := append([]byte(nil), b.Bytes()...)
	fmt.Fprintf(&b, "\n%s\n\n", notice)
	b.Write(markdown(r))
	data := b.Bytes()
	if len(data) > summaryLimit {
		data = append(metadata, []byte("\nUnknown: summary exceeds 512 KiB; evidence omitted rather than truncated. Review the saved input locally.\n")...)
	}
	n, err := out.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}
