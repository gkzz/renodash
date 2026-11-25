# Retries and partial results

`get` reads one Dashboard target. It can keep successful PR metadata when another PR fails.

## Input and validation

See [get input validation](get.md#input-and-validation) for stdin and option rules.

## Behavior

### Time limits and retries

| Limit | Value |
| --- | --- |
| Dashboard and all PR requests together | 5 minutes |
| Each PR attempt | 30 seconds |
| Attempts per PR | At most 3, including the first request |
| Wait before attempt 2 | 0.5–1 seconds |
| Wait before attempt 3 | 1–2 seconds |

The five-minute budget starts after stdin reading and target validation. Stdin reading waits for EOF and has no time limit.

Waits use exponential backoff with jitter: the delay grows and includes a random range.

| Failure | Action |
| --- | --- |
| Request timeout or identified HTTP 5xx | Retry within the limits above |
| Rate limit with a known wait | Wait at least that long; retry only if the wait fits the command's remaining time |
| Authentication, permission, invalid link, decoding, validation, or unknown error | Do not retry |
| Parent cancellation, SIGINT/SIGTERM, or command deadline | Stop requests and retry waits; fail the command |

If a PR still reports a rate limit after retries, stop fetching further PRs.
With `--allow-partial`, keep successes and report remaining unique valid PRs with `code: "rate_limit"` and zero attempts.
This also applies when the wait is unknown or exceeds the remaining budget.

### Repeated links

- Fetch each valid PR number once, apart from retries. Keep first-seen order, including failed PRs.
- Report each distinct invalid link once, using its original string and zero attempts.
- Never invent metadata or a PR state for a failed request.

## Output and failures

With `--allow-partial`, keep existing Dashboard fields and successful PRs. Add:

| Field | Meaning |
| --- | --- |
| `retrieval.status` | `complete` or `partial` |
| `retrieval.errors` | An array of failed resources; `[]` when none failed |

Each error has these fields:

| Field | Meaning |
| --- | --- |
| `resource`, `repository` | Resource kind and repository |
| `number` | PR number, when known |
| `link` | Original link for an invalid link |
| `code` | Error category, such as `timeout`, `decode_error`, `invalid_metadata`, `invalid_link`, or `fetch_failed` |
| `attempts` | Number of attempts made |
| `retryable` | Whether the error category allows retries; another attempt is not guaranteed |

Errors do not include raw transport diagnostics.
See [GitHub CLI errors](github_pr.md#structured-pr-errors) for HTTP and GraphQL categories.

| Result with `--allow-partial` | JSON | Exit code |
| --- | --- | --- |
| All PRs fetched, or no linked PRs | `complete`, empty errors | `0` |
| Some or all PRs failed | `partial`, successful PRs and errors | `0` |
| Dashboard failure, invalid input, cancellation, or command deadline | No result JSON | `1` |
| Output failure | May leave incomplete JSON | `1` |

Absence from `pull_requests` does not prove that a PR does not exist; consult `retrieval.errors`.
If all PRs fail, `pull_requests` is `[]`.
Without `--allow-partial`, a final PR error stops collection without result JSON.

## Implementation and test evidence

- Input and output: [get.go](../../../../internal/command/get.go).
- Collection and retries: [retrieval.go](../../../../internal/command/retrieval.go).
- Tests: [get_test.go](../../../../internal/command/get_test.go) and [retrieval_test.go](../../../../internal/command/retrieval_test.go).

Offline tests do not verify live GitHub availability or authentication.
