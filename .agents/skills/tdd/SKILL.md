---
name: tdd
description: Apply specification-driven TDD in renodash when implementing features or bug fixes, documenting the TDD workflow and examples, or reviewing specification and test coverage.
---

# Specification-driven TDD

Connect specifications, implementation, and tests within the requested scope.
Use the repository guides as the source of procedures rather than duplicating them here.

## Repository references

The repository root is `../../../` relative to this file. Run commands from that root.

- [TDD workflow and example](../../../docs/tdd.md): read when starting work.
- [Specification coverage review](../../../docs/specification-coverage.md): locate relevant specifications and known gaps.
- [Testing strategy](../../../docs/testing.md): read when adding or changing tests.
- [Development](../../../docs/development.md): check prerequisites and validation commands.
- [Specification index](../../../docs/README.md#specifications): find the corresponding contract under `docs/specs/`.
- [Writing documentation](../../../docs/writing.md): read when changing guides or specifications.

## Features and bug fixes

1. Read the relevant specification, implementation, and existing tests. Choose one observable behavior.
   For a bug that violates an existing contract, use that contract. Clarify missing or ambiguous rules before implementing changes.
2. **Red:** add a focused test with independent expected values and run it.
   Confirm failure because the intended behavior is missing. Compilation errors and environment failures do not establish Red.
3. **Green:** add the required behavior, then run the focused test and affected package tests.
4. **Refactor:** simplify the implementation or tests where useful while preserving behavior. Rerun affected tests.
   Do not force structural changes when no cleanup is needed.
5. Update the relevant specification, implementation/test links, and coverage notes. Run the repository's required checks.

Do not generate expected values from the implementation under test. Use table-driven tests for related cases.
Use local fake clients or saved fixtures for transport tests and cancel contexts from fake callbacks.
Avoid introducing real-time waits or mutable global hooks as the default testing approach.
For behavior-preserving refactoring, start with existing tests; do not break production code merely to manufacture Red.

## Workflow documentation and examples

- Update existing guides. Keep behavior contracts in specifications and development procedures in guides.
- Do not claim earlier PRs used test-first development merely because their final code includes tests.
- Use existing behavior to explain the relationship between specifications, implementation, and regression tests.
  Do not introduce deliberate defects to present a reading example as test-first development.
- Execute documented commands and verify their expected results.
  For new behavior, distinguish the intended Red failure from Green and post-refactoring success. Mark unexecuted steps as unverified.

## Specification coverage review

- State the review scope and map each behavior area to its specification, implementation, and representative tests.
- Distinguish missing specifications, missing tests, intentional exclusions, and unresolved external-service assumptions.
- Do not infer exhaustive coverage from the existence of files or tests. Host test success does not establish support for other operating systems.
- Record concrete follow-up items in `docs/specification-coverage.md`.
  Create or update GitHub issues only within the user's requested scope.
- For review or documentation requests, do not expand the task into implementing every gap found.

## Completion report

Summarize the behavior or documentation changed, validation commands and results, unverified items, and remaining work.
Distinguish an intentional Red result from successful regression checks, and never report checks that were not run as passing.
