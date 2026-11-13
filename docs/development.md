# Development

## Setup and tests

Go 1.27 or later is required. The development environment uses the Go version pinned in `.config/mise/config.toml`, managed by mise. Install mise before using the `make` commands below.

Run `make install` to preview tool and dependency installation, then `make install-apply` to install them.

The offline Actions Summary uses the Go renderer in `internal/summary`.
Summary generation and Go tests do not require jq. The release workflow still
uses jq for its JSON smoke test.

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

See the [version and build information specification](specs/internal/command/version.md) for version selection, commit SHA, and dirty markers.
