# Testing strategy

Follow Go conventions and keep tests close to the behavior they verify.

Use the [specification and TDD workflow](tdd.md) when changing behavior.
Track missing specifications, test gaps, and follow-up work in [GitHub Issues](https://github.com/gkzz/renodash/issues).

- Use the standard `testing` package and `*_test.go` files beside package code.
- Put package fixtures and golden files in `testdata/`. Use table-driven tests for related cases and `t.TempDir()` for temporary files.
- Test parsing and rendering within their packages. Use `internal/integration` to check CLI input, output, errors, and exit codes through real processes.
- Keep integration tests offline with saved fixtures and a fake `gh` command. Tests must not require GitHub credentials or network access.
- Compare rendered output with independent golden files. Do not regenerate summary expectations from the Go renderer; see the [fixture notes](../internal/summary/testdata/README.md).
- Build directly invoked test binaries with a fixed `.exe` suffix, such as `fake-actions-summary.exe`. Executables found through `PATH` need the platform's lookup name; see below.
- Keep small helpers local to their tests. Do not add a shared utility package just for executable names or a single call site.
- Add tests for observable behavior, boundary cases, and failures. Avoid tests that only repeat implementation details.
- Use independent expected values; do not derive them from the implementation under test.

See [Development](development.md#setup-and-tests) for prerequisites and test commands.
A local test run verifies the host OS; it does not establish Windows compatibility.

## Command subprocess tests

The command transport tests use a copy of their Go test executable as a fake `gh`.
[fakegh_test.go](../internal/command/fakegh_test.go) dispatches helper modes before the normal test runner starts.
The helper can print received arguments, return supplied stdout, or fail with supplied stderr.
It does not invoke a shell, call the real GitHub CLI, or make network requests.
Each test sets its own helper mode and payload; test cleanup restores the environment.

The production client finds `gh` through `PATH`, so the helper must use the platform's lookup name.
Use `gh` on Unix and `gh.exe` on Windows, where the test sets `PATHEXT` to `.EXE`.
The helper directory replaces `PATH` so these tests do not fall back to an installed `gh`.
The offline integration tests use their own fixture-serving Go executable with the same lookup names.

Run the command package tests from the repository root:

```sh
go test ./internal/command -count=1
```

The test workflow currently runs on Ubuntu.
Windows and macOS execution remain unverified; removing shell scripts alone does not establish support on those systems.
Cross-compilation checks buildability, not process execution, executable lookup, or cancellation on the target OS.
