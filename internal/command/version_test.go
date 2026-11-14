package command

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
		{"missing", "", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			info := &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: tt.revision}}}
			if got := buildCommit(tt.sha, info); got != tt.wantCommit {
				t.Fatalf("got %q, want %q", got, tt.wantCommit)
			}
		})
	}
	if got := buildCommit("", nil); got != "" {
		t.Fatalf("no build info: %q", got)
	}
}

func TestBuildVersion(t *testing.T) {
	for _, tt := range []struct {
		name, release, moduleVersion, want string
	}{
		{"release overrides module", "v0.0.4", "v0.0.3", "v0.0.4"},
		{"tagged module", "", "v0.0.4", "v0.0.4"},
		{"pseudo version", "", "v0.0.4-0.20261103085119-d4ec06cd2d6d", "v0.0.4-0.20261103085119-d4ec06cd2d6d"},
		{"local build", "", "(devel)", "dev"},
		{"empty version", "", "", "dev"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			info := &debug.BuildInfo{Main: debug.Module{Version: tt.moduleVersion}}
			if got := buildVersion(tt.release, info); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
	if got := buildVersion("", nil); got != "dev" {
		t.Fatalf("no build info: %q", got)
	}
}

func TestBuildDirty(t *testing.T) {
	for _, tt := range []struct {
		name     string
		settings []debug.BuildSetting
		want     bool
	}{
		{"true", []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}, true},
		{"false", []debug.BuildSetting{{Key: "vcs.modified", Value: "false"}}, false},
		{"empty", []debug.BuildSetting{{Key: "vcs.modified", Value: ""}}, false},
		{"missing", []debug.BuildSetting{{Key: "vcs.revision", Value: "sha"}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildDirty(&debug.BuildInfo{Settings: tt.settings}); got != tt.want {
				t.Fatalf("got %t, want %t", got, tt.want)
			}
		})
	}
	if buildDirty(nil) {
		t.Fatal("nil build info must not be dirty")
	}
}

func TestVersionCommand(t *testing.T) {
	oldVersion, oldCommit := releaseVersion, commit
	oldReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		releaseVersion, commit = oldVersion, oldCommit
		readBuildInfo = oldReadBuildInfo
	})
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	}
	commit = "0123456789abcdef"
	var out, diagnostic bytes.Buffer
	if code := Run([]string{"version"}, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || out.String() != "renodash dev (commit: 0123456789abcdef)\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	releaseVersion = "v0.1.0"
	out.Reset()
	if code := Run([]string{"version"}, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || out.String() != "renodash v0.1.0\n" {
		t.Fatalf("release build: code=%d stdout=%q", code, out.String())
	}
	for _, tt := range []struct {
		name, version, releaseCommit, moduleVersion, revision, modified, want string
	}{
		{"tagged install", "", "", "v0.0.4", "", "", "renodash v0.0.4\n"},
		{"unknown commit clean", "", "", "(devel)", "", "false", "renodash dev\n"},
		{"pseudo version", "", "", "v0.0.4-0.20261103085119-d4ec06cd2d6d", "vcs-sha", "true", "renodash v0.0.4-0.20261103085119-d4ec06cd2d6d\n"},
		{"local clean", "", "", "(devel)", "vcs-sha", "false", "renodash dev (commit: vcs-sha)\n"},
		{"local dirty", "", "", "(devel)", "vcs-sha", "true", "renodash dev (commit: vcs-sha, dirty)\n"},
		{"unknown commit dirty", "", "", "(devel)", "", "true", "renodash dev\n"},
		{"release dirty", "v0.1.0", "release-sha", "v0.0.4", "vcs-sha", "true", "renodash v0.1.0\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			releaseVersion, commit = tt.version, tt.releaseCommit
			readBuildInfo = func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: tt.moduleVersion},
					Settings: []debug.BuildSetting{
						{Key: "vcs.revision", Value: tt.revision},
						{Key: "vcs.modified", Value: tt.modified},
					},
				}, true
			}
			out.Reset()
			diagnostic.Reset()
			if code := Run([]string{"version"}, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || out.String() != tt.want {
				t.Fatalf("code=%d stdout=%q stderr=%q, want %q", code, out.String(), diagnostic.String(), tt.want)
			}
		})
	}
	releaseVersion, commit = "", ""
	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
	out.Reset()
	diagnostic.Reset()
	if code := Run([]string{"version"}, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || out.String() != "renodash dev\n" {
		t.Fatalf("missing build info: code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
	releaseVersion, commit = "v0.1.0", "0123456789abcdef"
	for _, args := range [][]string{{"version", "extra"}, {"version", "--json"}, {"version", "--help", "extra"}} {
		out.Reset()
		diagnostic.Reset()
		if code := Run(args, failingIO{}, &out, &diagnostic); code == 0 || out.Len() != 0 || diagnostic.Len() == 0 {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, out.String(), diagnostic.String())
		}
	}
	for _, args := range [][]string{{"--help"}, {"-h"}, {"version", "--help"}, {"version", "-h"}} {
		out.Reset()
		diagnostic.Reset()
		if code := Run(args, failingIO{}, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 || !strings.Contains(out.String(), "renodash version") {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, out.String(), diagnostic.String())
		}
	}
	diagnostic.Reset()
	if code := Run([]string{"version"}, failingIO{}, failingIO{}, &diagnostic); code == 0 || !strings.Contains(diagnostic.String(), "write failed") {
		t.Fatalf("code=%d stderr=%q", code, diagnostic.String())
	}
}
