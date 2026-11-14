# Version and build information

`renodash version` prints the version and source commit without reading standard input or making network requests.
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
| 3 | `unknown` | Neither source provides a commit |

The CLI does not infer a SHA from a Go pseudo-version.
Builds without version control system (VCS) information, such as those built with `-buildvcs=false`, may have no SHA.
Some tagged `go install` builds may also have no SHA. Missing build information does not make the command fail.

The commit display includes `dirty` only when Go build information has `vcs.modified: true`. This also applies to embedded release versions and an `unknown` SHA. If `vcs.modified` is absent, no `dirty` marker is shown.

The CLI does not invoke `git` at runtime. The SHA identifies the source commit; `dirty` only indicates uncommitted changes at build time. It does not describe those changes, so two builds with the same version and SHA can still differ.

## Implementation and test evidence

- Implementation: [version.go](../../../../internal/command/version.go).
- Selection and output tests: [version_test.go](../../../../internal/command/version_test.go).
