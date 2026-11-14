---
name: docs-writing
description: Create, update, or review renodash documentation, including README files, guides, behavior specifications, and contributor documentation. Apply repository writing guidelines for accessible language, structure, examples, and checks. Do not use for code-only tasks without documentation changes.
---

# Documentation writing

Make documentation accurate and understandable for readers who are not fluent in English.
Keep writing rules in the repository guides; use this skill to apply them within the requested scope.

## Repository references

The repository root is `../../../` relative to this file. Run commands from that root.

- [Writing documentation](../../../docs/writing.md): read before creating, editing, or reviewing documentation, including its visual explanation guidance and checks.
- [Japanese writing guidelines](../../../docs/writing.ja.md): also read when working on Japanese documentation. Translations are not required for every English document.
- [Documentation index](../../../docs/README.md): find related guides and behavior contracts under `docs/specs/`.
- [TDD skill](../tdd/SKILL.md): use for behavior changes and specification-driven development; keep that workflow in the existing skill.

## Workflow

1. Identify the audience, document purpose, and requested scope. Read related documentation, implementation, and representative tests.
   Preserve technical facts and terminology. Separate verified behavior from assumptions and mark unresolved claims.
2. Apply the relevant writing guide to the structure and language without changing technical meaning.
   For complex topics, consider purpose, mechanism, an example, and constraints in that order; this is not a required template.
3. Choose examples and visual explanations using the format table in the writing guide.
   Use real renodash commands, inputs, expected results, and error conditions supported by the implementation or specifications.
   Verify examples against those sources and execute commands when practical; mark unexecuted examples as unverified.
4. Review accessibility using the writing guide: explain relationships and abbreviations, use consistent terms, and preserve technical accuracy when simplifying English.
5. Verify technical claims, commands, expected output, relative links, Markdown rendering, and any Mermaid diagrams.
   Run `pnpm run lint:docs` and `git diff --check`. Follow the writing guide for prerequisites and language-specific checks.
   The existing lint script excludes `.agents/skills/`; when editing a skill, also run textlint on that file with the repository's English configuration.

## Behavior specifications

For readability reviews, check claims against the implementation and tests without expanding the task into behavior changes.
For requested behavior changes, use the TDD skill alongside this skill: TDD handles the contract and tests; this skill handles presentation.
Keep contracts in `docs/specs/` and preserve links between specifications, implementations, and tests.
Do not introduce unsupported behavior through examples or wording changes.

## Completion report

Summarize documents changed or review findings, validation commands and results, and remaining limitations.
Distinguish executed checks from manual review and unavailable checks. Never claim an unrun check passed.
