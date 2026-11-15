package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func snapshot(payload string) Source { return Source{Payload: json.RawMessage(payload)} }
func TestNodeCoreRanges(t *testing.T) {
	target := Target{Product: "nodejs", Distribution: "official", Environment: "linux"}
	source := snapshot(`{"99":{"cve":["CVE-2022-32222"],"vulnerable":"14.x || 16.x || 18.x","patched":"^14.20.0 || ^16.20.0 || ^18.9.1","ref":"https://nodejs.org/en/blog/vulnerability/september-2022-security-releases/","affectedEnvironments":["linux"]}}`)
	for _, tc := range []struct {
		version, state string
		matches        int
	}{
		{"18.9.0", "Affected", 1}, {"18.9.1", "NoMatchInCheckedSources", 0}, {"16.19.9", "Affected", 1}, {"16.20.0", "NoMatchInCheckedSources", 0}, {"24.22.2", "NoMatchInCheckedSources", 0}, {"18.9.1-rc.1", "Unknown", 0}, {"broken", "Unknown", 0},
	} {
		t.Run(tc.version, func(t *testing.T) {
			got := (NodeCoreAdapter{}).Assess(target, tc.version, source)
			if got.State != tc.state || len(got.Matches) != tc.matches {
				t.Fatalf("got %+v, want %s / %d matches", got, tc.state, tc.matches)
			}
		})
	}
	target.Distribution = "npm"
	if got := (NodeCoreAdapter{}).Assess(target, "18.9.0", source); got.State != "Unknown" {
		t.Fatalf("npm substitution: %+v", got)
	}
	target.Distribution = "official"
	target.Environment = "windows"
	if got := (NodeCoreAdapter{}).Assess(target, "18.9.0", source); got.State != "NoMatchInCheckedSources" {
		t.Fatalf("environment: %+v", got)
	}
}
func TestIncompleteEvidence(t *testing.T) {
	target := Target{Product: "nodejs", Distribution: "official", Environment: "linux"}
	for _, payload := range []string{`null`, `{}`, `[]`, `{`, `{"1":{"vulnerable":"???","patched":"^18.9.1"}}`, `{"1":{"vulnerable":"18.x"}}`} {
		if got := (NodeCoreAdapter{}).Assess(target, "18.9.0", snapshot(payload)); got.State != "Unknown" || len(got.Limitations) == 0 {
			t.Errorf("payload %s: %+v", payload, got)
		}
	}
	got := (NodeCoreAdapter{}).Assess(target, "18.9.0", Source{Failure: "fetch failed"})
	if got.State != "Unknown" || !strings.Contains(strings.Join(got.Limitations, " "), "fetch failed") {
		t.Fatalf("failure: %+v", got)
	}
}
func TestAggregatePreservesMatchesAndLimitations(t *testing.T) {
	got := aggregate([]Assessment{{State: "Affected", Matches: []Finding{{ID: "confirmed"}}}, {State: "Unknown", Limitations: []string{"offline"}}})
	if got.State != "Affected" || len(got.Matches) != 1 || len(got.Limitations) != 1 {
		t.Fatalf("got %+v", got)
	}
	for _, tc := range []struct {
		in    []Assessment
		state string
	}{
		{nil, "Unknown"}, {[]Assessment{{State: "NoMatchInCheckedSources"}}, "NoMatchInCheckedSources"}, {[]Assessment{{State: "NoMatchInCheckedSources"}, {State: "Unknown", Limitations: []string{"offline"}}}, "Unknown"},
	} {
		if got := aggregate(tc.in); got.State != tc.state {
			t.Errorf("got %+v want %s", got, tc.state)
		}
	}
}

// Fabricated records for matching behavior only; no real vulnerability claim.
func TestEnvironments(t *testing.T) {
	for _, tc := range []struct{ input, source, state string }{
		{"linux", "linux", "Affected"}, {"darwin", "darwin", "Affected"}, {"macos", "darwin", "Affected"},
		{"win32", "win32", "Affected"}, {"windows", "win32", "Affected"}, {"linux", "all", "Affected"},
		{"all", "linux", "Unknown"}, {"plan9", "all", "Unknown"}, {"ubuntu", "linux", "Unknown"},
		{"linux", "future-os", "Unknown"}, {"linux", "win32", "NoMatchInCheckedSources"},
	} {
		t.Run(tc.input+"/"+tc.source, func(t *testing.T) {
			source := snapshot(`{"1":{"vulnerable":"18.x","patched":"^18.9.1","ref":"https://example.invalid/test-only-advisory","affectedEnvironments":["` + tc.source + `"]}}`)
			got := (NodeCoreAdapter{}).Assess(Target{Product: "nodejs", Distribution: "official", Environment: tc.input}, "18.9.0", source)
			if got.State != tc.state || (tc.state == "Unknown" && len(got.Limitations) == 0) {
				t.Fatalf("got %+v want %s", got, tc.state)
			}
		})
	}
}

func TestEnvironmentMatchPreservesUnknownLabel(t *testing.T) {
	source := snapshot(`{"TEST-ONLY":{"vulnerable":"18.x","patched":"^18.9.1","ref":"https://example.invalid/test-only","affectedEnvironments":["linux","future-os"]}}`)
	got := (NodeCoreAdapter{}).Assess(Target{Product: "nodejs", Distribution: "official", Environment: "linux"}, "18.9.0", source)
	if got.State != "Affected" || len(got.Matches) != 1 || len(got.Limitations) != 1 {
		t.Fatal(got)
	}
}
