# Specification coverage review

This review records the behavior areas present after merged PR #60, building on #55 and #57–#59.
It maps documents to implementation and test evidence without measuring line coverage or proving that every branch is tested.
Use it to choose focused follow-ups for [Issue #56](https://github.com/gkzz/renodash/issues/56).

## Behavior inventory

Paths below are relative to the repository root. Test names identify representative evidence, not exhaustive coverage.

| Behavior area | Specification or current reference | Implementation and test evidence | Assessment |
| --- | --- | --- | --- |
| Issue reference syntax | [Input validation](specs/internal/command/validation.md) | `internal/command/validation.go`; `validation_test.go`, `TestParseIssueRef` | Documented; upstream account-rule questions remain explicit |
| Issue fetching, response validation, output ordering | [Dashboard fetching](specs/internal/command/get.md) | `internal/command/get.go`; `get_test.go`, `TestGet`, `TestMatchesIssueURL`, warning and timeout tests | Documented; validation and write-error gaps remain |
| GitHub transport and request cancellation | [GitHub CLI client](specs/internal/command/github-client.md) | `internal/command/github_client.go`, `pr.go`, `get.go`; `github_client_test.go`, `TestGHClientErrors`, `TestFetchRejectsLateSuccess` | Documented; running-process cancellation and portability need follow-up |
| Dashboard controls and output collections | [JSON and parsing](specs/internal/dashboard/output-format.md) | `internal/dashboard/parser.go`, `dashboard.go`; `parser_test.go`; `internal/command/run_test.go`, `TestFixtures`, `TestEmptyDashboard` | Documented; no claim of exhaustive Markdown coverage |
| Research extraction and provenance | [Research context](specs/internal/dashboard/output-format.md#research-context-for-agents) | `internal/dashboard/research.go`; `research_test.go`, `TestResearchEvidence`, `TestGroupedResearchEvidence` | Documented; unsupported title forms remain outside the contract |
| Linked PR metadata | [Linked PR metadata](specs/internal/dashboard/output-format.md#linked-pull-request-metadata), [GitHub client](specs/internal/command/github-client.md) | `internal/command/pr.go`; `pr_test.go`, `TestLinkedPRNumber`, `TestPullRequestStatesAndInvalidMetadata` | Output documented; detailed link and response validation rules need expansion |
| Command dispatch, help, parse flags, and exit handling | [Usage](usage.md), [output conventions](specs/internal/dashboard/output-format.md#output-conventions) | `internal/command/run.go`; `run_test.go`, `TestErrors`, `TestHelp`, `TestWriteError`; `internal/integration/offline_test.go` | Partially documented across guides; no dedicated command contract |
| Version and build information | [Version specification](specs/internal/command/version.md) | `internal/command/version.go`; `version_test.go` | Dedicated specification and tests exist |
| Offline Actions Summary rendering | [Fixture provenance](../internal/summary/testdata/README.md), [development guide](development.md) | `internal/summary/render.go`; `render_test.go`, `TestGolden`, `TestURLs`, `TestIOErrors`; `internal/integration/summary_test.go` | Tests and fixtures exist; no dedicated renderer specification |

## Remaining work

These unchecked items are follow-up work, not guarantees provided by the current tests.
Update this list or link a focused issue when work starts; update the corresponding specification when it finishes.

### Missing or partial specifications

- [ ] Add command behavior rules under `docs/specs/internal/command/` for dispatch, help, parse flags, diagnostics, and exit codes.
  Describe `parse` warning behavior separately from `get`; their current formatting differs.
- [ ] Expand linked PR rules to cover accepted link forms, repository matching, rejected URLs, number bounds, and response metadata validation.
  Record what the implementation validates rather than implying strict timestamp validation for PR metadata.
- [ ] Add `docs/specs/internal/summary/` documentation for input validation, escaping, link rules, display limits, check fallbacks, and output failures.
  Link the independent golden fixtures and offline integration tests.

### Known test gaps

- [ ] Add focused `get` tests for missing or malformed `updated_at`, missing or null bodies, empty query markers, and URL user information.
- [ ] Add a `get` stdout write-failure test. Existing `TestWriteError` covers the offline `parse` path.
- [ ] Test cancellation after a fake `gh` process has started, using a deterministic startup signal and bounded cleanup.
- [ ] Replace or isolate shell-based fake executables before claiming Windows support for command tests.
  A successful local test run establishes behavior only on that host.

The input validation specification also records unresolved upstream account naming questions.
Keep those separate from local format guarantees; offline tests cannot establish GitHub account creation rules or availability.

## Maintaining the review

For a behavior change, follow the [TDD workflow](tdd.md) and check the relevant row, specification, and tests together.
Record new gaps as concrete follow-up items rather than marking an entire behavior area as fully covered.
Keep development procedures here and in guides; place behavior contracts under `docs/specs/`.
