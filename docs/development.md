# Development

## Setup and tests

Go 1.27 or later is required. The development environment uses the Go version pinned in `.config/mise/config.toml`, managed by mise. Install mise before using the `make` commands below.

Run `make install` to preview tool and dependency installation, then `make install-apply` to install them.

```sh
make go-fmt  # Format Go source
make lint    # Check formatting and run go vet
make test    # Run tests
make check   # Check formatting, run static analysis, and test
```

You can also run tests directly with Go:

```sh
go test ./...
```

Tests cover saved Dashboards and derived group-update fixtures, checkbox states, preservation of unknown and malformed controls, exclusion of code fences and unrelated sections, and read errors. There are no external module dependencies.

GitHub Actions runs the same checks on pushes to main, pull requests, and manual workflow runs.

The CLI entry point is `main.go` at the repository root. Command handling lives in `internal/command`; Dashboard parsing lives in `internal/dashboard`. There is no public Go library API.

See the [fixture documentation](../testdata/README.md) for sources, capture dates, and how derived fixtures were created. See [Releasing](release.md) for publishing instructions.

## Build information

The displayed version uses the first available value: the release version embedded with `-ldflags -X`, the main module version in Go build information, or `dev`. Installing a tag with `go install` displays that tag's version. A pseudo-version installed through mise's Go backend is displayed as supplied. Local `go run` builds display `dev`.

The commit SHA comes from the embedded release value, then Go's `vcs.revision`, or falls back to `unknown`. The CLI does not infer a SHA from a Go pseudo-version. Builds without VCS information, such as those built with `-buildvcs=false`, and some tagged `go install` builds may have no SHA. Missing build information does not make the command fail.

The commit display includes `dirty` only when Go build information has `vcs.modified: true`. This also applies to embedded release versions and an `unknown` SHA. If `vcs.modified` is absent, no `dirty` marker is shown.

The CLI does not invoke `git` at runtime. The SHA identifies the source commit; `dirty` only indicates uncommitted changes at build time. It does not describe those changes, so two builds with the same version and SHA can still differ.
