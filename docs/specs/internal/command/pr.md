# Linked pull requests

This specification defines linked pull request (PR) resolution and response validation for `get --with-pr-metadata`.
See [PR output fields](#linked-pull-request-metadata) for output fields and check interpretation.
See [Dashboard fetching](get.md) for issue fetching and output failures.

## Input and validation

### Link resolution

Resolve nonempty item `pr_link` values against the Dashboard repository selected by the accepted issue URL.
Without `--with-pr-metadata`, `get` preserves links without resolving them or fetching PRs.
Offline `parse` also preserves links without resolution.

For repository `gkzz/example`, these forms identify PR 33:

| Link form | Example |
| --- | --- |
| Parent-relative | `../pull/33` |
| Repository-relative | `pull/33` |
| Root-relative | `/gkzz/example/pull/33` |
| Absolute | `https://github.com/gkzz/example/pull/33` |

Expand the two relative prefixes using the Dashboard repository.
Prefix a single leading slash with `https://github.com`; do not expand protocol-relative links beginning with `//`.
Then apply these rules to the parsed URL:

| Component | Rule |
| --- | --- |
| Scheme | Require `https` |
| Host | Require `github.com`, compared without case sensitivity; explicit ports fail |
| User information | Reject when present |
| Query | Reject nonempty queries and an empty query marker (`?`) |
| Fragment | Reject nonempty fragments; an empty trailing `#` is accepted |
| Escaped path | Require `/OWNER/REPO/pull/NUMBER`; compare the repository without case sensitivity |
| Path separator | Require the literal lowercase `/pull/` segment |
| Number | Require one or more ASCII decimal digits and a positive value representable by Go `int` |

Do not decode escaped path characters to make the repository or number match.
Trailing slashes, extra path segments, signs, whitespace, and escaped digits fail.
Leading zeroes are accepted: `../pull/0033` resolves to integer 33.
The upper bound depends on the executable's `int` width; this validator does not impose a separate GitHub PR-number limit.
Repository syntax is governed by the [input validation specification](validation.md), not revalidated here.

For example, `https://github.com/GKZZ/example/pull/33` matches `gkzz/example`.
A different repository, another host, `http`, or `../pull/33/files` fails.
No Dashboard-supplied host or complete URL is passed to the GitHub client.

An invalid URL or repository/path match returns `unsupported PR link <quoted link>`.
An empty number uses that same error.
A non-digit, zero, or overflowing number returns `invalid PR number in <quoted link>`.
Both errors quote the original link using Go `%q`.

### Response validation

Decode each response as JSON with the following fields. Ignore unrelated fields.
Invalid JSON or incompatible field types return `decode PR NUMBER: <decoder error>`.
Missing fields and JSON null values decode to Go zero values in this response structure.
Validation then rejects values that do not meet the requirements below.

| Response field | Validation and output |
| --- | --- |
| `number` | Must equal the requested integer number |
| `url` | Must pass the link rules above for the Dashboard repository and resolve to the requested number; preserve the original string |
| `state` | Require exactly `OPEN`, `CLOSED`, or `MERGED` |
| `title` | No nonempty requirement; preserve as `title` |
| `isDraft` | Preserve as `is_draft`; missing or null becomes `false` |
| `headRefOid` | Require a nonempty string; preserve as `head_sha` |
| `updatedAt` | Require a nonempty string; preserve as `updated_at` |
| `mergeable` | Preserve as `mergeable`; missing, null, or empty becomes `UNKNOWN` |
| `mergeStateStatus` | Preserve as `merge_state_status`; missing, null, or empty becomes `UNKNOWN` |
| `statusCheckRollup` | Preserve array entries as raw JSON in `checks`; missing or null becomes `[]` |

Preserve other nonempty merge values so new GitHub values remain visible. These fields do not prove that a PR is ready to merge.
Record `fetched_at` after each successful PR request, using UTC RFC 3339.
See [output fields](#linked-pull-request-metadata) for timestamp limits and merge state meanings.

The response URL uses the same resolver as Dashboard links.
It can be a supported relative link and can contain leading zeroes; an absolute canonical URL is not required.

The validator does not parse `updatedAt` as a timestamp or check the format of `headRefOid`.
Whitespace-only strings pass their nonempty checks.
This differs from the issue response, whose `updated_at` must parse as RFC 3339.
Do not infer timestamp or commit-hash validity from successful PR validation.
No rule requires a nonempty title, checks, or individual check fields.
Check entries are not validated against a schema or used to decide merge eligibility.

A failed metadata condition returns `PR NUMBER JSON has missing or mismatched metadata`.
A client or context error is wrapped as `fetch PR OWNER/REPO#NUMBER: <error>`.

## Behavior

### Fetch order and request boundaries

Visit items in input order, skipping empty PR links.
Resolve each link before checking for a duplicate integer number.
Collect each distinct number once, preserving first-seen order in `pull_requests`.
A temporary fetch failure may trigger another attempt for that number.
Different accepted forms and leading zeroes for the same number share one request.
An invalid later link still fails even if an earlier link resolved to the same PR.
With no linked PRs, return an empty array.

Fetch by integer number and the Dashboard repository through the injected GitHub client.
The production client uses `gh api graphql --include` with a fixed query and typed target variables.
See [GitHub CLI client](github_client.md) for transport, request budgets, and cancellation.
Each unique PR receives its own request budget while preserving parent cancellation.
Retry transient fetch errors within the [retrieval limits](retrieval.md).
By default, stop at the first remaining link, fetch, decoding, or metadata error without result JSON.
With `--allow-partial`, continue independent PRs and record structured errors; see [Retries and partial results](retrieval.md).

### Linked pull request metadata

Run `renodash get --with-pr-metadata OWNER/REPO#ISSUE_NUMBER` to include a top-level `pull_requests` array.
Without this option, the output stays unchanged. Offline `parse` does not fetch PR metadata.

Each entry contains `number`, `url`, `state`, `title`, `is_draft`, `head_sha`, `updated_at`, `checks`, `mergeable`, `merge_state_status`, and `fetched_at`.
States are `OPEN`, `CLOSED`, or `MERGED`, as returned by GitHub.
The command collects each distinct linked PR in first-seen order, with bounded retries for transient failures.
It only follows links to PRs in the Dashboard's repository.
It does not search for other Renovate PRs or verify the linked PR's author.

`mergeable` and `merge_state_status` preserve GitHub's `mergeable` and `mergeStateStatus` values.
Missing, null, or empty merge fields become `UNKNOWN`. Other values are kept, including new GitHub values.

| Values | Meaning |
| --- | --- |
| `mergeable: CONFLICTING` | The PR has merge conflicts |
| `mergeable: MERGEABLE` with `merge_state_status: BLOCKED` | Another condition blocks merging; the cause needs checking |
| `UNKNOWN` or an unfamiliar value | More information is needed |

These values alone do not prove that a PR is ready to merge.
`fetched_at` records when the CLI received each PR response, in UTC RFC 3339.
It differs from the PR update time (`updated_at`) and Dashboard fetch time (`source.fetched_at`).
PR and check data may change during fetching. The timestamp does not prove that every check applies to `head_sha`.

The `checks` array preserves the selected GraphQL check objects retrieved through GitHub CLI.
Check runs include fields such as `name`, `status`, and `conclusion`.
Commit status contexts include `context` and `state`.
See the [PR transport specification](github_pr.md) for query fields and pagination.
An empty array means no checks were returned; it does not imply CI success or permission to merge.
The result reflects the data returned by GitHub CLI, including any upstream limits.

See [Dashboard fetching](get.md#output-and-failures) for failure behavior and
[request budgets](github_client.md#deadlines-and-cancellation) for timeouts.
The command does not decide merge eligibility or update priority.

## Implementation and test evidence

Implementation: `linkedPRNumber`, `decodePullRequest`, and `fetchPR` in [pr.go](../../../../internal/command/pr.go).
Production transport: `ghClient.FetchPR` in [github_pr.go](../../../../internal/command/github_pr.go).

The following tests are in [pr_test.go](../../../../internal/command/pr_test.go):

| Tests | Representative evidence |
| --- | --- |
| `TestLinkedPRNumber` | Four link forms, repository casing, wrong host/repository, zero, negative numbers, queries, fragments, and extra segments |
| `TestLinkedPRNumberRejectedBoundaries` | Rejected number overflow, empty query markers, and URL user information with exact diagnostics |
| `TestPullRequestMetadataMismatch` | Rejected response number, URL number, and URL repository mismatches without partial results |
| `TestPRMergeObservations` | Merge values, missing values, new GitHub values, and UTC fetch times |
| `TestGetWithPRMetadata` | Duplicate requests, selected output fields, mixed check types, and fetch failure without stdout |
| `TestPullRequestStatesAndInvalidMetadata` | Three accepted states, rejected unknown state, null checks, and empty PR collections |
| `TestMultiplePRsHaveIndependentTimeouts` | Distinct request budgets, deduplication, and first-seen output order |
| `TestPRFetchPreservesParentCancellation` | Parent cancellation rejects a successful client response |

`TestFetchRejectsLateSuccess` in [github_client_test.go](../../../../internal/command/github_client_test.go) covers cancellation and expired deadlines for issue and PR requests.
`TestOfflineCapturedMetadata` in [offline_test.go](../../../../internal/integration/offline_test.go) checks the built CLI against captured PR fixtures through a fake `gh`.
`TestOfflineMergeObservations` checks made-up merge values through the built CLI.
`TestOfflineLookupErrors` in that file covers failed fetches, malformed JSON, and missing metadata without stdout.
See [fixture provenance](../../../../testdata/README.md) for captured data sources.

These are representative tests, not exhaustive validation coverage.
The timestamp and hash checks described above are limited to nonempty strings.
Offline evidence does not establish live GitHub behavior, upstream field completeness, or support on other operating systems.
