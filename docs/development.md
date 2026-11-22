# Development

## Setup and tests

Go 1.27 or later is required. The development environment uses tool versions pinned in [the mise configuration](../.config/mise/config.toml).
Install [mise](https://mise.jdx.dev/getting-started.html) before using the `make` commands below. Run all commands in this guide from the repository root.

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

The Go CLI has no external runtime module dependencies. Tests use `github.com/google/go-cmp/cmp` for structural comparisons.

See [Writing documentation](writing.md#run-the-checks) for documentation lint commands and rules.

GitHub Actions runs the same checks on pushes to main, pull requests, and manual workflow runs.

## Dependency vulnerability checks

The [Lint workflow](../.github/workflows/lint.yml) also runs `pnpm audit` and `govulncheck ./...` as separate jobs on the same events.
These audits are separate from `make lint` and `make check`.
`pnpm audit` includes development dependencies and uses the default severity threshold without filters.
`govulncheck` checks Go dependencies and the standard library for known vulnerabilities, analyzing whether vulnerable functions are reachable.
Findings that meet each tool's default failure criteria fail its job.

To investigate an audit failure locally, install the project tools and dependencies with `make install-apply`.
Then run these commands from the repository root, with the mise-managed Go and pnpm on your `PATH`:

```sh
pnpm audit
go install golang.org/x/vuln/cmd/govulncheck@e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5 # v1.9.0
"$(go env GOPATH)/bin/govulncheck" ./...
```

The Go installation command uses the default `GOBIN`; if you set `GOBIN`, run `govulncheck` from that directory instead.
The audits require network access to vulnerability databases.
Review each finding before updating dependencies or considering an exception.

## Code layout

| Path | Responsibility |
| --- | --- |
| `main.go` | CLI entry point |
| `internal/command` | Command arguments, fetching, output, and exit handling |
| `internal/dashboard` | Dashboard parsing and research context extraction |
| `internal/summary` | Offline Actions Summary rendering |
| `internal/integration` | Offline CLI and summary integration tests |

There is no public Go library API.

See the [fixture documentation](../testdata/README.md) for sources, capture dates, and how derived fixtures were created. See [Releasing](release.md) for publishing instructions.

## Build information

See the [version and build information specification](specs/internal/command/version.md) for version selection, commit SHA, and dirty markers.

## Changing behavior

Use the [specification and TDD workflow](tdd.md) for new behavior and bug fixes.
It explains cancellation handling and regression tests for the GitHub client introduced in PR #60.
See [Testing strategy](testing.md) for test conventions and [GitHub Issues](https://github.com/gkzz/renodash/issues) for remaining documentation and test gaps.
