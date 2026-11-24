# Resilient Dashboard retrieval

`get` accepts one Dashboard target and can keep successful PR metadata when another PR fails.

## Input and validation

`--stdin` reads one target, limited to 4096 bytes including surrounding whitespace.
Trim surrounding whitespace, including a final LF or CRLF, then apply the existing issue-reference validator.
Reject empty input, more than one target, read errors, oversized input, or a positional target combined with `--stdin` before any network request.
Positional input remains supported.

`--allow-partial` requires `--with-pr-metadata`.
Without it, failures stop collection without emitting result JSON.

## Behavior

Each command has a five-minute budget covering the Dashboard and all PR requests.
Interrupt and termination signals cancel the command, including active subprocesses and retry waits.
Each PR request has a 30-second attempt deadline and at most three attempts.
Retry only timeouts, reliably classified HTTP 5xx failures, and rate limits whose known wait fits the remaining budget.
Use exponential backoff with jitter: 0.5–1 seconds before attempt two and 1–2 seconds before attempt three.
Do not retry authentication, permissions, invalid links, decoding, validation, or unknown errors.
Parent cancellation and the command deadline always abort collection.

## Output and failures

With `--allow-partial`, keep existing Dashboard fields and successful PRs.
Add `retrieval.status` (`complete` or `partial`) and an always-present `retrieval.errors` array.
Each error includes `resource`, `repository`, optional `number`, optional `link`, `code`, `attempts`, and `retryable`.
Never expose raw transport diagnostics in these errors.
Retryability describes the error category; it does not promise another attempt.

Valid PR numbers are deduplicated, including failures, in first-seen order.
Invalid links produce `invalid_link` errors with their original link and zero attempts, deduplicated by exact link.
Invalid JSON produces `decode_error`; mismatched metadata produces `invalid_metadata`.
Timeouts produce `timeout`; unclassified transport failures produce `fetch_failed`.
No failed PR receives fabricated metadata or a nonexistent state.

Complete retrieval, no linked PRs, and partial retrieval exit zero.
All PRs failing still returns partial JSON with an empty PR array.
Dashboard errors, invalid command input, cancellation, and command deadline expiry return nonzero without result JSON.
Output failures return nonzero and may leave an incomplete stream.

## Implementation and test evidence

See [get.go](../../../../internal/command/get.go), [pr.go](../../../../internal/command/pr.go), and [resilient_test.go](../../../../internal/command/resilient_test.go).
Offline tests do not establish live GitHub availability or authentication behavior.
