# GitHub CLI client

The client fetches Issue and pull request (PR) JSON through the authenticated GitHub CLI (`gh`).
It does not check response metadata or parse Dashboards.
See [Dashboard fetching](get.md) for Issue validation and output behavior.

## Behavior

### Operations

| Operation | Command |
| --- | --- |
| Issue | `gh api repos/OWNER/REPO/issues/NUMBER` |
| PR | `gh api graphql --hostname github.com --include` with a fixed query and target variables |

Repository casing is preserved. Numbers use decimal integer formatting.
Arguments are passed to the executable without a shell.
Issue success returns stdout unchanged. The PR adapter reads HTTP headers and unwraps the GraphQL response before metadata validation.
It selects the existing PR fields and check fields, following all check-context pages.
It rejects GraphQL errors, repeated or missing pagination cursors, and a changing head SHA across pages.
The queried commit must match the PR head SHA; this does not establish required-check coverage or merge readiness.
PR link parsing, deduplication, and metadata validation remain in [pr.go](../../../../internal/command/pr.go).

### Deadlines and cancellation

The calling layer owns request budgets; the concrete client does not create its own timeouts.

- `fetchIssue` in [get.go](../../../../internal/command/get.go) gives the Issue request a 30-second timeout.
- `fetchPR` in [pr.go](../../../../internal/command/pr.go) gives each unique PR request a separate 30-second timeout.
- Both helpers derive contexts from their parent and release resources with `defer cancel()`.
- Parent cancellation and earlier parent deadlines apply to the child request.
- The Issue request and all PR attempts share the command's five-minute budget and signal cancellation context.
- See [Resilient retrieval](retrieval.md) for retry limits, backoff, and partial-result behavior.
- After a successful client return, both helpers check the request context. They reject data returned after cancellation or deadline expiry.
- Client errors are returned unchanged by the helpers. The command adds the existing fetch context using `%w`.

`executeGH` uses `exec.CommandContext` to stop the process when its context ends.
A final fetch failure prevents result JSON unless `--allow-partial` applies to that PR.

## Output and failures

### Structured PR errors

The PR adapter reads status and headers from `gh api --include`, including on process failure.
It classifies HTTP 401, 403, 404, 429, and 5xx without inspecting stderr text.
Rate limits use `Retry-After` seconds or HTTP date, or `X-RateLimit-Reset` when remaining quota is zero.
A GraphQL `RATE_LIMITED` error also identifies a rate limit, even with HTTP 200.
A known wait is required before retrying a rate limit; unknown waits fail conservatively.
Other GraphQL errors and unknown process failures are not retried.
Raw headers, error bodies, and diagnostics are never included in structured errors.

The same check fields are selected as [GitHub CLI v2.97.0](https://github.com/cli/cli/blob/v2.97.0/api/query_builder.go).
No direct GitHub HTTP client or authentication implementation is added.

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
