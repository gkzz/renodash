package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Static registration of one official API; no home-grown source client or range matcher.
func eligibility(t Target) string {
	if t.Ecosystem != "Go" || t.Distribution != "upstream-module" {
		return "unsupported ecosystem or distribution; requires a Go upstream module"
	}
	if !regexp.MustCompile(`^[a-z0-9.-]+\.[a-z]{2,}/[A-Za-z0-9._~!/+\-]+$`).MatchString(t.Module) || strings.Contains(t.Module, "..") || strings.Contains(t.Module, "//") || strings.HasSuffix(t.Module, "/") {
		return "unsupported Go module identity; runtime and stdlib are outside this experiment"
	}
	for _, v := range []string{t.Old, t.Proposed} {
		if strings.ContainsAny(v, "@ \t\r\n") || strings.HasPrefix(v, "-") {
			return "invalid compound module query"
		}
	}
	return ""
}
func emptyAssessment(v string) Assessment {
	return Assessment{Version: v, State: "Unknown", Matches: []string{}, Records: []json.RawMessage{}, Limitations: []string{}}
}
func inspect(ctx context.Context, sdk Runner, t Target, dir string) (Report, error) {
	r := Report{Experiment: "Official Go API advisory discovery — NOT MERGE", SDK: sdkVersion, Database: database, Target: t, Old: emptyAssessment(t.Old), Proposed: emptyAssessment(t.Proposed), Coverage: "Only Go Vulnerability Database records returned by the official API for this exact upstream module and supplied versions. No completeness, freshness, exploitability, symbol reachability, environment applicability, transitive coverage, Node.js runtime support, or current PR assessment is established. Saved bytes are SDK output, not raw database HTTP responses."}
	if err := os.Mkdir(dir, 0700); err != nil {
		return r, fmt.Errorf("create new evidence directory: %w", err)
	}
	r.SelectionReason = eligibility(t)
	if r.SelectionReason != "" {
		r.Old.Limitations = []string{r.SelectionReason}
		r.Proposed.Limitations = []string{r.SelectionReason}
	} else {
		r.SelectionReason = "Selected the official golang.org/x/vuln/scan query API for an upstream Go module. Target identity is caller-supplied."
		var err error
		r.Old, err = query(ctx, sdk, t, t.Old, "old", dir)
		if err != nil {
			return r, err
		}
		r.Proposed, err = query(ctx, sdk, t, t.Proposed, "proposed", dir)
		if err != nil {
			return r, err
		}
	}
	r.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	return r, nil
}

type capture struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (c *capture) WriteString(s string) (int, error) { return c.Write([]byte(s)) }

