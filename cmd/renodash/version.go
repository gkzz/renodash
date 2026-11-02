package main

import (
	"fmt"
	"io"
	"regexp"
	"runtime/debug"
	"strings"
)

// Set by setup/release-build.sh using -ldflags -X.
var version, commit string

var moduleVersion = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
var pseudoVersion = regexp.MustCompile(`-([0-9]{14}|[^+]*\.[0-9]{14})-[0-9a-f]+(\+[0-9A-Za-z.-]+)?$`)

func buildVersion(releaseVersion, releaseCommit string, info *debug.BuildInfo) (string, string) {
	v, c := releaseVersion, releaseCommit
	if info != nil {
		if v == "" && moduleVersion.MatchString(info.Main.Version) && !pseudoVersion.MatchString(info.Main.Version) && !strings.HasSuffix(info.Main.Version, "+dirty") {
			v = info.Main.Version
		}
		if c == "" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					c = setting.Value
				}
			}
		}
	}
	if v == "" {
		v = "dev"
	}
	if c == "" {
		c = "unknown"
	}
	return v, c
}

func printVersion(stdout io.Writer) error {
	info, _ := debug.ReadBuildInfo()
	v, c := buildVersion(version, commit, info)
	_, err := fmt.Fprintf(stdout, "renodash %s (commit: %s)\n", v, c)
	return err
}
