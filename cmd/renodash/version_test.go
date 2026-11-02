package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func TestBuildVersion(t *testing.T) {
	for _, tt := range []struct {
		name, release, sha, module, revision, wantVersion, wantCommit string
	}{
		{"release overrides", "v0.0.2", "release-sha", "v0.0.1", "vcs-sha", "v0.0.2", "release-sha"},
		{"module tag", "", "", "v0.0.1", "", "v0.0.1", "unknown"},
		{"development", "", "", "(devel)", "vcs-sha", "dev", "vcs-sha"},
		{"missing", "", "", "", "", "dev", "unknown"},
		{"partial release", "v0.0.2", "", "(devel)", "vcs-sha", "v0.0.2", "vcs-sha"},
		{"partial commit", "", "release-sha", "v0.0.1", "vcs-sha", "v0.0.1", "release-sha"},
		{"pseudo base", "", "", "v0.0.0-20260101000000-abcdef012345", "", "dev", "unknown"},
		{"pseudo patch", "", "", "v0.0.2-0.20260101000000-abcdef012345", "", "dev", "unknown"},
		{"pseudo prerelease", "", "", "v0.0.2-rc.1.0.20260101000000-abcdef012345+incompatible", "", "dev", "unknown"},
		{"dirty tag", "", "", "v0.0.2+dirty", "vcs-sha", "dev", "vcs-sha"},
		{"dirty pseudo", "", "", "v0.0.2-0.20261102144408-4ae40a406fd0+dirty", "vcs-sha", "dev", "vcs-sha"},
		{"prerelease tag", "", "", "v0.1.0-rc.1", "", "v0.1.0-rc.1", "unknown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			info := &debug.BuildInfo{Main: debug.Module{Version: tt.module}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: tt.revision}, {Key: "vcs.modified", Value: "true"}}}
			v, c := buildVersion(tt.release, tt.sha, info)
			if v != tt.wantVersion || c != tt.wantCommit {
				t.Fatalf("got %q %q, want %q %q", v, c, tt.wantVersion, tt.wantCommit)
			}
		})
	}
	v, c := buildVersion("", "", nil)
	if v != "dev" || c != "unknown" {
		t.Fatalf("no build info: %q %q", v, c)
	}
}

func TestVersionCommand(t *testing.T) {
	oldVersion, oldCommit := version, commit
	t.Cleanup(func() { version, commit = oldVersion, oldCommit })
	version, commit = "v0.0.2", "0123456789abcdef"
	var out, diagnostic bytes.Buffer
	if code := run([]string{"version"}, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || out.String() != "renodash v0.0.2 (commit: 0123456789abcdef)\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	for _, args := range [][]string{{"version", "extra"}, {"version", "--json"}, {"version", "--help", "extra"}} {
		out.Reset()
		diagnostic.Reset()
		if code := run(args, failingIO{}, &out, &diagnostic); code == 0 || out.Len() != 0 || diagnostic.Len() == 0 {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, out.String(), diagnostic.String())
		}
	}
	for _, args := range [][]string{{"--help"}, {"-h"}, {"version", "--help"}, {"version", "-h"}} {
		out.Reset()
		diagnostic.Reset()
		if code := run(args, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || !strings.Contains(out.String(), "renodash version") {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, out.String(), diagnostic.String())
		}
	}
	diagnostic.Reset()
	if code := run([]string{"version"}, failingIO{}, failingIO{}, &diagnostic); code == 0 || !strings.Contains(diagnostic.String(), "write failed") {
		t.Fatalf("code=%d stderr=%q", code, diagnostic.String())
	}
}
