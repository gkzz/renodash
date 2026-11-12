# Development

## Setup and tests

Go 1.27 or later is required. The development environment uses the Go version pinned in `.config/mise/config.toml`, managed by mise. Install mise before using the `make` commands below.

Run `make install` to preview tool and dependency installation, then `make install-apply` to install them.

```sh
make go-fmt  # Format Go source
make lint    # Check Go formatting, run go vet, and lint English and Japanese docs
make test    # Run tests
make check   # Run all lint checks and Go tests
```

You can also run tests directly with Go:

```sh
go test ./...
```

Tests cover:

- Saved Dashboards and derived group-update fixtures
- Checkbox states and unknown or malformed controls
- Code fences and unrelated sections that the parser must skip
- Input read errors

The Go CLI has no external module dependencies.

Documentation checks use Node.js 22 or later and pnpm 12.15.0. Install the documentation tools with `pnpm install --frozen-lockfile`, then run `make docs-lint`.
See [Writing documentation](writing.md) for the writing guidelines and rule configuration.

GitHub Actions runs the same checks on pushes to main, pull requests, and manual workflow runs.

The CLI entry point is `main.go` at the repository root. Command handling lives in `internal/command`; Dashboard parsing lives in `internal/dashboard`. There is no public Go library API.

See the [fixture documentation](../testdata/README.md) for sources, capture dates, and how derived fixtures were created. See [Releasing](release.md) for publishing instructions.

## Build information

The displayed version uses the first available value in this order:

1. The release version embedded with `-ldflags -X`
2. The main module version in Go build information
3. `dev`

Installing a tag with `go install` displays that tag's version. A pseudo-version installed through mise's Go backend is displayed as supplied. Local `go run` builds display `dev`.

The commit SHA comes from the embedded release value, then Go's `vcs.revision`, or falls back to `unknown`. The CLI does not infer a SHA from a Go pseudo-version. Builds without VCS information, such as those built with `-buildvcs=false`, and some tagged `go install` builds may have no SHA. Missing build information does not make the command fail.

The commit display includes `dirty` only when Go build information has `vcs.modified: true`. This also applies to embedded release versions and an `unknown` SHA. If `vcs.modified` is absent, no `dirty` marker is shown.

The CLI does not invoke `git` at runtime. The SHA identifies the source commit; `dirty` only indicates uncommitted changes at build time. It does not describe those changes, so two builds with the same version and SHA can still differ.
