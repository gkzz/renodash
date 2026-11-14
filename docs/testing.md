# Testing strategy

Follow Go conventions and keep tests close to the behavior they verify.

Use the [specification and TDD workflow](tdd.md) when changing behavior.
Track missing specifications, test gaps, and follow-up work in [GitHub Issues](https://github.com/gkzz/renodash/issues).

- Use the standard `testing` package and `*_test.go` files beside package code.
- Put package fixtures and golden files in `testdata/`. Use table-driven tests for related cases and `t.TempDir()` for temporary files.
- Test parsing and rendering within their packages. Use `internal/integration` to check CLI input, output, errors, and exit codes through real processes.
- Keep integration tests offline with saved fixtures and a fake `gh` command. Tests must not require GitHub credentials or network access.
- Compare rendered output with independent golden files. Do not regenerate summary expectations from the Go renderer; see the [fixture notes](../internal/summary/testdata/README.md).
- Build test binaries with a fixed `.exe` suffix, such as `fake-actions-summary.exe`. This also works on Linux and macOS and avoids OS branches for naming.
- Keep small helpers local to their tests. Do not add a shared utility package just for executable names or a single call site.
- Add tests for observable behavior, boundary cases, and failures. Avoid tests that only repeat implementation details.
- Use independent expected values; do not derive them from the implementation under test.

See [Development](development.md#setup-and-tests) for prerequisites and test commands.
A local test run verifies the host OS; it does not establish Windows compatibility.
