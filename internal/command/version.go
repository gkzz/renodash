package command

import (
	"fmt"
	"io"
	"runtime/debug"
)

// Set by setup/release-build.sh using -ldflags -X.
var releaseVersion, commit string
var readBuildInfo = debug.ReadBuildInfo

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
	return c
}

func buildDirty(info *debug.BuildInfo) bool {
	if info == nil {
		return false
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.modified" {
			return setting.Value == "true"
		}
	}
	return false
}

func printVersion(stdout io.Writer) error {
	info, _ := readBuildInfo()
	version := buildVersion(releaseVersion, info)
	suffix := ""
	if sha := buildCommit(commit, info); version == "dev" && sha != "" {
		dirty := ""
		if buildDirty(info) {
			dirty = ", dirty"
		}
		suffix = fmt.Sprintf(" (commit: %s%s)", sha, dirty)
	}
	_, err := fmt.Fprintf(stdout, "renodash %s%s\n", version, suffix)
	return err
}
