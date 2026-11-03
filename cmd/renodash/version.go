package main

import (
	"fmt"
	"io"
	"runtime/debug"
)

// Set by setup/release-build.sh using -ldflags -X.
var releaseVersion, commit string

func buildVersion(release string, info *debug.BuildInfo) string {
	if release != "" {
		return release
	}
	if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

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
	_, err := fmt.Fprintf(stdout, "renodash %s (commit: %s)\n", buildVersion(releaseVersion, info), buildCommit(commit, info))
	return err
}
