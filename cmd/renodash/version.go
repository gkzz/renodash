package main

import (
	"fmt"
	"io"
	"runtime/debug"
)

const version = "v0.0.3"

// Set by setup/release-build.sh using -ldflags -X.
var releaseVersion, commit string

func buildCommit(releaseCommit string, info *debug.BuildInfo) string {
	c := releaseCommit
	if info != nil {
		if c == "" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					c = setting.Value
				}
			}
		}
	}
	if c == "" {
		c = "unknown"
	}
	return c
}

func printVersion(stdout io.Writer) error {
	info, _ := debug.ReadBuildInfo()
	v := version
	if releaseVersion != "" {
		v = releaseVersion
	}
	_, err := fmt.Fprintf(stdout, "renodash %s (commit: %s)\n", v, buildCommit(commit, info))
	return err
}
