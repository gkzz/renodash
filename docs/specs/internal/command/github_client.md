# GitHub client

Fetch Issue data through the GitHub REST API.
See [PR transport](github_pr.md) for GraphQL responses, check pages, and HTTP error classification.

## Behavior

### Issue requests

- Send a GET request to `https://api.github.com/repos/OWNER/REPO/issues/NUMBER`.
- Send `X-GitHub-Api-Version: 2022-11-28` and `User-Agent: renodash`.
- Preserve repository casing and format numbers as decimal integers.
- Use standard TLS verification, ignore proxy environment variables, and reject redirects.
- Use HTTP/1.1 over TLS to keep credentials out of Go HTTP/2 debug logs.
- Return the successful response body. The caller checks its metadata.

### Credentials

Check context cancellation before looking up credentials.
Use the first nonempty token from `GH_TOKEN`, `GITHUB_TOKEN`, or `gh auth token --hostname github.com`.
Trim surrounding whitespace.
Resolve the token on the first fetch and reuse it for Issue and linked PR requests.
A rejected token ends the fetch.

### Deadlines and cancellation

The caller sets time limits. The client applies the request context to HTTP requests and saved login lookup.

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

## Implementation and test evidence

The unexported `githubClient` interface in [github_client.go](../../../../internal/command/github_client.go) contains `FetchIssue` and `FetchPR`.
`Run` provides a client with a shared HTTP transport for REST and GraphQL.

- [github_issue_test.go](../../../../internal/command/github_issue_test.go) checks credentials, request headers, HTTP errors, redirects, and context behavior.
- [github_client_test.go](../../../../internal/command/github_client_test.go) checks missing credentials and cancellation during saved login lookup. It also checks that canceled requests discard successful responses.
- [offline_get_test.go](../../../../internal/command/offline_get_test.go) checks command JSON with offline Issue and PR responses, including an actual CLI run without `gh`.
