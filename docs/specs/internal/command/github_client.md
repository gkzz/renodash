# GitHub client

Fetch Issue data through the GitHub REST API.
See [PR transport](github_pr.md) for GraphQL responses, check pages, and HTTP error classification.

## Behavior

### Issue requests

- Send a GET request to `https://api.github.com/repos/OWNER/REPO/issues/NUMBER`.
- Preserve repository casing and format numbers as decimal integers.
- Use standard TLS verification, ignore proxy environment variables, and reject redirects.
- Return the successful response body. The caller checks its metadata.

### Issue credentials

Check context cancellation before looking up credentials.
Use the first nonempty token from `GH_TOKEN`, `GITHUB_TOKEN`, or `gh auth token --hostname github.com`.
Trim surrounding whitespace.
Look up the token only when fetching an Issue and reuse it within the command.
A rejected token ends the fetch.

### Deadlines and cancellation

The caller sets time limits. The client applies the request context to HTTP and `gh` operations.

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

### Issue errors

| Condition | Error |
| --- | --- |
| Missing credentials or HTTP 401 | Authentication guidance |
| HTTP 404 | `issue not found or inaccessible` |
| Other HTTP errors | Issue request failure with numeric status |
| Transport or response-read failure | Generic request or read error |
| Canceled or expired context | The context error |

Issue errors omit credentials, response headers, and error response bodies.

### PR process errors and diagnostics

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

The unexported `githubClient` interface contains `FetchIssue` and `FetchPR`.
`Run` provides a client with an HTTP transport for Issues and the existing GitHub CLI client for PRs.

- [github_issue_test.go](../../../../internal/command/github_issue_test.go) checks credentials, request headers, HTTP errors, redirects, and context behavior.
- [github_client_test.go](../../../../internal/command/github_client_test.go) checks process errors, missing credentials, and cancellation. It also checks that canceled requests discard successful responses.
- [offline_get_test.go](../../../../internal/command/offline_get_test.go) checks command JSON with an injected Issue response and captured PR fixtures.
