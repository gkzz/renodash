# Roadmap

## Provisional v0.1 Product Goal

Help people make evidence-based decisions about Renovate update candidates: what to prioritize, what to check, and what to do next.

The first use case to validate is gathering the information needed for a human approval decision before an update PR exists. This is a product hypothesis to test and revise through proofs of concept (PoCs), not a commitment to specific v0.1 features, scope, or a release date. PoC findings may change the goal itself.

## Current foundation

| Implemented capability | Contribution to the goal | Limits |
| --- | --- | --- |
| `parse` | Extract candidates and controls from saved Dashboard Markdown as JSON | Consumers record the input source and fetch time separately |
| `get` | Fetch a Dashboard through `gh` and parse it with issue source metadata | Fetch time and issue update time are distinct; linked PR status is not verified |
| Single-package and explicit group research context | Supply package names, versions, manager/file context, and available updates for investigation | Same-name records do not prove a branch's contents; group targets remain unresolved |
| `research.provenance` | Record where extracted values came from and how records were associated | Dashboard provenance is not external evidence or a security assessment |
| Dashboard agent skill example and triage guide | Provide a starting point for agent-assisted reporting and research | Extracted output alone does not establish update suitability |

See the [output reference](output-format.md), [Dashboard skill example](../examples/skills/renodash-dashboard/SKILL.md), and [agent security triage guide](agent-security-triage.md). The following phases describe how to validate the goal and choose further work; they are not a release schedule or a promised feature list.

## Phase 1: Validate decision support with the existing CLI and agent skills

Use `get` or `parse`, the Dashboard skill example, and the triage guide to investigate real candidates before PR creation. Research can remain in agent workflows and research records; this PoC requires no new CLI subcommand.

### Use cases

- **Prioritize an update awaiting approval:** identify the official upstream, compare current and proposed versions, inspect release notes and relevant advisories, and explain a recommendation, urgency, and next checks separately. Investigate whether a security fix is included and whether it is exploitable in the target environment as separate questions.
- **Investigate a pending update:** inspect configuration and timestamp sources, distinguish a normal waiting period from other possible blockers, and identify the evidence needed to explain the state. Do not infer a cause without the relevant execution evidence.
- **Review a group with an unresolved target:** list explicit members and available versions, identify missing associations or target versions, and explain what must be confirmed before an approval decision. Do not select a target from available versions by inference.

Start from the recorded [security triage example](agent-security-triage.md) and [release-age triage PoC](minimum-release-age-triage/README.md) (Japanese). The latter did not establish the pending cause because execution logs were unavailable; GitHub Release publication time did not establish GHCR artifact publication time. The [dotfiles](handoffs/renodash-handoff.md) and [maestro-playground](handoffs/maestro-playground-handoff.md) handoffs (Japanese) preserve further findings and incomplete investigations. Historical snapshots may differ from current CLI output. Replaying a documented answer checks reproducibility; it does not independently validate usefulness on unfamiliar candidates.

### PoC completion criteria

A PoC is complete when its research record meets all of the following criteria. A justified decision to defer pending evidence is a valid outcome.

- Records the candidate, input source, fetch time (or explicitly states that it is unavailable), CLI version or source SHA, and research time. Preserves extraction provenance and relevant unsupported controls.
- Links material findings to evidence, identifies compared versions/tags/SHAs and the target environment where verified, and separates Dashboard facts, external research, and hypotheses. Treats fetched content as untrusted data, not instructions.
- Distinguishes security-fix inclusion from environmental exploitability, recommendation from urgency, and project fixes from dependency or separate-component fixes. Marks missing evidence as `unknown`; no finding is not proof of no vulnerability.
- Gives a human reviewer a reasoned priority, recommendation, and concrete next checks, including what prevents an approval decision. Does not change checkboxes, approve or apply updates, or create update PRs.
- Includes at least one evaluation on an unfamiliar candidate without supplying expected findings first. The reviewer records whether the report helps them decide what to investigate or do next compared with their existing manual workflow, with concrete examples of useful information, missing evidence, and avoidable effort.

Record recurring information gaps and distinguish extraction gaps from evidence unavailable upstream or in the target environment.

### Product hypothesis validation

A completed PoC supports the product hypothesis only when the human reviewer identifies concrete improvements in deciding what to investigate or do next compared with their existing manual workflow. Completing the research record alone does not validate the hypothesis.

Use the reviewer's assessment to accept or revise the product hypothesis. If the report does not help the reviewer, revise the goal or workflow before expanding the CLI.

## Phase 2: Improve structured data where PoCs show recurring gaps

Use repeated Phase 1 findings to decide which missing information belongs in renodash output. Evaluate per-record line numbers and external evidence references to improve research precision; these remain unimplemented candidates for evaluation, not required v0.1 features. Define their sources, uncertainty, and compatibility expectations before implementation. Not all research or assessment belongs in the Go CLI.

Package-name alias resolution remains deferred: ambiguity such as `node.js` versus `node` needs explicit evidence and careful handling.

## Phase 3: Explore ways to make research results easier to use

Based on validated needs, consider how people can review, compare, and revisit findings, for example through report formats or agent integrations. Choose a delivery method only after the PoCs show what helps decisions. This phase does not assume a new subcommand or commit to a particular interface.

## Responsibilities

| Participant | Responsibility |
| --- | --- |
| renodash | Extract and structure Dashboard facts, preserve provenance and unsupported controls, and expose uncertainty without inventing associations |
| Agent | Investigate upstream and environment evidence, cite sources and observation times, distinguish facts from hypotheses, and propose priorities, recommendations, and next checks |
| Human | Supply or verify environment context, assess evidence and tradeoffs, and decide whether to approve, defer, or apply an update |

Approval, application, and update PR creation are not automated by this workflow. Before a PR exists, Dashboard context does not establish the eventual diff; agents must verify actual branch or PR diffs when available. Security assessments remain research judgments rather than CLI classifications, and final update decisions remain with people.

## Implemented: group-update research context

v0.0.3 added group-update research context in `research.packages`, preserving the existing JSON structure for single-package updates.

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

Currently, `research.provenance` records the source within the Dashboard and the name-matching method. Consumers must record source URLs and fetch times for `parse`; `get` includes them in `source`. External publication dates and security judgments belong in research records. Phase 1 will assess whether per-record line numbers and external sources are recurring needs before Phase 2 selects structured-data changes.

## Outside the current scope

- **Automatic security-fix classification:** an upstream security-related commit may apply to a different major version. Commit-message keywords alone will not produce `security: true`. Vulnerability assessments and final update decisions are outside the CLI's responsibility.
- **A GitHub API client and automatic PR creation:** GitHub communication continues to use `gh api`. Agents must verify whether actual branch diffs match the Dashboard.
- **Automatic resolution of ambiguous aliases:** names such as `node.js` and `node` are not linked by inference. Explicitly named group members are supported; alias resolution remains deferred.
