# Official Go API advisory discovery spike

This NOT MERGE experiment wraps the public `golang.org/x/vuln/scan` API.
The pinned module version is `v1.9.0`. Query mode accepts a module and version without advisory IDs.
The official implementation owns database acquisition and affected-version matching.
The adapter delegates HTTP requests, version comparisons, and affected-range evaluation to that API.
It does not change the renodash CLI, production workflows, or merge policy.

## Input and validation

Accept one target JSON object with `ecosystem`, `distribution`, `module`, `old_version`, and `proposed_version`.
Reject unknown fields, trailing JSON, missing identity, and input larger than 64 KiB.
The static registry supports only ecosystem `Go` and distribution `upstream-module`.
Require a module path with a domain and path component. Standard library and runtime targets are unsupported.
Reject option-like or compound query inputs before calling the API.
Delegate version syntax and affected-range matching to the official implementation.
Identity remains caller-supplied, not verified against a PR or dependency graph.

## Behavior

Call `scan.Command` with `-mode=query`, `-json`, the fixed database `https://vuln.go.dev`, and `module@version`.
Use `Start` and `Wait`; do not launch a separate executable or target code.
Query old and proposed versions separately with a 30-second deadline per query and 60-second operation deadline.
Limit captured stdout to 8 MiB and diagnostics to 1 MiB per query.
Do not overwrite an existing evidence directory.
Save API stdout, diagnostics, SHA-256 digest, query, UTC completion time, and reported API configuration.
The saved stream is SDK output, not the original database HTTP response bytes.
Database retrieval limits, caching, and redirects remain upstream implementation behavior.

Accept the `v1.0.0` JSON protocol and query-mode configuration for the expected database.
Interpret returned `osv` records as matches reported by the official query implementation.
Check exact module identity in returned records, preserve native record JSON and aliases, and ignore withdrawn records.
Keep confirmed matches when a later stream or query failure occurs.
Source failures, output limits, invalid streams, conflicting IDs, and missing configuration remain limitations.
Without a confirmed match, unresolved limitations produce `Unknown`.
A completed empty query produces `NoMatchInCheckedSources` for this database snapshot and exact module scope only.
An empty result never establishes safety, freshness, or complete database coverage.
Do not merge independent records solely because they share an alias.

## Output and failures

Write the common JSON report to stdout and `report.json` in the evidence directory.
Include the target, SDK version, database, selection reason, two assessments, assessment time, evidence files, and coverage limits.
Input, evidence-write, and output errors return nonzero status.
Query failures remain visible as limitations, independently of the process exit status.
Do not infer exploitability, call reachability, environment applicability, transitive coverage, or Node.js runtime support.
No result approves, blocks, or merges a PR.

## Implementation and test evidence

See [the isolated module](../../../adapters/govulncheck-discovery/).
Run `go test ./... -count=1` and `go vet ./...` inside that module.
Tests inject SDK output for protocol, identity, partial failure, evidence preservation, and unsupported inputs.
A canceled-context test exercises the real public API without network access.
Saved query outputs support offline reproduction. Live verification and OSS comparisons belong in Issue #80.
