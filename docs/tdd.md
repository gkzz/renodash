# Specification and TDD workflow

Use test-driven development (TDD) to connect an expected behavior to a failing test, a small implementation change, and refactoring.
This guide builds on the validation work in PRs #55, #57, and #59, the fetching specification in #58, and dependency injection in #60.
It describes the recommended workflow, not the order in which those PRs were developed.

## When to update specifications

- Update the relevant specification when changing accepted input, output, errors, timeouts, or other observable behavior.
- Add a specification when a new behavior has no suitable document. Mirror the responsible implementation directory under `docs/specs/`.
- For a bug fix, check whether the existing specification already describes the expected behavior. Correct missing or ambiguous rules before implementing the fix.
- For a refactoring that preserves behavior, keep the contract unchanged. Update implementation links, test names, and coverage notes if they change.
- For wording, formatting, or link fixes, run the relevant documentation checks. A new Go test is unnecessary.

Keep behavior rules in specifications and development steps in guides.
Link to existing rules rather than copying them into several documents.
Record missing coverage in the [coverage review](specification-coverage.md) or a linked follow-up issue.

## Red → Green → Refactor

Run commands from the repository root with the tools described in [Development](development.md).

1. Choose one observable behavior. Find its specification and state the expected input, result, and failure conditions.
2. **Red:** add a focused test with independent expected values. Run it and confirm that it fails because the behavior is missing.
   A compile error, missing fixture, or unavailable tool does not establish the intended failure.
3. **Green:** make the smallest implementation change that satisfies the behavior. Run the focused test, then the affected package tests.
4. **Refactor:** simplify names, duplication, or dependency passing while keeping the behavior unchanged. Rerun the affected tests after each change.
5. Check the specification, implementation links, tests, and coverage notes together. Run `make check` before review.

Use table-driven tests for related cases, explicit expected values as in #57, and local injected clients as in #60.
Avoid network access and real sleeps for context tests; cancel the context from a fake client instead.
For a small change, one rule and one focused regression test can be enough. No separate design document or test framework is required.
In the PR, report the commands and results, including any checks that were not run.

## Reading example: cancellation handling and regression tests

The GitHub client added in PR #60 provides an example of how a specification, implementation, and regression test fit together.
Read the following parts together to understand the behavior they protect.

### Start with the contract

The [GitHub client specification](specs/internal/command/github-client.md#deadlines-and-cancellation) requires requests to reject data after cancellation or deadline expiry.
Even when a client returns bytes without an error, the request must return a context error and no data.

### Follow the implementation

In [pr.go](../internal/command/pr.go), `fetchPR` creates a request context with a 30-second timeout and passes it to the injected client.
After checking the client error, it checks the request context before returning data:

```go
if err := ctx.Err(); err != nil {
	return nil, err
}
```

This check rejects late data when the client returns no error.
The request context also respects an earlier parent deadline and parent cancellation.

### Read the regression test

`TestFetchRejectsLateSuccess` in [github_client_test.go](../internal/command/github_client_test.go) covers Issue and PR requests with cancellation and expired deadlines.
For cancellation cases, the fake client cancels the parent during the request and then returns successful bytes.
For deadline cases, the parent deadline has already expired before the request starts.

The test checks that the result contains no data, the error matches the expected context error, and the client received a canceled or expired context.
Canceling from the fake callback makes the cancellation case deterministic without real sleeps or network access.

Run the existing tests from the repository root:

```sh
go test ./internal/command -run '^TestFetchRejectsLateSuccess$' -count=1 -v
go test ./internal/command
```

Expect four passing subtests and a passing package test run.
These tests show the current behavior; they do not establish whether PR #60 was developed test-first.
For a new behavior or bug fix, use the Red → Green → Refactor workflow above with a test for the requested change.
