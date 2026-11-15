package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func loadInput(t *testing.T, name string) Input {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	in, err := parseInput(f)
	if err != nil {
		t.Fatal(err)
	}
	return in
}
func TestCapturedBoundary(t *testing.T) {
	in := loadInput(t, "boundary.json")
	report := buildReport(in, time.Date(2026, 11, 14, 15, 0, 0, 0, time.UTC))
	if report.Old.State != "Affected" || report.Proposed.State != "NoMatchInCheckedSources" {
		t.Fatalf("old=%+v proposed=%+v", report.Old, report.Proposed)
	}
	if report.Old.Matches[0].CVEs[0] != "CVE-2022-32222" || report.Sources[0].Source.Revision != "f643f84d4977d0ba379dc7c5805c9b65206369b5" {
		t.Fatalf("lost provenance: %+v", report)
	}
	if report.CheckedAt != "2026-11-14T15:00:00Z" || len(report.Sources[0].Source.Payload) != 0 {
		t.Fatal("incorrect output metadata")
	}
	// A different source failing must not hide a confirmed match.
	failed := in.Sources[0]
	failed.Payload = nil
	failed.Failure = "retrieval failed"
	in.Sources = append(in.Sources, failed)
	report = buildReport(in, time.Now())
	if report.Old.State != "Affected" || len(report.Old.Limitations) != 1 || report.Proposed.State != "Unknown" {
		t.Fatalf("aggregation: %+v", report)
	}
}
func TestUnknownAdapter(t *testing.T) {
	in := loadInput(t, "boundary.json")
	in.Sources[0].Adapter = "unsupported"
	r := buildReport(in, time.Now())
	if r.Old.State != "Unknown" || r.Proposed.State != "Unknown" {
		t.Fatal(r)
	}
}
func TestValidation(t *testing.T) {
	original := loadInput(t, "boundary.json")
	for name, modify := range map[string]func(*Input){
		"sha": func(i *Input) { i.HeadSHA = "" }, "url": func(i *Input) { i.PRURL = "javascript:alert(1)" }, "source URL": func(i *Input) { i.Sources[0].URL = "https://user:pass@example.com" }, "UTC": func(i *Input) { i.Sources[0].FetchedAt = "2026-11-14T15:00:00+09:00" }, "provenance": func(i *Input) { i.Sources[0].Revision = "" }, "ambiguous source": func(i *Input) { i.Sources[0].Failure = "offline" },
	} {
		t.Run(name, func(t *testing.T) {
			data, _ := json.Marshal(original)
			var input Input
			json.Unmarshal(data, &input)
			modify(&input)
			data, _ = json.Marshal(input)
			var out bytes.Buffer
			if err := run(nil, bytes.NewReader(data), &out); err == nil || out.Len() != 0 {
				t.Fatalf("err=%v output=%s", err, &out)
			}
		})
	}
	for _, s := range []string{"null", "{}", `{} {}`, `{"unexpected":true}`} {
		var out bytes.Buffer
		if err := run(nil, strings.NewReader(s), &out); err == nil || out.Len() != 0 {
			t.Fatalf("input %s: %v", s, err)
		}
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
func TestOutput(t *testing.T) {
	in := loadInput(t, "boundary.json")
	in.Files = []string{"<script>|[link](javascript:x)\nnext"}
	in.Sources[0].Coverage = "<img src=x>|\ntext"
	data, _ := json.Marshal(in)
	var out bytes.Buffer
	if err := run([]string{"-format", "markdown"}, bytes.NewReader(data), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<script>") || strings.Contains(out.String(), "<img") || !strings.Contains(out.String(), "&#124;") || !strings.Contains(out.String(), "CVE-2022-32222") {
		t.Fatal(out.String())
	}
	if err := run(nil, bytes.NewReader(data), shortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	out.Reset()
	if err := run(nil, bytes.NewReader(data), &out); err != nil {
		t.Fatal(err)
	}
	var got Report
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Schema != "spike.v1" {
		t.Fatalf("%v %s", err, &out)
	}
}
func TestPRSnapshot(t *testing.T) {
	in := loadInput(t, "pr53.json")
	r := buildReport(in, time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	if r.HeadSHA != "cc4ccda827937e4d7b9ce164899d3781acd04085" || r.Target != (Target{Product: "nodejs", Distribution: "official", Environment: "linux", Old: "24.22.2", Proposed: "26.13.0"}) {
		t.Fatalf("identity: %+v", r)
	}
	if strings.Join(r.Files, ",") != ".config/mise/config.toml,.config/mise/mise.lock" {
		t.Fatal(r.Files)
	}
	var records map[string]json.RawMessage
	if err := json.Unmarshal(in.Sources[0].Payload, &records); err != nil || len(records) != 204 {
		t.Fatalf("snapshot: %d %v", len(records), err)
	}
	if len(r.Sources) != 1 || r.Sources[0].Source.Revision != "f643f84d4977d0ba379dc7c5805c9b65206369b5" || !strings.Contains(r.Sources[0].Source.Coverage, "November security coverage is unverified") {
		t.Fatal(r.Sources)
	}
	for _, a := range []Assessment{r.Old, r.Proposed} {
		if a.State != "Unknown" || len(a.Matches) != 0 || len(a.Limitations) != 8 {
			t.Fatalf("assessment: %+v", a)
		}
		for i, id := range []string{"2", "27", "28", "29", "33", "34", "35", "38"} {
			if a.Limitations[i] != "record "+id+": missing ranges, reference, or environment" {
				t.Fatal(a.Limitations)
			}
		}
	}
}

func TestInputSizeLimit(t *testing.T) {
	in := loadInput(t, "boundary.json")
	data, _ := json.Marshal(in)
	data = append(data, bytes.Repeat([]byte(" "), 16<<20)...)
	if _, err := parseInput(bytes.NewReader(data)); err == nil {
		t.Fatal("oversized input accepted")
	}
}
func TestRecordFailurePreservesMatch(t *testing.T) {
	in := loadInput(t, "boundary.json")
	var records map[string]json.RawMessage
	json.Unmarshal(in.Sources[0].Payload, &records)
	records["malformed"] = json.RawMessage(`{"vulnerable":"???"}`)
	in.Sources[0].Payload, _ = json.Marshal(records)
	r := buildReport(in, time.Now())
	if r.Old.State != "Affected" || len(r.Old.Matches) != 1 || len(r.Old.Limitations) != 1 || r.Proposed.State != "Unknown" {
		t.Fatalf("old=%+v proposed=%+v", r.Old, r.Proposed)
	}
}

// This manually transcribed single-CVE example does not extend the saved core index.
func TestPR53PublishedCVE(t *testing.T) {
	in := loadInput(t, "pr53-cve.json")
	raw, err := os.ReadFile("testdata/CVE-2026-123863.json")
	if err != nil {
		t.Fatal(err)
	}
	if in.Sources[0].Revision != fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) {
		t.Fatal("raw CNA snapshot digest mismatch")
	}
	r := buildReport(in, time.Date(2026, 11, 15, 0, 52, 16, 0, time.UTC))
	for _, a := range []Assessment{r.Old, r.Proposed} {
		if a.State != "Affected" || len(a.Matches) != 1 || a.Matches[0].CVEs[0] != "CVE-2026-123863" || a.Matches[0].Vulnerable != ">=24.0.0 <24.22.3 || >=25.0.0 <26.13.1" {
			t.Fatalf("%+v", a)
		}
	}
}
func TestExternalMarkdownText(t *testing.T) {
	in := loadInput(t, "boundary.json")
	in.Sources[0].Coverage = "![x](https://evil.test) @everyone <script> & &#10;\n# heading"
	r := buildReport(in, time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	got := string(markdown(r))
	for _, unsafe := range []string{"![x]", "https://evil.test", "@everyone", "<script>", "\n# heading"} {
		if strings.Contains(got, unsafe) {
			t.Fatalf("unescaped %q", unsafe)
		}
	}
	if !strings.Contains(got, "Proposed") || !strings.Contains(got, r.HeadSHA) || !strings.Contains(got, "official / linux") {
		t.Fatal(got)
	}
}
