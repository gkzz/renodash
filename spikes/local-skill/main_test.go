package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// This test executable supplies a deterministic process interface without Go builds or network access.
func TestMain(m *testing.M) {
	if os.Getenv("RENODASH_TEST_ADAPTER") == "1" {
		adapterProcess()
		return
	}
	os.Exit(m.Run())
}
func adapterProcess() {
	if os.Getenv("RENODASH_TEST_ADAPTER") != "1" {
		return
	}
	var input object
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		os.Exit(2)
	}
	target := input["target"].(object)
	sources := input["sources"].([]any)
	state := func(v string) assessment {
		a := assessment{State: "Unknown", Limitations: []string{"synthetic limitation"}}
		if target["product"] == "nodejs" && len(sources) > 0 {
			s := sources[0].(object)
			p, ok := s["payload"].(object)
			if ok && len(p) > 0 {
				switch v {
				case "18.9.0", "18.8.0":
					a.State = "Affected"
					a.Matches = append(a.Matches, struct {
						CVEs []string `json:"cves"`
					}{[]string{"CVE-SYNTHETIC"}})
				case "18.9.1":
					a.State = "NoMatchInCheckedSources"
				}
			}
		}
		return a
	}
	out := object{"old": state(target["old_version"].(string)), "proposed": state(target["proposed_version"].(string)), "checked_at": "2026-11-15T00:00:00Z", "sources": []any{}}
	if len(sources) > 0 {
		out["sources"] = []any{object{"source": sources[0]}}
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
func TestContracts(t *testing.T) {
	t.Setenv("RENODASH_TEST_ADAPTER", "1")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err = contracts(binary); err != nil {
		t.Fatal(err)
	}
}
func TestRejectEmptyAdapterOutput(t *testing.T) {
	if _, err := evaluate("/bin/true", snapshot("18.9.0", "18.9.1")); err == nil {
		t.Fatal("empty output accepted")
	}
}
func TestOriginalAdapterContracts(t *testing.T) {
	binary := os.Getenv("RENODASH_ADVISORY_BINARY")
	if binary == "" {
		t.Skip("set RENODASH_ADVISORY_BINARY for pinned adapter integration")
	}
	if err := contracts(binary); err != nil {
		t.Fatal(err)
	}
}
func TestInspection(t *testing.T) {
	bundle := t.TempDir()
	if err := os.Mkdir(filepath.Join(bundle, "runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	raw := `{"Config":{"User":"spike","Env":["HOME=/tmp/home"]},"HostConfig":{"NetworkMode":"none","ReadonlyRootfs":true,"CapDrop":["ALL"],"SecurityOpt":["no-new-privileges:true"]},"Mounts":[{"Type":"bind","Destination":"/spike","RW":false}]}`
	cases := []struct {
		name   string
		mutate func(*inspection)
	}{{"valid", func(*inspection) {}}, {"root", func(i *inspection) { i.Config.User = "root" }}, {"network", func(i *inspection) { i.HostConfig.NetworkMode = "bridge" }}, {"writable-root", func(i *inspection) { i.HostConfig.ReadonlyRootfs = false }}, {"capabilities", func(i *inspection) { i.HostConfig.CapDrop = nil }}, {"elevation", func(i *inspection) { i.HostConfig.SecurityOpt = nil }}, {"writable-mount", func(i *inspection) { i.Mounts[0].RW = true }}, {"extra-mount", func(i *inspection) { i.Mounts = append(i.Mounts, i.Mounts[0]) }}, {"credential", func(i *inspection) { i.Config.Env = append(i.Config.Env, "GH_TOKEN=synthetic") }}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var i inspection
			if err := json.Unmarshal([]byte(raw), &i); err != nil {
				t.Fatal(err)
			}
			i.Mounts[0].Source = filepath.Join(bundle, "runtime")
			c.mutate(&i)
			err := checkInspection(i, bundle)
			if (err == nil) != (c.name == "valid") {
				t.Fatalf("unexpected verdict: %v", err)
			}
		})
	}
}
