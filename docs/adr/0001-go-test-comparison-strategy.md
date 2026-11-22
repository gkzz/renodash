# ADR 0001: Go test comparison strategy

Status: Proposed

Date: 2026-11-22

## Context

At the baseline commit `848ea5c`, renodash uses standard `testing`, table-driven tests, and `reflect.DeepEqual` for structured values.
Package fixtures and independent golden files cover parsing and rendering.
Offline integration tests exercise real processes with a fake `gh`, without GitHub credentials.
The baseline Go module has no external dependencies.

[PR #99](https://github.com/gkzz/renodash/pull/99) investigates clearer comparison diagnostics.
Its spike adds `go-cmp` and tries changes in parser, research, and offline integration tests.
These trial changes are evidence, not an approved adoption decision.

## Decision Drivers

- Show nested field differences when a test fails.
- Preserve comparison accuracy, including nil versus empty collections and slice order.
- Keep tests readable and maintainable within the existing Go conventions.
- Limit dependencies and the APIs reviewers need to understand.
- Preserve offline tests, fake commands, and execution through `go test` in CI.

## Considered Options

| Option | Assessment for renodash |
| --- | --- |
| Standard `testing` only | Fits test execution and simple checks. Structured failure diagnostics require custom formatting and maintenance. |
| `testing` + `go-cmp` | Adds field differences while preserving test execution and control flow. Requires one test dependency and care with comparison rules. |
| `testing` + Testify | Offers assertions, mocks, and waiting helpers. Current checks do not demonstrate a need for these additional APIs. |
| `testing` + Gomega | Works with standard tests and supports polling conditions. Current subprocess tests already use explicit synchronization. |
| Ginkgo + Gomega | Supports hierarchical scenarios. Current table-driven tests do not justify a new suite structure and its migration cost. |

`testing/synctest` serves a different role: testing goroutines and virtual time within a process.
Consider it when those tests grow; it does not replace the current external-process and loopback synchronization tests.

## Decision

Use standard `testing` for execution and failure reporting.
Use `cmp.Diff(want, got)` for complete Dashboard and Research values, nested dependency evidence, and captured JSON metadata.
Report differences with the label `(-want +got)`.
Keep `t.Error` and `t.Errorf` for checks that continue, and `t.Fatal` and `t.Fatalf` for checks that stop.

Prefer ordinary equality for scalar values.
Use `slices.Equal` and `maps.Equal` for simple collections only when nil and empty have the same intended meaning.
Use `slices.EqualFunc` with `slices.Equal` for ordered command argument lists.
For the selected nonempty expectations, these functions preserve the existing pass/fail behavior.
Keep `reflect.DeepEqual` where its semantics are intentional and migration offers no useful diagnostic improvement.

Use default `go-cmp` options for structures: preserve nil versus empty and slice order.
Do not add `cmpopts.EquateEmpty`, sorting, ignored fields, or JSON conversion merely to make comparisons pass.
Do not add Testify, Gomega, or Ginkgo now.
Keep fake command dispatch as ordinary boolean checks and preserve real subprocess coverage.

Apply only the relevant existing-test changes from the spike.
Do not merge the spike or include `internal/testingcomparison/` in the implementation branch.
The implementation branch will inherit this record; Proposed does not assert maintainer approval.

## Consequences

Nested failures become easier to locate without custom per-field diagnostic code.
The CLI behavior, golden expectations, and test runner remain unchanged.
`github.com/google/go-cmp` becomes a module dependency used only by tests, with version, license, and update maintenance costs.
Go modules do not have a separate test dependency section.

Reviewers must check expected values, assertion severity, nil handling, order, and comparison options when changing tests.
`go-cmp` can invoke `Equal` methods and panics on unexported fields by default.
The compared Dashboard and Research types currently have exported fields and no `Equal` methods.
Reassess semantics if these types change; `cmp.Diff` is not a universal replacement for `reflect.DeepEqual`.

Reevaluate Testify if repeated assertion control flow becomes hard to maintain.
Compare waiting helpers if polling conditions become necessary, and Ginkgo if shared hierarchical setup becomes complex.
Before adoption, verify representative failures, comparison semantics, dependency versions and licenses, and CI compatibility.

The spike did not benchmark performance or maintenance effort, or execute the unselected libraries.
Its diagnostics do not establish behavior for all unexported fields or `Equal` methods.
Local Linux results do not establish macOS or Windows compatibility.
The existing `govulncheck ./...` job excludes test code by default; test dependency reachability requires `-test`.
Changing that CI scope is outside this decision's implementation.

## Evidence / References

The spike's evaluation reports equal mismatch detection for its samples, with clearer nested diagnostics from `go-cmp`.
The `%#v` alternative displays nested Research pointers as addresses.
The comparison tests cover parser inputs and deliberate nested-field, nil/empty, and order mismatches.
These are bounded diagnostic experiments, not a benchmark of all standard-library approaches.

- [Comparison document at spike commit e4073ae](https://github.com/gkzz/renodash/blob/e4073aeafb11432bad38fe86d605ee04d0aac50d/docs/test-libraries.ja.md)
- [Executable comparison and diagnostic checks](https://github.com/gkzz/renodash/blob/e4073aeafb11432bad38fe86d605ee04d0aac50d/internal/testingcomparison/comparison_test.go)
- [Standard testing](https://pkg.go.dev/testing) and [testing/synctest](https://pkg.go.dev/testing/synctest)
- [go-cmp comparison rules](https://pkg.go.dev/github.com/google/go-cmp/cmp)
- [slices.Equal](https://pkg.go.dev/slices#Equal), [maps.Equal](https://pkg.go.dev/maps#Equal), and [reflect.DeepEqual](https://pkg.go.dev/reflect#DeepEqual)
- [Testify](https://github.com/stretchr/testify), [Gomega](https://onsi.github.io/gomega/), and [Ginkgo](https://onsi.github.io/ginkgo/)
- [Developer test conventions](../testing.md)
