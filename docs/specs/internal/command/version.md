# Version and build information

`renodash version` prints the version and, for development builds, an available source commit without reading standard input or making network requests.
See [Usage](../../../usage.md#check-the-version) for commands.

## Behavior

### Version selection

The displayed version uses the first available value in this order:

1. The release version embedded with `-ldflags -X`
2. The main module version in Go build information
3. `dev`

Installing a tag with `go install` displays that tag's version. A pseudo-version installed through mise's Go backend is displayed as supplied. Local `go run` builds display `dev`.

### Commit and dirty marker

The commit SHA is the identifier of the source commit. The CLI selects it in this order:

| Priority | Source | When used |
| --- | --- | --- |
| 1 | Embedded release commit | The embedded value is nonempty |
| 2 | Go's `vcs.revision` build setting | No embedded commit is available |
| 3 | No commit suffix | Neither source provides a commit |

The CLI does not infer a SHA from a Go pseudo-version.
Builds without version control system (VCS) information, such as those built with `-buildvcs=false`, may have no SHA.
Some tagged `go install` builds may also have no SHA. Missing build information does not make the command fail.

Only the selected version `dev` includes an available commit SHA.
All other versions omit the commit and dirty marker, including tagged installs, embedded releases, and pseudo-versions.
Development builds include `dirty` only when a SHA is available and Go build information has `vcs.modified: true`.
Without a SHA, development builds omit the whole suffix, even when dirty. The CLI never displays `commit: unknown`.

## Output and failures

Successful version output goes to stdout with a trailing newline and empty stderr.

| Build | Output |
| --- | --- |
| Tagged install or embedded release | `renodash v0.0.6` |
| Development build with SHA | `renodash dev (commit: abc1234)` |
| Development build with SHA and uncommitted changes | `renodash dev (commit: abc1234, dirty)` |
| Development build without SHA | `renodash dev` |

The CLI does not invoke `git` at runtime. The SHA identifies the source commit; `dirty` only indicates uncommitted changes at build time. It does not describe those changes, so two builds with the same version and SHA can still differ.

## Implementation and test evidence

- Implementation: [version.go](../../../../internal/command/version.go).
- Selection and output tests: [version_test.go](../../../../internal/command/version_test.go).
