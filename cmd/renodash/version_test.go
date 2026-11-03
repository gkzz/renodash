package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func TestBuildCommit(t *testing.T) {
	for _, tt := range []struct {
		name, sha, revision, wantCommit string
	}{
		{"release overrides", "release-sha", "vcs-sha", "release-sha"},
		{"vcs revision", "", "vcs-sha", "vcs-sha"},
		{"missing", "", "", "unknown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			info := &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: tt.revision}}}
			if got := buildCommit(tt.sha, info); got != tt.wantCommit {
				t.Fatalf("got %q, want %q", got, tt.wantCommit)
			}
		})
	}
	if got := buildCommit("", nil); got != "unknown" {
		t.Fatalf("no build info: %q", got)
	}
}

func TestVersionCommand(t *testing.T) {
	oldVersion, oldCommit := releaseVersion, commit
	t.Cleanup(func() { releaseVersion, commit = oldVersion, oldCommit })
	commit = "0123456789abcdef"
	var out, diagnostic bytes.Buffer
	if code := run([]string{"version"}, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || out.String() != "renodash "+version+" (commit: 0123456789abcdef)\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	releaseVersion = "v0.1.0"
	out.Reset()
	if code := run([]string{"version"}, failingIO{}, &out, &diagnostic); code != 0 || out.String() != "renodash v0.1.0 (commit: 0123456789abcdef)\n" {
		t.Fatalf("release build: code=%d stdout=%q", code, out.String())
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