func (c *capture) Write(p []byte) (int, error) {
	if len(p) > c.limit-c.Len() {
		n, _ := c.Buffer.Write(p[:c.limit-c.Len()])
		c.exceeded = true
		return n, fmt.Errorf("SDK output capture limit exceeded")
	}
	return c.Buffer.Write(p)
}
func query(ctx context.Context, sdk Runner, t Target, v, label, dir string) (Assessment, error) {
	a := emptyAssessment(v)
	a.Query = t.Module + "@" + v
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	stdout, stderr := &capture{limit: 8 << 20}, &capture{limit: 1 << 20}
	err := sdk.Query(ctx, a.Query, stdout, stderr)
	a.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	a.EvidenceFile = label + ".sdk.json"
	sum := sha256.Sum256(stdout.Bytes())
	a.Digest = hex.EncodeToString(sum[:])
	if e := os.WriteFile(filepath.Join(dir, a.EvidenceFile), stdout.Bytes(), 0600); e != nil {
		return a, e
	}
	if e := os.WriteFile(filepath.Join(dir, label+".stderr.txt"), stderr.Bytes(), 0600); e != nil {
		return a, e
	}
	if err != nil {
		a.Limitations = append(a.Limitations, "official API query failed: "+err.Error())
	}
	if ctx.Err() != nil {
		a.Limitations = append(a.Limitations, "query context: "+ctx.Err().Error())
	}
	if stdout.exceeded || stderr.exceeded {
		a.Limitations = append(a.Limitations, "SDK output exceeds capture limit; saved output is incomplete")
	}
	if stderr.Len() > 0 {
		a.Limitations = append(a.Limitations, "SDK diagnostics present; inspect "+label+".stderr.txt")
	}
	parseStream(&a, t, stdout.Bytes())
	if len(a.Matches) > 0 {
		a.State = "Affected"
	} else if len(a.Limitations) == 0 {
		a.State = "NoMatchInCheckedSources"
	}
	meta, e := json.MarshalIndent(a, "", "  ")
	if e != nil {
		return a, e
	}
	if e := os.WriteFile(filepath.Join(dir, label+".assessment.json"), append(meta, '\n'), 0600); e != nil {
		return a, e
	}
	return a, nil
}
func parseStream(a *Assessment, t Target, data []byte) {
	dec := json.NewDecoder(bytes.NewReader(data))
	validConfig := false
	known := map[string][]byte{}
	for {
		var message map[string]json.RawMessage
		err := dec.Decode(&message)
		if err == io.EOF {
			break
		}
		if err != nil || len(message) != 1 {
			a.Limitations = append(a.Limitations, "invalid SDK JSON stream")
			break
		}
		for kind, raw := range message {
			switch kind {
			case "config":
				var c struct {
					Protocol string `json:"protocol_version"`
					Mode     string `json:"scan_mode"`
					DB       string `json:"db"`
				}
				if err := json.Unmarshal(raw, &c); err != nil || c.Protocol != "v1.0.0" || c.Mode != "query" || c.DB != database {
					a.Limitations = append(a.Limitations, "unsupported SDK protocol, mode, or database")
					continue
				}
				if validConfig {
					a.Limitations = append(a.Limitations, "duplicate SDK configuration")
				}
				validConfig = true
				a.SDKConfig = bytes.Clone(raw)
			case "osv":
				if !validConfig {
					a.Limitations = append(a.Limitations, "OSV record before valid SDK configuration")
					continue
				}
				var rec struct {
					ID        string `json:"id"`
					Withdrawn string `json:"withdrawn"`
					Affected  []struct {
						Package struct {
							Name      string `json:"name"`
							Ecosystem string `json:"ecosystem"`
						} `json:"package"`
					} `json:"affected"`
				}
				if err := json.Unmarshal(raw, &rec); err != nil || rec.ID == "" {
					a.Limitations = append(a.Limitations, "invalid SDK advisory record")
					continue
				}
				exact := false
				for _, item := range rec.Affected {
					if item.Package.Name == t.Module && item.Package.Ecosystem == "Go" {
						exact = true
					}
				}
				if !exact {
					a.Limitations = append(a.Limitations, rec.ID+": no exact Go module identity match")
					continue
				}
				var canonical bytes.Buffer
				if err := json.Compact(&canonical, raw); err != nil {
					a.Limitations = append(a.Limitations, "invalid native advisory JSON")
					continue
				}
				if before, ok := known[rec.ID]; ok {
					if bytes.Equal(before, canonical.Bytes()) {
						continue
					}
					a.Limitations = append(a.Limitations, "differing records for "+rec.ID+"; manual review required")
				}
				known[rec.ID] = bytes.Clone(canonical.Bytes())
				a.Records = append(a.Records, bytes.Clone(raw))
				if rec.Withdrawn == "" && !slices.Contains(a.Matches, rec.ID) {
					a.Matches = append(a.Matches, rec.ID)
				}
			case "progress": // Informational SDK progress; raw stream preserves the text.
			default:
				a.Limitations = append(a.Limitations, "unexpected SDK query message: "+kind)
			}
		}
	}
	if !validConfig {
		a.Limitations = append(a.Limitations, "missing valid SDK query configuration")
	}
}
