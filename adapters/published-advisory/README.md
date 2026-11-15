# Published advisory evidence adapter spike

**NOT MERGE investigation spike.**

This report compares dependency versions against publicly disclosed security advisories.
It does not discover vulnerabilities, assess exploitability, or establish that a version is safe.

This standalone Go tool tests a source adapter boundary for dependency evidence.
This tool is separate from the renodash CLI and does not change merge rules.

## Run

Use Go 1.27 or later. From this directory:

```sh
go mod download
go test ./...
go run . < testdata/pr53.json
go run . -format markdown < testdata/pr53.json > /tmp/published-advisory-evidence.md
```

The last command writes Markdown suitable for a GitHub Actions Job Summary.
It does not upload the file or change a workflow.
The root module's `go test ./...` does not run this nested module's tests.

## Boundary

```text
Saved external payload + target identity + provenance
  -> source adapter
  -> common version assessments
  -> aggregation
  -> JSON or Markdown
```

`EvidenceAdapter` owns provider-specific interpretation.
The first adapter reads the Node.js security working group's core index.
Aggregation and rendering use common assessments without interpreting provider payloads.
Static registration is enough for this experiment; no dynamic plugin system is introduced.
The spike uses a separate Go module so its semver dependency does not enter the renodash core.

External acquisition remains a caller responsibility, like other client boundaries around external services.
The tool reads snapshots from standard input without network access or subprocess execution.
It never executes PR changes.
Target identity, inspected files, and versions are caller-supplied evidence; this spike does not extract or authenticate them.
The existing GitHub client and PR parser remain unchanged.

## Assessment scope

Each adapter returns `Affected`, `NoMatchInCheckedSources`, or `Unknown` for each version.
Confirmed matches remain visible alongside failures from other records or sources.
Unknown adapters, invalid ranges, unsupported identities, and retrieval failures keep limitations.
Assessment states never change the command's exit code into a vulnerability gate.

For Node.js core data, the adapter checks `vulnerable` and excludes versions matching `patched`.
It uses [Masterminds semver](https://github.com/Masterminds/semver) constraints rather than implementing a range parser.
Stable versions only are supported. Prereleases produce `Unknown`.
Cross-library equivalence with npm semver is not established for every possible range.
Supported target environments are `linux`, `darwin`, and `win32`.
Input aliases `macos` and `windows` normalize to `darwin` and `win32`.
Unknown inputs, distribution names such as `ubuntu`, and input `all` produce `Unknown`.
Source `all` matches every supported target; it does not extend distribution support.
Unrecognized source identifiers preserve uncertainty when no explicit match exists.
These checks compare record labels; they do not establish cross-platform execution support.
Missing fields are treated conservatively as incomplete evidence, even for older records.

A source URL and revision are provenance labels supplied by the caller, not authentication of the payload.
No-match results cover only the supplied records, version, environment, and snapshot time.
They do not establish complete advisory coverage, freshness, or safety.
Musl/unofficial builds and other platforms in a lockfile require separate identification and assessment.
Live CI collection, automatic fetching, other adapters, and production integration remain outside this spike.

See the [behavior contract](../../docs/specs/adapters/published-advisory.md) and [fixture provenance](testdata/README.md).

## Public reports

Follow the [public reporting boundaries](../../docs/dependency-review.md#public-reporting-boundaries).
A matching proposed version is evidence about a published range, not proof that this repository or a deployment is exploitable.
The maintainer decides whether to merge and performs merges manually.
The tool does not redact secrets or decide whether disclosure is permitted; review inputs before publication.
Source publication and retrieval times can differ. A completed query does not establish comprehensive coverage.
The tool neither fetches nor authenticates advisory data and does not independently identify the PR's product or versions.
Official binaries, npm packages, and distribution-specific `nodejs` packages require distinct identities.
Saved snapshots may be stale, incomplete, or historically inconsistent.

See the [Issue #80 findings draft](testdata/issue80-findings.md) for experimental results and unresolved decisions.

## Offline Job Summary experiment

The summary mode compares a snapshot against an independent context before publishing findings.
The Go `prepare-fixture` subcommand adapts saved fixtures into scenario inputs separately from the existing offline CLI summary.
The workflow passes its scenario, tested SHA, executed-code SHA, and observed CI outcome through flags.
No Python runtime or network lookup is needed for fixture preparation.

Generate an affected-version example from this directory:

```sh
go run . prepare-fixture -scenario affected \
  -ci-sha aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  -code-sha bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
  -ci-outcome success -output-dir /tmp/advisory-fixture
go run . -format markdown \
  -summary-context /tmp/advisory-fixture/advisory-context.json \
  -snapshot /tmp/advisory-fixture/advisory-snapshot.json
```

These example SHAs are synthetic placeholders.
`-fixture-dir` defaults to `testdata`; use it to select another saved fixture directory.
The command writes generated inputs to `-output-dir`, creating the directory when needed.
The `missing` scenario removes any existing generated `advisory-snapshot.json` at that location to prevent accidental reuse.
Unsupported scenarios, invalid metadata, or invalid fixtures fail preparation before output is written.
Use a dedicated output directory for these generated files.

To render the saved context directly, run from this directory:

```sh
go run . -format markdown \
  -summary-context testdata/summary-context.json \
  -snapshot testdata/pr53-cve.json > /tmp/advisory-summary.md
```

The saved context declares PR #53's historical identity separately from the advisory snapshot.
Its CI and executed-code SHAs are synthetic placeholders, not claims about a real CI run.
Supply reviewed context values for other local experiments; the tool does not fetch or authenticate them.
All target fields and inspected files must match; file order can differ.
A mismatch, missing snapshot, or malformed snapshot produces `Unknown` without snapshot findings.
Failed CI preserves confirmed matches from a matching snapshot.
Every summary identifies itself as an offline fixture experiment and separates the expected head, CI, and executed-code SHAs.
Output above 512 KiB becomes a bounded `Unknown` notice rather than partial evidence.
Only valid, environment-applicable ranges are listed as checked ranges, including non-matches.
Different range strings for one CVE require manual review; the tool does not resolve their semantic differences.

The `advisory-spike` job in the [PR workflow](../../.github/workflows/renodash.yml) exercises nine saved-data scenarios.
It executes the triggering PR's merge revision, including changed adapter code and tests, with `permissions: {}` and persisted checkout credentials disabled.
This is an unprivileged test of PR code, not a trusted assessment of that PR's dependency changes.
The historical PR #53 fixture remains separate from the triggering PR.
The job does not use explicitly supplied secrets, PR comments, or merge actions.
Summary publication uses `always()` and a fixed fallback when the tool or context is unavailable.
Runner loss or cancellation can still prevent publication.

The failed-CI scenario uses a separate intentional failure step with `continue-on-error`.
Its `outcome` remains `failure` in the context, while expected failure does not make the PR check fail.
Actual adapter test failures still fail the job; summary generation attempts to preserve their outcome.
This remains a NOT MERGE artifact; local reproduction does not verify GitHub execution or upload behavior.
The existing `review` job remains a separate CLI display experiment.
