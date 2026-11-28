# Native GitHub API spike

[Issue #125](https://github.com/gkzz/renodash/issues/125) tested whether Go can make the GitHub requests that `renodash get` sends through `gh`.

**Recommendation for #116.** Use Go's `net/http` for API requests and keep `gh` sign-in for saved credentials.
The [prototype](../../internal/command/github_http_spike_test.go) runs in tests.

## How it works

```text
GH_TOKEN or GITHUB_TOKEN    Use the first non-empty value
         |
         +-- Neither set    Read the saved token with gh auth token
         |
Go net/http                 Send requests to https://api.github.com
         +-- REST           Read the Dashboard issue
         +-- GraphQL        Read linked PRs and their checks
         |
Existing Go code            Validate results, retry failed PRs, and write JSON
```

### Credential order for #116

| Situation | Expected behavior |
| --- | --- |
| `GH_TOKEN` is set and non-empty | Use it, even if `GITHUB_TOKEN` is also set. |
| Only `GITHUB_TOKEN` is set and non-empty | Use it. |
| Neither is set and `gh` has a saved login | Read the token with `gh auth token --hostname github.com`. |
| `gh` is absent | An environment token is enough. Without one, return a safe authentication error. |
| Saved credentials are unavailable or lookup fails | Return a safe authentication error with login guidance. |
| The chosen token is invalid | Return the failure. Do not try another token. |
| A token needs renewal | Verify how `gh auth token` handles saved credentials. Environment tokens need renewal at their source. |

The HTTP client can send requests without `gh`. Reading a saved login needs `gh`. The spike injects the resolved test token into the client. It leaves production credential lookup to #116.

The prototype rejects redirects, ignores proxy environment variables, and uses standard TLS verification. It sends the token only to the fixed API host. Errors omit tokens, raw HTTP headers, and raw bodies.

## Client comparison

| Client | What we observed | Decision |
| --- | --- | --- |
| `net/http` | The prototype matched `gh` on live data. It gives Go direct access to the HTTP status and headers. | Use it with the existing retry code. It adds no Go module. |
| go-gh v2.17.0 | Keyring lookup called a fake `gh`. The high-level GraphQL call returned the server error text without the HTTP headers. | A lower-level adapter would still be needed to read wait times and report safe errors. |
| go-github v93.0.0 | After a simulated secondary rate limit, the next REST call returned an error without sending a request. | Its own rate-limit checks affect how attempts are counted. A separate GraphQL client would be needed. |

The library tests used fake responses in an isolated Go module. The table records test observations. The source links show how each library implements those paths. See the pinned sources for [go-gh authentication](https://github.com/cli/go-gh/blob/v2.17.0/pkg/auth/auth.go), [go-gh GraphQL](https://github.com/cli/go-gh/blob/v2.17.0/pkg/api/graphql_client.go), and [go-github](https://github.com/google/go-github/blob/v93.0.0/github/github.go).

GraphQL keeps the current PR query and returns check runs and commit statuses in one list. REST exposes them separately. For PR #30 at the same commit, REST returned `mergeable: true` and `mergeable_state: blocked`. GraphQL returned `MERGEABLE` and `BLOCKED`. This comparison covers one PR state. See [REST checks](https://docs.github.com/en/rest/checks/runs), [REST statuses](https://docs.github.com/en/rest/commits/statuses), and the [GraphQL check rollup](https://docs.github.com/en/graphql/reference/commits#statuscheckrollup).

## Results

**Live, 2026-11-28 UTC.** We used a saved `gh` login with both token environment variables unset. The selected fields matched for Issue #5 and all four PRs. Each reader sent one API request per target. The times include setup and come from one run.

| Target | PR state | Mergeability / merge state | Checks | Go | `gh` |
| --- | --- | --- | ---: | ---: | ---: |
| Issue #5 | — | — | — | 409 ms | 448 ms |
| PR #30 | Closed, draft | MERGEABLE / BLOCKED | 10 | 695 ms | 866 ms |
| PR #51 | Merged | UNKNOWN / UNKNOWN | 6 | 605 ms | 740 ms |
| PR #52 | Merged | UNKNOWN / UNKNOWN | 6 | 554 ms | 915 ms |
| PR #53 | Closed | UNKNOWN / UNKNOWN | 8 | 491 ms | 788 ms |

`TestNativeSpikeLive` reads the saved token with `gh auth token` and passes it directly to the Go HTTP client. The test keeps `gh` on `PATH` for the comparison. Credential lookup by the Go client and live requests with `gh` absent from `PATH` remain unverified by this test. GitHub rejected a test token, and the error omitted that token.

A follow-up run added the production `ghClient.FetchPR` path to the same four PR comparisons.
The selected fields and decoded metadata matched across all three paths.

**Offline.** The tests passed for:

- 101 checks across two pages, mixed check types, missing data, and a changed PR head.
- Timeouts, cancellation, retries of the failed PR, and partial JSON.
- Rate limits and errors while reading the response body. See the table below.
- Empty output in strict mode after a failure, and zero requests for invalid stdin.
- Missing tokens, token precedence, redirects, and safe error messages.

| Response | Result when the body read fails |
| --- | --- |
| HTTP 429 with `Retry-After`, including a PR attempt timeout | Keep the rate limit and the server's wait time. |
| HTTP 403 with a complete secondary rate-limit error | Keep the rate limit. |
| HTTP 200 with a complete GraphQL rate-limit error | Keep the rate limit and any server wait time. |
| HTTP 401 or 503 with a truncated body | Keep the authentication or server error. |
| HTTP 200 with a truncated success response | Report a fetch error. |

A PR gets up to three attempts, with 30 seconds per attempt across all pages. The full command has five minutes after input validation. A retry starts at the first page of the failed PR. Go can retry some connection failures at a lower level, so an attempt count is not a wire request count. See the [retry and partial output specification](../specs/internal/command/retrieval.md) and [stdin and JSON specification](../specs/internal/command/get.md).

**Still unverified live.** Open or conflicting PRs, pending checks, private access, insufficient permissions, rate-limit exhaustion, and more than 100 checks. The saved exports and simulated responses are described in the [fixture notes](../../testdata/pr-metadata/README.md).

## Run the checks

Baseline commit: `579589ab691fe3abf0cd06b083c930baaf977816`. Tested with Go 1.27.4 on Linux AMD64 and GitHub CLI 2.106.0.

From the repository root, run the [offline tests](../../internal/command/github_native_spike_test.go). They use local HTTP servers and need no GitHub credentials.

```sh
RENODASH_NATIVE_SPIKE_LIVE=0 go test ./internal/command \
  -run '^TestNativeSpike' -count=1
```

The [live test](../../internal/command/github_native_live_spike_test.go) passes the same resolved token to Go and sets it in the environment for the `gh` subprocess. It reports field names if values differ. To compare both clients using a token from a saved login, sign in with [`gh auth login`](https://cli.github.com/manual/gh_auth_login) and run:

```sh
env -u GH_TOKEN -u GITHUB_TOKEN RENODASH_NATIVE_SPIKE_LIVE=1 \
  go test ./internal/command -run '^TestNativeSpikeLive$' -count=1 -v
```

For each PR, compare the production `ghClient.FetchPR` result with the native HTTP result and `gh pr view --json`.
The test compares selected JSON fields and the metadata decoded by renodash.
If any comparison differs, the test reads all three paths once more.
If any path changes, or heads or update times differ, the result is inconclusive.
A stable difference at the same head and update time is reported as a mismatch.

Run the [library probes](../../testdata/github-library-probe/probe_test.go) from the repository root. The go-gh credential probe uses a fake executable and needs `/bin/sh`. The probes use fake tokens and in-memory HTTP responses. Run this task on demand for the spike.

```sh
MISE_CONFIG_DIR="$PWD/.config/mise" mise run github-library-probe
```

## Work in #116

- Connect the native HTTP adapter to `Run`. Keep the existing query, validation, retry rules, JSON, and exit codes.
- Use the credential order above. Test environment credentials without `gh` and a fake `gh` that accepts only the credential command.
- Check credential refresh for expiring tokens. Decide how to handle environments that need a proxy or a redirect.
- Run the offline and release checks. Repeat live comparisons for the cases above when test targets are available.
