# Offline PR metadata fixtures

Captured from public repository `gkzz/renodash` at 2026-11-12T23:51:41Z using:

```sh
gh pr view NUMBER --repo gkzz/renodash --json number,url,state,title,isDraft,headRefOid,updatedAt,statusCheckRollup
```

- [#51](https://github.com/gkzz/renodash/pull/51): captured MERGED.
- [#52](https://github.com/gkzz/renodash/pull/52): captured MERGED.
- [#53](https://github.com/gkzz/renodash/pull/53): captured OPEN.
- [#30](https://github.com/gkzz/renodash/pull/30): captured CLOSED.

PR JSON files are captured responses without state or check edits. #51 and #52
are merged, #53 is open, and #30 is closed without merging. #51–#53 include
successful CheckRun and StatusContext observations and a skipped publish job.
#30 also captures failed label checks; pending checks use synthetic unit cases.
These observations are snapshots, not current merge assessments.

`dashboard-issue.json` is synthetic, including its body and update timestamp.
It uses real repository, issue, and PR identifiers; it is not a historical
Dashboard capture. Its deliberate order is #53, #51, #30, #52, #53, testing
first-seen order and duplicate lookup suppression. #30 is a state coverage
fixture, not a dependency update candidate observed on the live Dashboard.

The saved responses have no merge fields, so the CLI reports `UNKNOWN`.
A separate offline test adds made-up `MERGEABLE` and `BLOCKED` values without changing the saved files.

The test adapter adds GraphQL and HTTP response structures around the saved PR data.
It does not change the saved files.

The offline test also synthesizes empty checks, malformed JSON, missing metadata,
and lookup failures in memory. Summary unit tests synthesize failed and pending
checks. None of these are presented as captured GitHub observations.

Responses cannot reproduce historical API timing, permissions, or platform
behavior. Refresh fixtures deliberately: review source URLs, capture time,
metadata, checks, and this provenance together. Do not include credentials or
private data. The CLI's fetched_at records test execution time, not capture time.

## Run locally

```sh
RENODASH_OUTPUT=/tmp/renodash.json go test -count=1 ./internal/command
go run ./internal/tools/fake-actions-summary < /tmp/renodash.json
```

The command test supplies the Issue fixture through an HTTP transport and
serves PR responses by running a fixture-only `gh` executable.
The Actions workflow also checks `parse` and the `get` error for missing credentials with an installed CLI.

The `pr-*.graphql.json` fixtures are synthetic GraphQL responses derived from the saved CLI exports.
They nest workflow names under `checkSuite.workflowRun.workflow` and use `createdAt` and a synthetic `description` for status contexts.
The offline adapter uses their check nodes; the original exports remain independent expected output.
