# Version and build information

The displayed version uses the first available value in this order:

1. The release version embedded with `-ldflags -X`
2. The main module version in Go build information
3. `dev`

Installing a tag with `go install` displays that tag's version. A pseudo-version installed through mise's Go backend is displayed as supplied. Local `go run` builds display `dev`.

The commit SHA comes from the embedded release value, then Go's `vcs.revision`, or falls back to `unknown`. The CLI does not infer a SHA from a Go pseudo-version. Builds without VCS information, such as those built with `-buildvcs=false`, and some tagged `go install` builds may have no SHA. Missing build information does not make the command fail.

The commit display includes `dirty` only when Go build information has `vcs.modified: true`. This also applies to embedded release versions and an `unknown` SHA. If `vcs.modified` is absent, no `dirty` marker is shown.

The CLI does not invoke `git` at runtime. The SHA identifies the source commit; `dirty` only indicates uncommitted changes at build time. It does not describe those changes, so two builds with the same version and SHA can still differ.
