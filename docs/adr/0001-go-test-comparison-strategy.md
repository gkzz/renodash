# ADR 0001: Go test comparison strategy

Status: Proposed

Date: 2026-11-22

## Context

### Existing tests

At baseline commit `848ea5c`, tests use:

- Standard `testing` and table-driven tests.
- `reflect.DeepEqual` to compare structured values.
- Package fixtures and separate golden files (expected output) for parsing and rendering.
- Offline integration tests with real processes and a fake `gh`, without GitHub credentials.
- No external Go module dependencies.

### Problem to solve

Whole-value comparisons detect mismatches, but their failure messages can be hard to read.
Printing a Dashboard with `%#v` shows nested Research pointers as addresses.
The goal is to show those values while keeping the existing comparison rules.
This does not require more tests of internal state.

### Comparison experiment

[PR #99](https://github.com/gkzz/renodash/pull/99) tried `go-cmp` for clearer differences.
A second experiment showed that standard formatting can also display nested Research values.
Keep the experiment separate from the implementation.

## Decision Drivers

### Comparison accuracy

- Compare complete expected results.
- Preserve nil versus empty collections and slice order.
- Keep each check's choice to continue or stop after failure.

### Readability and maintenance

- Use familiar Go conditionals and standard packages.
- Improve messages where values are hard to inspect.
- Avoid adding dependency review, updates, and new APIs before they are needed.
- Keep offline tests and execution through `go test` in continuous integration (CI).

## Considered Options

| Option | Benefit | Cost or limit |
| --- | --- | --- |
| Standard packages with small message improvements | Keeps comparison rules and adds no dependencies | Large values still need manual inspection |
| `testing` + `go-cmp` | Shows nested field differences automatically | Adds a dependency and different comparison rules |
| `testing` + Testify or Gomega | Adds assertions and waiting helpers | Current tests show no need for these APIs |
| Ginkgo + Gomega | Supports nested test scenarios | Current table-driven tests do not need a new suite structure |

`testing/synctest` serves a separate purpose: testing goroutines and virtual time within one process.
It does not replace the current external-process and loopback synchronization tests.

## Decision

Use standard Go packages. Improve failure messages where needed.

### Comparison methods

| Value or check | Use |
| --- | --- |
| Complete Dashboard and Research values | `reflect.DeepEqual` |
| Nested dependency evidence and captured JSON metadata | `reflect.DeepEqual` |
| Scalar values | Ordinary equality (`==` or `!=`) |
| Simple slices or maps where nil and empty mean the same thing | `slices.Equal` or `maps.Equal` |
| Ordered command argument lists | `slices.EqualFunc` with `slices.Equal` |

### Failure messages and test flow

- Print expected and actual values with `%#v`.
- For Dashboard mismatches, also print each changed Research value with its Item index.
- Keep the whole-value comparison so fields omitted from extra messages still affect the result.
- Keep `t.Error` and `t.Errorf` for checks that continue.
- Keep `t.Fatal` and `t.Fatalf` for checks that stop.

### Implementation scope

- Keep Go tests free of external module dependencies.
- Keep fake command dispatch as boolean checks and preserve real subprocess tests.
- Keep CLI behavior and independent golden expectations unchanged.
- Do not import the experiment's comparison package or internal-record experiments.
- Do not build a general diff engine or add another test library now.
- Handle named fields in fixture literals separately; see [Issue #101](https://github.com/gkzz/renodash/issues/101).

## Consequences

### Benefits and limits

| Area | Effect |
| --- | --- |
| Dependencies | No external Go module dependencies, including tests |
| Failure messages | Nested Research values are visible instead of only pointer addresses |
| Comparison accuracy | Whole-value checks still distinguish nil, empty collections, and slice order |
| Readability | Uses standard Go APIs; large values still produce long messages |
| Maintenance | New nested pointer types may need a small message improvement |

### Internal fields

Dashboard and Research currently contain exported fields.
Internal types such as `dependencyRecord` have unexported fields.
Their presence is normal and does not require comparing all internal state.
Prefer checking returned results and observable behavior.

`reflect.DeepEqual` compares unexported fields; `go-cmp` can panic on them with default settings.
Ignoring those fields can hide meaningful differences.
Keep field visibility based on application needs, rather than comparison tools.

### Future library decisions

Consider a library only when a concrete problem remains after reasonable improvements with standard packages.
Record that problem, an example failure, alternatives, and dependency costs in a separate ADR.
Star counts and release intervals alone do not establish safety or maintenance quality.
Test dependencies also execute on developer machines and in CI, so their review and updates have a cost.

## Evidence / References

The second experiment checked nested values, nil versus empty collections, slice order, and actual parser results.
It also compared unexported fields using a real internal type.
These are selected Linux checks, not performance measurements or proof of compatibility on other operating systems.

The second experiment is recorded on `spike/testing-library-comparison` at commit `540d545`:

- [Comparison report](https://github.com/gkzz/renodash/blob/540d545/docs/test-libraries.ja.md)
- [Standard-only checks](https://github.com/gkzz/renodash/blob/540d545/internal/testingcomparison/standard_test.go)
- [Unexported-field checks](https://github.com/gkzz/renodash/blob/540d545/internal/dashboard/comparison_spike_test.go)

### API references

- [Standard testing](https://pkg.go.dev/testing) and [formatting](https://pkg.go.dev/fmt)
- [reflect.DeepEqual](https://pkg.go.dev/reflect#DeepEqual), [slices.Equal](https://pkg.go.dev/slices#Equal), and [maps.Equal](https://pkg.go.dev/maps#Equal)
- [go-cmp comparison rules](https://pkg.go.dev/github.com/google/go-cmp/cmp)
- [Developer test conventions](../testing.md)
