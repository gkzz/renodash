# Documentation

Guides, reference material, and development plans for renodash. Start with [Usage](usage.md).

## Usage and development

- [Usage](usage.md): installation, commands, troubleshooting, and version output
- [Writing documentation](writing.md): English guidelines and textlint checks ([Japanese](writing.ja.md))
- [Development](development.md): environment setup, tests, and build information
- [Testing strategy](testing.md): Go conventions, offline fixtures, and test helpers
- [Releasing](release.md): publishing, archives, PR labels, and release notes
- [Roadmap](ROADMAP.md): current focus and direction
- [Roadmap summary (Japanese)](ROADMAP.ja.md): current focus and future direction
- [Test fixtures](../testdata/README.md): snapshot sources and derived fixtures

## Specifications

Store behavior specifications under `docs/specs/`. Mirror the implementation directory path below that root.
Use a descriptive filename for the behavior, such as `docs/specs/internal/command/validation.md` for input checks in `internal/command`.
For specifications covering several packages, use the package responsible for the main behavior and link to related specifications.
Keep usage guides, development instructions, and plans outside `specs/`.

| Specification | Implementation |
| --- | --- |
| [JSON and parsing](specs/internal/dashboard/output-format.md) | `internal/dashboard`; command output in `internal/command` |
| [Input validation](specs/internal/command/validation.md) | `internal/command/validation.go` |
| [Dashboard fetching](specs/internal/command/get.md) | `internal/command/get.go`; exit handling in `internal/command/run.go` |
| [Version and build information](specs/internal/command/version.md) | `internal/command/version.go` |

## Agent-assisted research

- [Dashboard skill example](../examples/skills/renodash-dashboard/SKILL.md): report `get` results and their provenance

To use the example skill, copy it into your agent's skill directory, such as `.agents/skills/renodash-dashboard/SKILL.md`.

The root README is also available in [Japanese](../readme_ja.md).
