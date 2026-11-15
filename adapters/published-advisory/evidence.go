package main

import (
	"encoding/json"
)

type Target struct {
	Product      string `json:"product"`
	Distribution string `json:"distribution"`
	Environment  string `json:"environment"`
	Old          string `json:"old_version"`
	Proposed     string `json:"proposed_version"`
}
type Source struct {
	Adapter   string          `json:"adapter"`
	URL       string          `json:"url"`
	Revision  string          `json:"revision"`
	FetchedAt string          `json:"fetched_at"`
	Coverage  string          `json:"coverage"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Failure   string          `json:"failure,omitempty"`
}
type Finding struct {
	ID         string   `json:"id"`
	CVEs       []string `json:"cves"`
	Vulnerable string   `json:"vulnerable_range"`
	Patched    string   `json:"patched_range"`
	Reference  string   `json:"reference"`
}
type Assessment struct {
	State       string    `json:"state"`
	Matches     []Finding `json:"matches"`
	Limitations []string  `json:"limitations"`
}

// EvidenceAdapter is a source-specific boundary; core aggregation does not parse provider payloads.
type EvidenceAdapter interface {
	Assess(Target, string, Source) Assessment
}
type NodeCoreAdapter struct{}

func unknown(reason string) Assessment {
	return Assessment{State: "Unknown", Matches: []Finding{}, Limitations: []string{reason}}
}
func aggregate(results []Assessment) Assessment {
	out := Assessment{State: "NoMatchInCheckedSources", Matches: []Finding{}, Limitations: []string{}}
	if len(results) == 0 {
		return unknown("no sources supplied")
	}
	for _, r := range results {
		out.Matches = append(out.Matches, r.Matches...)
		out.Limitations = append(out.Limitations, r.Limitations...)
		if r.State == "Unknown" && len(r.Limitations) == 0 {
			out.Limitations = append(out.Limitations, "source assessment incomplete")
		}
	}
	if len(out.Limitations) > 0 {
		out.State = "Unknown"
	}
	if len(out.Matches) > 0 {
		out.State = "Affected"
	}
	return out
}
