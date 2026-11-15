package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeSDK func(context.Context, string, io.Writer, io.Writer) error

func (f fakeSDK) Query(c context.Context, q string, o, e io.Writer) error { return f(c, q, o, e) }
func testTarget() Target {
	return Target{Ecosystem: "Go", Distribution: "upstream-module", Module: "example.org/library", Old: "v1.2.2", Proposed: "v1.2.3"}
}

const config = `{"config":{"protocol_version":"v1.0.0","scan_mode":"query","db":"https://vuln.go.dev"}}` + "\n"

// Fabricated SDK output; no real vulnerability is asserted by this record.
const record = `{"osv":{"id":"GO-SYNTHETIC-1","affected":[{"package":{"name":"example.org/library","ecosystem":"Go"}}],"aliases":["CVE-SYNTHETIC"]}}` + "\n"

func TestSDKAdapter(t *testing.T) {
	var calls []string
	sdk := fakeSDK(func(ctx context.Context, q string, out, stderr io.Writer) error {
		calls = append(calls, q)
		io.WriteString(out, config)
		if strings.HasSuffix(q, "@v1.2.2") {
			io.WriteString(out, record)
		}
		return nil
	})
	dir := filepath.Join(t.TempDir(), "evidence")
	got, err := inspect(context.Background(), sdk, testTarget(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "example.org/library@v1.2.2,example.org/library@v1.2.3" {
		t.Fatal(calls)
	}
	if got.Old.State != "Affected" || got.Proposed.State != "NoMatchInCheckedSources" || len(got.Old.Records) != 1 || len(got.Old.Matches) != 1 {
		t.Fatal(got)
	}
	for _, a := range []Assessment{got.Old, got.Proposed} {
		if raw, err := os.ReadFile(filepath.Join(dir, a.EvidenceFile)); err != nil || len(raw) == 0 || len(a.Digest) != 64 || a.RetrievedAt == "" {
			t.Fatal(a, err)
		}
	}
	if _, err := inspect(context.Background(), sdk, testTarget(), dir); err == nil {
		t.Fatal("existing evidence overwritten")
	}
}
func TestSDKFailuresRemainUnknown(t *testing.T) {
	for _, mode := range []string{"failure", "empty", "malformed", "missing-config", "wrong-mode", "unknown-protocol", "wrong-module", "limit", "partial"} {
		t.Run(mode, func(t *testing.T) {
			sdk := fakeSDK(func(_ context.Context, _ string, out, stderr io.Writer) error {
				switch mode {
				case "failure":
					return fmt.Errorf("synthetic database failure")
				case "empty":
					return nil
				case "malformed":
					io.WriteString(out, config+"{")
				case "missing-config":
					io.WriteString(out, record)
				case "wrong-mode":
					io.WriteString(out, strings.Replace(config, "query", "source", 1))
				case "unknown-protocol":
					io.WriteString(out, strings.Replace(config, "v1.0.0", "v2.0.0", 1))
				case "wrong-module":
					io.WriteString(out, config+strings.Replace(record, "example.org/library", "example.org/other", 1))
				case "limit":
					_, err := io.WriteString(out, strings.Repeat("x", (8<<20)+1))
					return err
				case "partial":
					io.WriteString(out, config+record)
					return fmt.Errorf("synthetic partial failure")
				}
				return nil
			})
			dir := filepath.Join(t.TempDir(), "evidence")
			got, err := inspect(context.Background(), sdk, testTarget(), dir)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "limit" {
				data, err := os.ReadFile(filepath.Join(dir, got.Old.EvidenceFile))
				if err != nil || len(data) > 8<<20 || !strings.Contains(strings.Join(got.Old.Limitations, " "), "capture limit") {
					t.Fatal("capture was not bounded", len(data), err)
				}
			}
			want := "Unknown"
			if mode == "partial" {
				want = "Affected"
			}
			if got.Old.State != want || len(got.Old.Limitations) == 0 {
				t.Fatal(got.Old)
			}
		})
	}
}
func TestUnsupportedTargetSkipsSDK(t *testing.T) {
	for _, module := range []string{"stdlib", "example.org/library@v1.0.0", "-help"} {
		in := testTarget()
		in.Module = module
		got, err := inspect(context.Background(), fakeSDK(func(context.Context, string, io.Writer, io.Writer) error {
			t.Fatal("unsupported input queried")
			return nil
		}), in, filepath.Join(t.TempDir(), "evidence"))
		if err != nil || got.Old.State != "Unknown" || got.SelectionReason == "" {
			t.Fatal(got, err)
		}
	}
}
func TestOfficialSDKCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (officialSDK{}).Query(ctx, "example.org/library@v1.2.2", io.Discard, io.Discard); err == nil {
		t.Fatal("canceled SDK query succeeded")
	}
}
func TestSavedOfficialSDKOutput(t *testing.T) {
	var digests map[string]string
	raw, err := os.ReadFile("testdata/live-20261115/sha256.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &digests); err != nil {
		t.Fatal(err)
	}
	for name, want := range digests {
		data, err := os.ReadFile(filepath.Join("testdata/live-20261115", name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			t.Fatal("capture digest mismatch", name)
		}
	}
	in := Target{Ecosystem: "Go", Distribution: "upstream-module", Module: "golang.org/x/text", Old: "v0.3.6", Proposed: "v0.3.7"}
	sdk := fakeSDK(func(_ context.Context, q string, out, stderr io.Writer) error {
		name := "proposed.sdk.json"
		if strings.HasSuffix(q, "@v0.3.6") {
			name = "old.sdk.json"
		}
		data, err := os.ReadFile(filepath.Join("testdata/live-20261115", name))
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	})
	got, err := inspect(context.Background(), sdk, in, filepath.Join(t.TempDir(), "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Old.State != "Affected" || got.Proposed.State != "Affected" || len(got.Old.Matches) != 4 || len(got.Proposed.Matches) != 3 {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(got.Old.Matches, []string{"GO-2021-0113", "GO-2022-1059", "GO-2026-5970", "GO-2026-6629"}) ||
		!reflect.DeepEqual(got.Proposed.Matches, []string{"GO-2022-1059", "GO-2026-5970", "GO-2026-6629"}) {
		t.Fatal(got)
	}
	if len(got.Old.Limitations) != 0 || len(got.Proposed.Limitations) != 0 {
		t.Fatal(got)
	}
	for _, a := range []Assessment{got.Old, got.Proposed} {
		for _, raw := range a.Records {
			var r map[string]any
			if err := json.Unmarshal(raw, &r); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestSDKRecordHandling(t *testing.T) {
	for _, tc := range []struct {
		name, stream, want string
		limitations        bool
		records            int
	}{
		{"duplicate", config + record + record, "Affected", false, 1},
		{"conflict", config + record + strings.Replace(record, "CVE-SYNTHETIC", "CVE-SYNTHETIC-OTHER", 1), "Affected", true, 2},
		{"withdrawn", config + strings.Replace(record, `"id":"GO-SYNTHETIC-1"`, `"id":"GO-SYNTHETIC-1","withdrawn":"2020-01-01T00:00:00Z"`, 1), "NoMatchInCheckedSources", false, 1},
		{"partial-stream", config + record + "{", "Affected", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sdk := fakeSDK(func(_ context.Context, _ string, out, stderr io.Writer) error {
				_, err := io.WriteString(out, tc.stream)
				return err
			})
			got, err := inspect(context.Background(), sdk, testTarget(), filepath.Join(t.TempDir(), "evidence"))
			if err != nil {
				t.Fatal(err)
			}
			if got.Old.State != tc.want || len(got.Old.Records) != tc.records || (len(got.Old.Limitations) > 0) != tc.limitations {
				t.Fatal(got.Old)
			}
		})
	}
}
func TestTargetValidation(t *testing.T) {
	for _, data := range []string{`null`, `{}`, `{"ecosystem":"Go","cve":"not-an-input"}`, `{} {}`} {
		if _, err := decodeTarget([]byte(data)); err == nil {
			t.Fatal("accepted invalid target", data)
		}
	}
	for _, mode := range []string{"ecosystem", "distribution", "version"} {
		in := testTarget()
		switch mode {
		case "ecosystem":
			in.Ecosystem = "npm"
		case "distribution":
			in.Distribution = "debian"
		case "version":
			in.Proposed = "v1.2.3@other"
		}
		got, err := inspect(context.Background(), fakeSDK(func(context.Context, string, io.Writer, io.Writer) error {
			t.Fatal("unsupported target called API")
			return nil
		}), in, filepath.Join(t.TempDir(), "evidence"))
		if err != nil || got.Proposed.State != "Unknown" {
			t.Fatal(got, err)
		}
	}
}
