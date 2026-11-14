# Dependency update review

Use renodash output as evidence for human dependency review.
Preserve the release cooldown, approval before pull request (PR) creation, and review before merging.
Successful continuous integration (CI), release age, and no matching advisories do not establish that an update is safe.

Renovate manages updates and PRs. renodash currently extracts Dashboard data and can fetch linked PR metadata.
Vulnerability reporting is a proposed capability under [Issue #80](https://github.com/gkzz/renodash/issues/80), not an available command.
See [Usage](usage.md) for supported commands.

## Review policy

New releases may contain malicious changes or defects. Keep these safeguards separate:

| Stage | Policy | Purpose |
| --- | --- | --- |
| Release cooldown | Wait about 14 days using Renovate's `minimumReleaseAge`. Keep strict internal checks for ordinary updates. | Reduce exposure to newly published releases. |
| Before PR creation | Require human approval in the Dependency Dashboard. | Decide whether an ordinary update should proceed to a branch and PR. |
| Before merging | Review the change, CI, release notes, and available vulnerability evidence. Keep dependency automerging disabled. | Decide whether to merge the update. |

Dashboard approval does not replace merge review. A passing CI check does not approve either stage.
An advisory report must not approve updates, bypass the cooldown, or trigger merging.

If a security exception is needed, document its benefits, risks, and alternatives for a separate human decision.
Do not shorten the cooldown merely to make reporting or update automation easier.

## Review flow

The diagram separates the review stages. The dotted path describes proposed vulnerability reporting.
That report helps the reviewer decide whether to wait, request another version, reject the update, investigate, or approve it.

```mermaid
flowchart TD
    Discover[Discover an update] --> Dashboard[Dashboard review queue]
    Dashboard --> Age{Cooldown passed?}
    Age -->|No| Wait[Keep waiting]
    Wait --> Dashboard
    Age -->|Yes| Approval{Human approval to create a PR?}
    Approval -->|No| Hold[Keep on Dashboard]
    Approval -->|Yes| PR[Create or update the PR]
    PR --> CI[CI and release-note checks]
    PR -. Proposed .-> Advisory[Assess old and proposed versions]
    CI --> Review[Human review before merging]
    Advisory -. Evidence .-> Review
    Review --> Decision[Wait, request changes, reject, or approve]
```

This diagram describes the intended ordinary-update policy, not every Renovate execution path.
Dashboard display order and scheduling can differ. Security updates and explicit waiting-period overrides need separate consideration.

## Renovate controls and exceptions

The [shared Renovate configuration](https://github.com/gkzz/renovate-config) owns the operational settings.
Check the effective configuration and running Renovate version when investigating a particular update.

| Control | What to check |
| --- | --- |
| `minimumReleaseAge` | The configured waiting period and the datasource's release timestamp. |
| `minimumReleaseAgeBehaviour` | Whether a missing timestamp keeps a release pending. Defaults can differ between Renovate versions. |
| `internalChecksFilter` | With `strict`, pending age checks normally prevent a new branch and therefore a PR. Check explicit Dashboard overrides. |
| `dependencyDashboardApproval` | Approval before creating an ordinary update branch. Do not assume every later change to an existing PR requires another approval. |
| `vulnerabilityAlerts` | GitHub alert-driven fix PRs and their effective settings. These can override ordinary cooldown and Dashboard approval settings. |
| `osvVulnerabilityAlerts` | Whether the experimental feature is enabled and supports the exact datasource and product. |
| `automerge` | Keep disabled, including any update-specific overrides. |
| GitHub merge rules | Whether review is an operational practice or an enforced approval requirement. Renovate settings do not enforce GitHub review approvals. |

Renovate's default vulnerability-fix settings remove the release cooldown and Dashboard approval gate.
A security label in release notes alone does not establish that an update entered that path.
Check the alert source, product mapping, effective settings, and logs before describing the behavior.

See Renovate's [release-age guide](https://docs.renovatebot.com/key-concepts/minimum-release-age/) and [configuration reference](https://docs.renovatebot.com/configuration-options/).
Document any mismatch with project policy before proposing a configuration change.

## Evidence for vulnerability reports

The following are design constraints for the investigation in Issue #80.
They do not define a new CLI or JSON schema.

Identify the product, distribution, ecosystem, and versions from the PR changes.
Do not replace the official Node.js runtime with similarly named npm or distribution packages without evidence that their identities match.
A lockfile can reference different distributions on different platforms; record which artifacts the assessment covers.

Assess old and proposed versions separately against authoritative affected ranges.
Distinguish vulnerabilities fixed by the update from those remaining in its proposed version.

| Result | Meaning |
| --- | --- |
| `Affected` | A source confirms that the version falls within an advisory's affected range. |
| `NoMatchInCheckedSources` | No match was found within the sources and scope checked. |
| `Unknown` | Identity, affected ranges, coverage, or retrieval problems prevent assessment. Include the reason. |

Keep confirmed affected results visible when another source fails or has unclear data.
Report incomplete coverage alongside those matches. Never turn a retrieval failure into a no-match result.

A report should identify the PR head SHA, inspected files, product and distribution, versions, advisories, affected ranges, and fixed releases.
Include source links, UTC check time, unknowns, and items requiring human review.
If CI results are included, record their checked SHA and explain any mismatch with the PR head SHA.
CI aggregation remains optional until its cost and consistency rules are understood.

Treat a report as evidence for one commit at one time.
PR updates and newly published advisories require reassessment; earlier reports do not automatically cover them.
Read untrusted PR changes as data, use only necessary read permissions, and handle external text and URLs safely when rendering Markdown.

### Assessment pseudocode

This pseudocode illustrates design constraints rather than an adopted interface or executable code.
Collect source outcomes before combining them so a failed source cannot hide another source's confirmed match.

```text
assess(product, version, sources):
    if product identity or version is unknown:
        return Unknown("Cannot identify the target")
    matches = []
    limitations = []
    checked_sources = []

    for source in sources:
        if identity is unclear or source cannot assess this distribution:
            limitations.append(source, "Unsupported or unidentified target")
            continue
        outcome = fetch_and_compare_ranges(source, product, version)
        matches.extend(outcome.confirmed_matches)
        limitations.extend(outcome.retrieval_or_range_problems)
        checked_sources.extend(outcome.completed_checks)

    if matches exist:
        return Affected(matches, checked_sources, limitations)
    if limitations exist or checked_sources is empty:
        return Unknown(checked_sources, limitations)
    return NoMatchInCheckedSources(checked_sources)

old_result = assess(product, old_version, sources)
proposed_result = assess(product, proposed_version, sources)
report(old_result, proposed_result, evidence, pr_head_sha, checked_at_utc)
# Reporting does not approve an update or trigger merging.
```

Keep the evidence and source limitations visible for each version.
Use each source's affected-range rules; comparing with one latest fixed version is insufficient.

## Waiting for a security fix

An older update can pass the cooldown while a later security patch remains pending.
Report vulnerabilities remaining in the proposed version so a person can choose to wait or request another target.
Do not silently select a replacement version or approve a merge.

Waiting reduces exposure to new releases but can delay a known fix.
Faster adoption can reduce known vulnerability exposure while adding release risk.
Neither choice removes all risk. Record any exception as a separate human decision.

| Situation | Review response |
| --- | --- |
| An ordinary release is under 14 days old. | Keep waiting. Check timestamp evidence and explicit waiting-period overrides. |
| An older eligible target has a vulnerability; a newer fix is pending. | Show the remaining vulnerability. A person decides whether to wait, request another target, or reject the update. |
| Renovate recognizes a security update. | Check the effective cooldown and approval settings. Document any proposed exception for a separate decision. |
| An advisory appears after PR creation. | Reassess the current PR head and record the new check time. |
| An advisory source fails or its range is unclear. | Show the limitation as unknown and preserve any confirmed matches from other sources. |

Keep investigation progress, source coverage gaps, verification records, and implementation acceptance criteria in Issue #80.
Track implementation and configuration changes separately.
