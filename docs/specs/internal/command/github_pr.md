# GitHub PR transport

Fetch pull request (PR) metadata and all check pages through `gh api graphql --hostname github.com --include`.
The caller validates PR metadata. See [linked PR validation](pr.md) and [process execution](github_client.md).

## Input and validation

Accept a repository, PR number, and context from the caller.
Preserve repository casing and format numbers as decimal integers.
Pass fixed query arguments directly to `gh`, without a shell.

## Behavior

### Check pages and normalization

- For PRs, read HTTP headers and extract PR data from the GraphQL response.
- Fetch every page of check results.
- Require exactly one commit on every page; otherwise return `invalid_metadata`.
- Allow a null check rollup on the initial page. Require a non-null rollup on continuation pages; otherwise return `invalid_pagination`.
- Normalize checks to the existing `gh pr view --json` format: flatten `workflowName`, rename StatusContext `createdAt` to `startedAt`, and omit `description` and `checkSuite`.
- Match GitHub CLI timestamp encoding, including `0001-01-01T00:00:00Z` for null check timestamps.
- Reject GraphQL errors, missing or repeated page cursors, and head SHA changes between pages.
- Require the queried commit to match the PR head SHA. This does not prove required-check coverage or merge readiness.

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
| Missing or null commits, missing or multiple commit nodes, or an empty commit OID | `invalid_metadata` |
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

Requests inherit the [client deadlines and cancellation rules](github_client.md#deadlines-and-cancellation).

## Implementation and test evidence

Implementation: [github_pr.go](../../../../internal/command/github_pr.go).
Tests: [github_pr_test.go](../../../../internal/command/github_pr_test.go).

- `TestGHPRPaginationAndArguments` checks query arguments, page continuation, and response unwrapping.
- `TestGHPRRejectsIncompleteEvidence`, `TestGHPRCommitErrorClassification`, and `TestGHPRRejectsMissingCommitOID` check response validation.
- `TestGHPRCheckCompatibility` and `TestGHPRPendingCheckTimes` check field and timestamp normalization.
- `TestGHPRInitialNullRollup` checks a head commit without checks.
- `TestStructuredHTTPFailures`, `TestGHPRClassifiesFailedProcessHTTP`, and `TestGHPRSecondaryRateLimit` check error classification and wait headers.
- `TestGHPRForbiddenPartialCollection` checks that transport classifications control later PR requests.

Tests are offline. They do not establish live authentication or GitHub availability.
The subprocess tests use a Go test executable through [fakegh_test.go](../../../../internal/command/fakegh_test.go), without shell scripts.
See [command subprocess tests](../../../testing.md#command-subprocess-tests) for helper naming and platform validation limits.
These tests do not establish live GitHub response completeness or support on other operating systems.
