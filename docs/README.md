# Documentation

Guides, reference material, development plans, and research records for renodash. Start with [Usage](usage.md).

## Usage and development

- [Usage](usage.md): installation, commands, troubleshooting, and version output
- [Writing documentation](writing.md): English guidelines and textlint checks ([Japanese](writing.ja.md))
- [Development](development.md): environment setup, tests, and build information
- [JSON and parsing reference](output-format.md): output fields, supported controls, and research context
- [Releasing](release.md): publishing, archives, PR labels, and release notes
- [Roadmap](ROADMAP.md): existing capabilities, evidence boundaries, and PoC-driven development options
- [ロードマップの要点（日本語）](ROADMAP.ja.md): 現在の機能、次に試すこと、対象外の機能
- [Test fixtures](../testdata/README.md): snapshot sources and derived fixtures

## Agent-assisted research

- [Historical agent security triage](agent-security-triage.md): preserved external-agent PoC, outside the current product scope
- [Dashboard skill example](../examples/skills/renodash-dashboard/SKILL.md): report `get` results and their provenance

To use the example skill, copy it into your agent's skill directory, such as `.agents/skills/renodash-dashboard/SKILL.md`.

## Historical research records (Japanese)

These documents preserve findings from the time of each investigation.

- Security-specific prompts and findings are historical external-agent research, outside current product workflows. See the [scope boundary](ROADMAP.md#intentional-vulnerability-intelligence-exclusions).
- Saved Dashboard bodies and JSON may differ from current CLI output and upstream information.

### Recorded investigations

- [Release-age triage PoC](minimum-release-age-triage/README.md)
- [dotfiles research handoff](handoffs/renodash-handoff.md)
- [maestro-playground research handoff](handoffs/maestro-playground-handoff.md)

The root README is also available in [Japanese](../readme_ja.md).
