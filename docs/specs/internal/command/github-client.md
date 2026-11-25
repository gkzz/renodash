# GitHub CLI client

The client fetches Issue and pull request (PR) data through the authenticated GitHub CLI (`gh`).
It reads HTTP and GraphQL responses. The caller validates PR metadata and parses the Dashboard.
See [Dashboard fetching](get.md) for Issue validation and output behavior.

## Behavior

### Operations

| Operation | Command |
| --- | --- |
| Issue | `gh api repos/OWNER/REPO/issues/NUMBER` |
| PR | `gh api graphql --hostname github.com --include` with a fixed query and target variables |

- Preserve repository casing and format numbers as decimal integers.
- Pass arguments directly to `gh`, without a shell.
- Return Issue stdout unchanged; the caller validates its metadata.
- For PRs, read HTTP headers and extract PR data from the GraphQL response.
- Fetch every page of check results.
- Require exactly one commit on every page; otherwise return `invalid_metadata`.
- Allow a null check rollup on the initial page. Require a non-null rollup on continuation pages; otherwise return `invalid_pagination`.
- Normalize checks to the existing `gh pr view --json` format: flatten `workflowName`, rename StatusContext `createdAt` to `startedAt`, and omit `description` and `checkSuite`.
- Match GitHub CLI timestamp encoding, including `0001-01-01T00:00:00Z` for null check timestamps.
- Reject GraphQL errors, missing or repeated page cursors, and head SHA changes between pages.
- Require the queried commit to match the PR head SHA. This does not prove required-check coverage or merge readiness.

[pr.go](../../../../internal/command/pr.go) handles PR links, duplicate numbers, and metadata validation.

### Deadlines and cancellation

The caller sets time limits; the client passes its context to `exec.CommandContext`.

| Scope | Time limit |
| --- | --- |
| Issue request | 30 seconds |
| Each PR attempt, including check pages | 30 seconds |
| Dashboard and all PR attempts together | 5 minutes |

- Each request inherits parent cancellation and earlier deadlines.
- Release each request context with `defer cancel()`.
- Reject data returned after cancellation or deadline expiry, even if the client reports success.
- Return client errors to the caller, which adds fetch context using `%w`.

See [Retries and partial results](retrieval.md) for retry waits, signal handling, and partial output.

## Output and failures

### Structured PR errors

Read status and headers from `gh api --include`, even when the process fails.
Use these values for HTTP classification; do not match stderr text.

| Response | Category |
| --- | --- |
| HTTP 401 | `authentication` |
| HTTP 403 without rate-limit evidence | `permission_denied` |
| HTTP 404 | `not_found_or_inaccessible` |
| HTTP 429, or HTTP 403 with rate-limit headers or a structured rate-limit error | `rate_limit` |
| HTTP 5xx | `http_5xx` |
| Other unsuccessful HTTP status | `http_error` |
| GraphQL `RATE_LIMITED` or a secondary rate-limit error message, including HTTP 200 | `rate_limit` |
| Other GraphQL errors | `graphql_error` |
| Invalid response JSON | `decode_error` |
| Missing or null commits, or missing or multiple commit nodes | `invalid_metadata` |
| Missing or null check nodes, or a missing or unsupported check node type | `invalid_metadata` |
| Head SHA mismatch or change between pages | `head_changed` |
| Missing or null check page metadata (including `hasNextPage`), or missing or repeated page cursor | `invalid_pagination` |

Inspect JSON error types and secondary rate-limit messages before classifying HTTP 403 as a permission error.
Do not classify ordinary permission messages as rate limits.

For rate-limit waits, use:

- `Retry-After`: seconds or an HTTP date.
- Otherwise, `X-RateLimit-Reset` when `X-RateLimit-Remaining` is zero.

Do not retry rate limits with unknown waits, other GraphQL errors, or unknown process failures.
Structured errors never include raw headers, response bodies, or diagnostics.

Select the same check fields as [GitHub CLI v2.97.0](https://github.com/cli/cli/blob/v2.97.0/api/query_builder.go).
Authentication and API requests still run through `gh`.

### Process errors and diagnostics

| Condition | Error |
| --- | --- |
| Missing executable | `gh is not installed or not in PATH` |
| Stderr contains `authentication`, `not logged in`, or `http 401` | `gh authentication failed; run gh auth login` |
| Stderr contains `http 404` or `not found` | `issue not found or inaccessible` |
| Other process failure | `gh command failed: <error>`, wrapping the original process error |
| Process failure with a canceled or expired context | The context error |

Diagnostic matching ignores case. Authentication checks precede not-found checks.
The existing not-found message also applies to PR requests.
Raw stderr is captured for classification but never copied into user-facing errors, because it may contain secrets.
Response validation errors remain separate from process errors.

## Implementation and test evidence

### Dependency injection

The unexported `githubClient` interface contains `FetchIssue` and `FetchPR`.
Both operations are needed by the `get` workflow, so one interface keeps dependency passing small.
`Run` supplies the stateless `ghClient{}` to `runWithClient`.
Tests pass a local `fakeGitHubClient` through the same command path without replacing package variables.
No constructor or dependency container is needed.

### Source and tests

Implementation: [github_client.go](../../../../internal/command/github_client.go).
Transport tests: [github_client_test.go](../../../../internal/command/github_client_test.go).

- `TestGHClientArguments` checks exact Issue arguments and unchanged stdout bytes through a fake executable.
- `TestGHPRPaginationAndArguments` checks PR arguments, check pagination, and response unwrapping.
- `TestGHPRRejectsIncompleteEvidence` checks head changes, cursor failures, and GraphQL errors.
- `TestStructuredHTTPFailures` checks HTTP status and rate-limit headers.
- [Retrieval tests](../../../../internal/command/retrieval_test.go) check retry recovery, limits, cancellation, and partial errors.
- `TestGHClientErrors` checks diagnostic mappings for both operations, missing executables, and cancellation before execution.
- `TestGHClientCancelsRunningProcess` waits for a fake process startup signal before canceling Issue and PR operations.
  It checks the context error, discarded partial stdout, and process connection closure within bounded waits.
- `TestExecuteGHFailure` checks process error wrapping and omission of raw diagnostics.
- `TestFetchRejectsLateSuccess` checks both request helpers with cancellation during the call and an expired parent deadline.
- `TestRunUsesGHClient` checks the production entry point through a fake executable.
- `TestRunGHFailureLeavesStdoutEmpty` checks the exit code, empty stdout, and sanitized CLI error on authentication failure.
- Command tests cover output, validation, PR deduplication, independent request budgets, and parent cancellation scenarios.

Tests are offline. They do not establish live authentication or GitHub availability.
The subprocess tests use a Go test executable through [fakegh_test.go](../../../../internal/command/fakegh_test.go), without shell scripts.
See [command subprocess tests](../../../testing.md#command-subprocess-tests) for helper naming and platform validation limits.
The running-process test uses a loopback TCP connection for synchronization, without contacting external services or using arbitrary sleeps.
These tests do not wait for a real 30-second request expiry or establish cleanup of descendant processes.
The existing time and Dashboard parser test hooks remain package variables outside this refactoring's scope.
