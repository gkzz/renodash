# Documentation

Guides, reference material, and development plans for renodash. Start with [Usage](usage.md).

## Usage and development

- [Usage](usage.md): installation, commands, troubleshooting, and version output
- [Dependency update review](dependency-review.md): cooldown, human approval, and evidence for update decisions
- [GitHub Actions dependency updates](actions-lock.md): lockfile updates and verification
- [Writing documentation](writing.md): English guidelines and textlint checks ([Japanese](writing.ja.md))
- [Development](development.md): environment setup, tests, and build information
- [Testing strategy](testing.md): Go conventions, offline fixtures, and test helpers
- [Architecture Decision Records](adr/README.md): technical decisions, alternatives, and their rationale
- [Specification and TDD workflow](tdd.md): behavior changes and a cancellation regression test walkthrough
- [Releasing](release.md): publishing, archives, PR labels, and release notes
- [Roadmap](ROADMAP.md): current focus and direction
- [Roadmap summary (Japanese)](ROADMAP.ja.md): current focus and future direction
- [Test fixtures](../testdata/README.md): snapshot sources and derived fixtures

## Specifications

Store behavior specifications under `docs/specs/`. Mirror the implementation directory path below that root.
Use the implementation basename for each specification, such as `docs/specs/internal/command/validation.md` for input checks in `internal/command`.
For specifications covering several packages, use the package responsible for the main behavior and link to related specifications.
Keep usage guides, development instructions, and plans outside `specs/`.
Start with the [specification template](templates/spec-template.md) and follow the [structure guidance](writing.md#behavior-specification-structure) when writing specifications.

| Specification | Implementation |
| --- | --- |
| [Command dispatch](specs/internal/command/run.md) | `internal/command/run.go`; process exit in `main.go` |
| [Dashboard JSON fields](specs/internal/dashboard/dashboard.md) | `internal/dashboard/dashboard.go` |
| [Markdown parsing command](specs/internal/command/parse.md) | `internal/command/parse.go` |
| [Dashboard parsing](specs/internal/dashboard/parser.md) | `internal/dashboard/parser.go` |
| [Research context](specs/internal/dashboard/research.md) | `internal/dashboard/research.go` |
| [Input validation](specs/internal/command/validation.md) | `internal/command/validation.go` |
| [Dashboard fetching](specs/internal/command/get.md) | `internal/command/get.go`; exit handling in `internal/command/run.go` |
| [Retries and partial results](specs/internal/command/retrieval.md) | `internal/command/retrieval.go` |
| [Linked pull requests](specs/internal/command/pr.md) | `internal/command/pr.go` |
| [GitHub CLI client](specs/internal/command/github_client.md) | `internal/command/github_client.go` |
| [GitHub PR transport](specs/internal/command/github_pr.md) | `internal/command/github_pr.go` |
| [Version and build information](specs/internal/command/version.md) | `internal/command/version.go` |
| [Offline Actions Summary](specs/internal/summary/render.md) | `internal/summary/render.go` |

## Agent-assisted development

- [Repository planning skill](../.agents/skills/repo-gardener/SKILL.md): compare Discussion notes with existing Issues and PRs, then recommend the next PR
- [Documentation writing skill](../.agents/skills/docs-writing/SKILL.md): apply the writing guides when creating, updating, or reviewing documentation
- [Specification-driven TDD skill](../.agents/skills/tdd/SKILL.md): apply the repository's specification and TDD workflow when implementing features or fixing bugs

## Agent-assisted research

- [Dashboard skill example](../examples/skills/renodash-dashboard/SKILL.md): report `get` results and their provenance

To use the example skill, copy it into your agent's skill directory, such as `.agents/skills/renodash-dashboard/SKILL.md`.

The root README is also available in [Japanese](../readme_ja.md).
