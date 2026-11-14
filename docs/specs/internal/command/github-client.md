# GitHub CLI client

The client fetches Issue and pull request (PR) JSON through the authenticated GitHub CLI (`gh`).
It does not check response metadata or parse Dashboards.
See [Dashboard fetching](get.md) for Issue validation and output behavior.

## Behavior

### Operations

| Operation | Command |
| --- | --- |
| Issue | `gh api repos/OWNER/REPO/issues/NUMBER` |
| PR | `gh pr view NUMBER --repo OWNER/REPO --json number,url,state,title,isDraft,headRefOid,updatedAt,statusCheckRollup` |

Repository casing is preserved. Numbers use decimal integer formatting.
Arguments are passed to the executable without a shell.
On success, return stdout bytes unchanged. The caller decodes and validates the JSON.
PR link parsing, deduplication, and metadata validation remain in [pr.go](../../../../internal/command/pr.go).

### Deadlines and cancellation

The calling layer owns request budgets; the concrete client does not create its own timeouts.

- `fetchIssue` in [get.go](../../../../internal/command/get.go) gives the Issue request a 30-second timeout.
- `fetchPR` in [pr.go](../../../../internal/command/pr.go) gives each unique PR request a separate 30-second timeout.
- Both helpers derive contexts from their parent and release resources with `defer cancel()`.
- Parent cancellation and earlier parent deadlines apply to the child request.
- PR requests start from a separate background context in `executeGet`. The Issue deadline does not limit them.
- After a successful client return, both helpers check the request context. They reject data returned after cancellation or deadline expiry.
- Client errors are returned unchanged by the helpers. The command adds the existing fetch context using `%w`.

`executeGH` uses `exec.CommandContext` to stop the process when its context ends.
A fetch failure prevents successful JSON output.

## Output and failures

### Errors and diagnostics

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

### Implementation and coverage

Implementation: [github_client.go](../../../../internal/command/github_client.go).
Transport tests: [github_client_test.go](../../../../internal/command/github_client_test.go).

- `TestGHClientArguments` checks exact Issue and PR arguments and unchanged stdout bytes through a fake executable.
- `TestGHClientErrors` checks diagnostic mappings for both operations, missing executables, and cancellation before execution.
- `TestExecuteGHFailure` checks process error wrapping and omission of raw diagnostics.
- `TestFetchRejectsLateSuccess` checks both request helpers with cancellation during the call and an expired parent deadline.
- `TestRunUsesGHClient` checks the production entry point through a fake executable.
- `TestRunGHFailureLeavesStdoutEmpty` checks the exit code, empty stdout, and sanitized CLI error on authentication failure.
- Command tests cover output, validation, PR deduplication, independent request budgets, and parent cancellation scenarios.

Tests are offline. They do not establish live authentication or GitHub availability.
The subprocess tests use a Go test executable through [fakegh_test.go](../../../../internal/command/fakegh_test.go), without shell scripts.
See [command subprocess tests](../../../testing.md#command-subprocess-tests) for helper naming and platform validation limits.
These tests do not wait for a real 30-second expiry or test cancellation of an already running subprocess.
The existing time and Dashboard parser test hooks remain package variables outside this refactoring's scope.
