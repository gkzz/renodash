# Roadmap

renodash structures information from Renovate's Dependency Dashboard for agent-assisted research. Improvements prioritize problems observed in proofs of concept. Security assessments and final update decisions remain with people and agents.

## Status and planned work

| Status | Work | Purpose |
| --- | --- | --- |
| Done (v0.0.3) | Extract research context for group updates | Reduce the work needed to identify packages to investigate |
| In progress | Clarify evidence and sources for research context | Prevent incorrect associations by agents |
| Deferred | Consider package-name alias resolution | Handle ambiguity such as `node.js` versus `node` carefully |

v0.0.3 added group-update research context in `research.packages`, preserving the existing JSON structure for single-package updates.

## Implemented: group-update research context

Previously, `research` covered only single-package updates and was omitted for groups. In the dotfiles PoC, identifying packages in this candidate required manual work:

```text
chore(deps): update toolchain (`go`, `node`, `pnpm`, `python`)
```

The Dashboard supplied the following context. These versions are part of the recorded example, not a list of current releases.

| Package | Current version | Available update |
| --- | --- | --- |
| go | 1.27.2 | 1.27.3 |
| node | 24.22.1 | 24.22.2 |
| pnpm | 12.14.0 | 12.16.1 |
| python | 3.15.0 | 3.15.2 |

The CLI associates explicitly named group members with same-name dependency records in the Dashboard. A name match does not prove that the branch updates that record. Fixture sources, capture dates, and derivation methods are documented. Candidates are retained in `available_updates`; the group's target remains `"unknown"` because the Dashboard does not establish it.

### Implementation checks

- [x] Extract explicitly listed package names from group updates.
- [x] Extract current versions and available updates from the Dashboard.
- [x] Retain manager and file information for same-name dependencies.
- [x] Avoid associating uncertain matches by inference.
- [x] Preserve compatibility with existing `research` JSON.
- [x] Derive a fixture from the saved dotfiles snapshot using the PoC group title; document its source and creation date in [testdata/README.md](../testdata/README.md).

## Evidence and sources

Extracted facts must be distinguishable from agent research and estimates. As the [release-age triage PoC](minimum-release-age-triage/README.md) (Japanese) illustrates, publication timestamps have different meanings depending on their sources. Further PoCs will identify recurring needs before expanding the CLI's structured output.

Currently, `research.provenance` records the source within the Dashboard and the name-matching method. Consumers must record source URLs and fetch times for `parse`; `get` includes them in `source`. External publication dates and security judgments belong in research records. The next PoC will assess whether per-record line numbers and external sources are needed.

## Outside the current scope

- **Automatic security-fix classification:** an upstream security-related commit may apply to a different major version. Commit-message keywords alone will not produce `security: true`. Vulnerability assessments and final update decisions are outside the CLI's responsibility.
- **A GitHub API client and automatic PR creation:** GitHub communication continues to use `gh api`. Agents must verify whether actual branch diffs match the Dashboard.
- **Automatic resolution of ambiguous aliases:** names such as `node.js` and `node` are not linked by inference. Explicitly named group members are supported; alias resolution remains deferred.
