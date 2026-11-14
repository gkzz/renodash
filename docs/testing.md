# Testing strategy

Follow Go conventions and keep tests close to the behavior they verify.

Use the [specification and TDD workflow](tdd.md) when changing behavior.
Track missing specifications, test gaps, and follow-up work in [GitHub Issues](https://github.com/gkzz/renodash/issues).

- Use the standard `testing` package and `*_test.go` files beside package code.
- Put package fixtures and golden files in `testdata/`. Use table-driven tests for related cases and `t.TempDir()` for temporary files.
- Test parsing and rendering within their packages. Use `internal/integration` to check CLI input, output, errors, and exit codes through real processes.
- Keep integration tests offline with saved fixtures and a fake `gh` command. Tests must not require GitHub credentials or network access.
- Compare rendered output with independent golden files. Do not regenerate summary expectations from the Go renderer; see the [fixture notes](../internal/summary/testdata/README.md).
- Use platform-native names for test binaries invoked by an explicit path: `fake-actions-summary.exe` on Windows and `fake-actions-summary` on Linux and macOS.
- For fake commands discovered through `PATH`, preserve the command name and platform-specific lookup requirements; see below.
- Keep small helpers local to their tests. Do not add a shared utility package just for executable names or a single call site.
- Add tests for observable behavior, boundary cases, and failures. Avoid tests that only repeat implementation details.
- Use independent expected values; do not derive them from the implementation under test.

See [Development](development.md#setup-and-tests) for prerequisites and test commands.
A local test run verifies the host OS; it does not establish Windows compatibility.

## Command subprocess tests

The command transport tests use a copy of their Go test executable as a fake `gh`.
[fakegh_test.go](../internal/command/fakegh_test.go) dispatches helper modes before the normal test runner starts.
The helper can print received arguments, return supplied stdout, or fail with supplied stderr.
It does not invoke a shell, call the real GitHub CLI, or contact external services.
Each test sets its own helper mode and payload; test cleanup restores the environment.

The running-process cancellation test uses a helper mode that connects to a loopback TCP listener and sends a startup signal.
The helper writes partial stdout and an authentication diagnostic before signaling, then waits for the connection to close.
The test cancels only after receiving the signal and checks the context error, discarded output, and connection closure.
Five-second waits bound startup and cancellation checks; closing the connection releases the helper on failure.
The helper also has a 30-second connection deadline as a fallback.
This local synchronization needs loopback socket access but no GitHub credentials or external network access.

The production client finds `gh` through `PATH`, so the helper must use the platform's lookup name.
Use `gh` on Unix and `gh.exe` on Windows, where the test sets `PATHEXT` to `.EXE`.
The helper directory replaces `PATH` so these tests do not fall back to an installed `gh`.
The offline integration tests use their own fixture-serving Go executable with the same lookup names.

Run the command package tests from the repository root:

```sh
go test ./internal/command -count=1
```

The test workflow runs all Go tests on Ubuntu.
A separate macOS job runs the command and integration packages with one Go version.
It checks executable behavior without repeating documentation tooling or adding architecture and Go-version matrices.
Windows execution remains unverified and is not enabled in the workflow.
Passing these tests does not establish exhaustive platform support.
Cross-compilation checks buildability, not process execution, executable lookup, or cancellation on the target OS.
