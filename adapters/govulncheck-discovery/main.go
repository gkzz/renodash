package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/vuln/scan"
)

const database = "https://vuln.go.dev"
const sdkVersion = "golang.org/x/vuln v1.9.0"

type Target struct {
	Ecosystem    string `json:"ecosystem"`
	Distribution string `json:"distribution"`
	Module       string `json:"module"`
	Old          string `json:"old_version"`
	Proposed     string `json:"proposed_version"`
}
type Assessment struct {
	Version      string            `json:"version"`
	State        string            `json:"state"`
	Matches      []string          `json:"matches"`
	Limitations  []string          `json:"limitations"`
	Records      []json.RawMessage `json:"records"`
	SDKConfig    json.RawMessage   `json:"sdk_config,omitempty"`
	Query        string            `json:"query"`
	RetrievedAt  string            `json:"retrieved_at"`
	EvidenceFile string            `json:"evidence_file,omitempty"`
	Digest       string            `json:"sha256,omitempty"`
}
type Report struct {
	Experiment      string     `json:"experiment"`
	SDK             string     `json:"sdk"`
	Database        string     `json:"database"`
	Target          Target     `json:"target"`
	SelectionReason string     `json:"selection_reason"`
	CheckedAt       string     `json:"checked_at"`
	Old             Assessment `json:"old"`
	Proposed        Assessment `json:"proposed"`
	Coverage        string     `json:"coverage"`
}

// Runner is a narrow boundary around the public Go API, not an advisory client.
type Runner interface {
	Query(context.Context, string, io.Writer, io.Writer) error
}
type officialSDK struct{ db string }

func (s officialSDK) Query(ctx context.Context, query string, out, diagnostics io.Writer) error {
	db := s.db
	if db == "" {
		db = database
	}
	cmd := scan.Command(ctx, "-mode=query", "-json", "-db", db, query)
	cmd.Stdout, cmd.Stderr = out, diagnostics
	cmd.Stdin = bytes.NewReader(nil)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}
func decodeTarget(data []byte) (Target, error) {
	var t Target
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return t, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return t, fmt.Errorf("expected one target JSON object")
	}
	if t.Module == "" || t.Distribution == "" || t.Ecosystem == "" || t.Old == "" || t.Proposed == "" {
		return t, fmt.Errorf("module, ecosystem, distribution, and both versions are required")
	}
	return t, nil
}
func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("govulncheck-discovery", flag.ContinueOnError)
	input := fs.String("input", "", "target JSON; no advisory IDs")
	evidence := fs.String("evidence-dir", "", "new evidence directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *input == "" || *evidence == "" {
		return fmt.Errorf("usage: govulncheck-discovery -input target.json -evidence-dir NEW_DIR")
	}
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	if err != nil {
		return err
	}
	if len(data) > 64<<10 {
		return fmt.Errorf("target exceeds 64 KiB")
	}
	t, err := decodeTarget(data)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	r, err := inspect(ctx, officialSDK{}, t, *evidence)
	if err != nil {
		return err
	}
	data, err = json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(*evidence, "report.json"), data, 0600); err != nil {
		return err
	}
	n, err := out.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}
func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
