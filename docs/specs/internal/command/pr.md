# Linked pull requests

This specification defines linked pull request (PR) resolution and response validation for `get --with-pr-metadata`.
See [JSON and parsing](../dashboard/output-format.md#linked-pull-request-metadata) for output fields and check interpretation.
See [Dashboard fetching](get.md) for issue fetching and output failures.

## Link resolution

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

## Fetch order and request boundaries

Visit items in input order, skipping empty PR links.
Resolve each link before checking for a duplicate integer number.
Fetch each distinct number once, preserving first-seen order in `pull_requests`.
Different accepted forms and leading zeroes for the same number share one request.
An invalid later link still fails even if an earlier link resolved to the same PR.
With no linked PRs, return an empty array.

Fetch by integer number and the Dashboard repository through the injected GitHub client.
The production client uses `gh pr view NUMBER --repo OWNER/REPO` with the requested JSON fields.
See [GitHub CLI client](github-client.md) for transport, request budgets, and cancellation.
Each unique PR receives its own request budget while preserving parent cancellation.
Stop at the first link, fetch, decoding, or metadata error.
Fetched PRs are not emitted as partial JSON; see [output failures](get.md#output-and-failures).

## Response validation

Decode each response as JSON with the following fields. Ignore unrelated fields.
Invalid JSON or incompatible field types return `decode PR NUMBER: <decoder error>`.
Missing fields and JSON null values keep Go zero values in this response structure.

| Response field | Validation and output |
| --- | --- |
| `number` | Must equal the requested integer number |
| `url` | Must pass the link rules above for the Dashboard repository and resolve to the requested number; preserve the original string |
| `state` | Require exactly `OPEN`, `CLOSED`, or `MERGED` |
| `title` | No nonempty requirement; preserve as `title` |
| `isDraft` | Preserve as `is_draft`; missing or null becomes `false` |
| `headRefOid` | Require a nonempty string; preserve as `head_sha` |
| `updatedAt` | Require a nonempty string; preserve as `updated_at` |
| `statusCheckRollup` | Preserve array entries as raw JSON in `checks`; missing or null becomes `[]` |

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

## Implementation and test evidence

Implementation: `linkedPRNumber`, `fetchPullRequests`, and `fetchPR` in [pr.go](../../../../internal/command/pr.go).
Production transport: `ghClient.FetchPR` in [github_client.go](../../../../internal/command/github_client.go).

The following tests are in [pr_test.go](../../../../internal/command/pr_test.go):

| Tests | Representative evidence |
| --- | --- |
| `TestLinkedPRNumber` | Four link forms, repository casing, wrong host/repository, zero, negative numbers, queries, fragments, and extra segments |
| `TestGetWithPRMetadata` | Duplicate requests, selected output fields, mixed check types, and fetch failure without stdout |
| `TestPullRequestStatesAndInvalidMetadata` | Three accepted states, rejected unknown state, null checks, and empty PR collections |
| `TestMultiplePRsHaveIndependentTimeouts` | Distinct request budgets, deduplication, and first-seen output order |
| `TestPRFetchPreservesParentCancellation` | Parent cancellation rejects a successful client response |

`TestFetchRejectsLateSuccess` in [github_client_test.go](../../../../internal/command/github_client_test.go) covers cancellation and expired deadlines for issue and PR requests.
`TestOfflineCapturedMetadata` in [offline_test.go](../../../../internal/integration/offline_test.go) checks the built CLI against captured PR fixtures through a fake `gh`.
`TestOfflineLookupErrors` in that file covers failed fetches, malformed JSON, and missing metadata without stdout.
See [fixture provenance](../../../../testdata/README.md) for captured data sources.

These are representative tests, not exhaustive validation coverage.
Rules for number overflow, empty query markers, URL user information, and individual metadata mismatches also follow from implementation review.
The timestamp and hash checks described above are limited to nonempty strings.
Offline evidence does not establish live GitHub behavior, upstream field completeness, or support on other operating systems.
