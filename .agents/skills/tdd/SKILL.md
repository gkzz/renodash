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
- [Testing strategy](../../../docs/testing.md): read when adding or changing tests.
- [Development](../../../docs/development.md): check prerequisites and validation commands.
- [Specification index](../../../docs/README.md#specifications): find the corresponding contract under `docs/specs/`.
- [Writing documentation](../../../docs/writing.md): read when changing guides or specifications.

## Apply the workflow

For features and bug fixes, follow the [TDD workflow](../../../docs/tdd.md#red--green--refactor) and its specification update rules.
For workflow documentation and examples, follow the same guide and distinguish observed test-first results from reading examples of existing tests.
Use the [testing strategy](../../../docs/testing.md) for test conventions and the writing guide for documentation changes.

## Specification coverage review

- State the review scope and map each behavior area to its specification, implementation, and representative tests.
- Distinguish missing specifications, missing tests, intentional exclusions, and unresolved external-service assumptions.
- Do not infer exhaustive coverage from the existence of files or tests. Host test success does not establish support for other operating systems.
- Track concrete coverage gaps and follow-up work in GitHub Issues rather than documentation status snapshots.
  Create or update GitHub issues only within the user's requested scope.
- For review or documentation requests, do not expand the task into implementing every gap found.

## Completion report

Summarize the behavior or documentation changed, validation commands and results, unverified items, and remaining work.
Distinguish an intentional Red result from successful regression checks, and never report checks that were not run as passing.
