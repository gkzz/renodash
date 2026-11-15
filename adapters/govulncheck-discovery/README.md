# Official Go API advisory discovery (NOT MERGE)

This standalone experiment wraps `golang.org/x/vuln/scan` at `v1.9.0`.
The official Go API queries its vulnerability database and matches versions.
The adapter supplies module/version queries, converts reported matches, and saves evidence.
It has no custom HTTP client or affected-range evaluator.
It accepts no advisory ID, advisory URL, or selected advisory payload.

## Run

Use Go 1.27 or later. The live command queries `https://vuln.go.dev` through the official API.
From the repository root, create the parent directory:

```sh
mkdir -p local/advisory-discovery
```

Then run from `adapters/govulncheck-discovery`:

```sh
go test ./... -count=1
go vet ./...
go run . -input testdata/target.json \
  -evidence-dir ../../local/advisory-discovery/go-sdk-run
```

Choose a new evidence directory for each run. Existing directories are rejected.
The JSON report goes to stdout and `report.json` in the evidence directory.
Each query saves SDK stdout, diagnostics, a digest, completion time, and normalized assessment.
The saved bytes are API output, not original database HTTP responses.
An embedded API may report `(devel)` as its scanner version; the pinned dependency is recorded separately.
Query failures appear in assessment limitations; an operationally successful command can report `Unknown`.

## Scope

The sample uses `golang.org/x/text` versions `v0.3.6` and `v0.3.7`.
This is a synthetic historical update pair, not a recorded PR or recommended upgrade.
A version fixing one advisory may match other advisories.
Only Go upstream-module identity is selected; target identity remains caller-supplied.

The adapter calls `scan.Command`, then `Start` and `Wait`, in query mode.
It does not launch `govulncheck` as a subprocess or load target project code.
Acquisition and affected-version matching belong to the official implementation.
Only the public Go API is imported; upstream `internal` packages are not imported or copied.
The dependency module and checksums are pinned in this isolated module.

`Affected` preserves SDK-reported matches even alongside partial failures.
Incomplete queries yield `Unknown` when no confirmed match remains.
`NoMatchInCheckedSources` covers only this database snapshot and supplied module/version.
Results do not establish complete coverage, exploitability, symbol reachability, environment applicability, or transitive coverage.
This experiment does not change renodash, production workflows, merge policy, or Node.js runtime support.

See the [behavior specification](../../docs/specs/adapters/govulncheck-discovery.md) and [saved evidence notes](testdata/README.md).

## Sources

- [Public scan API](https://pkg.go.dev/golang.org/x/vuln@v1.9.0/scan).
- [Pinned public API implementation](https://github.com/golang/vuln/blob/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5/scan/scan.go).
- [Pinned query implementation](https://github.com/golang/vuln/blob/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5/internal/scan/query.go), inspected as a reference only.
- [Go vulnerability database](https://go.dev/doc/security/vuln/database).

Similar OSS comparisons and live verification records are tracked in Issue #80.

## Third-party licenses

The official `golang.org/x/vuln` API is distributed under its BSD-style license.
See [third-party notices](THIRD_PARTY_NOTICES.md) for the pinned dependency's notice and distribution requirements.
Database entries have a separate CC BY 4.0 license, described in the [data attribution](testdata/README.md#data-attribution-and-license).
The repository's MIT license applies to its original code, without replacing third-party licenses.
