# GitHub CLI client

Run authenticated GitHub CLI (`gh`) processes and fetch Issue data.
See [PR transport](github_pr.md) for GraphQL responses, check pages, and HTTP error classification.

## Behavior

### Operations

- Fetch Issues with `gh api repos/OWNER/REPO/issues/NUMBER`.
- Preserve repository casing and format numbers as decimal integers.
- Pass arguments directly to `gh`, without a shell.
- Return Issue stdout unchanged; the caller validates its metadata.

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
