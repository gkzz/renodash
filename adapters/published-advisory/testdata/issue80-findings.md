# Issue #80 findings and decision draft

Copy-ready research record for Issue #80. PR #82 remains **NOT MERGE**.
Verification time: `2026-11-15T00:52:16Z`.

## Findings against acceptance criteria

| Criterion | Evidence and status |
| --- | --- |
| Identify PR #53 | Verified saved commit `cc4ccda827937e4d7b9ce164899d3781acd04085`: `.config/mise/config.toml` and `.config/mise/mise.lock` change Node.js `24.22.2` to `26.13.0`. Official Linux URLs establish the selected distribution. Other lockfile artifacts remain outside scope. |
| Reproduce a real CVE for both versions | `pr53-cve.json` and `TestPR53PublishedCVE` reproduce `Affected` for both versions for `CVE-2026-123863`. This is a manual transcription of public CNA ranges, not automatic acquisition. |
| Compare coverage and delay | The 204-record Security WG snapshot reaches October but does not establish November coverage. Eight incomplete records yield `Unknown`. November vendor/CNA evidence provides one additional match. Wider source comparison and Renovate mapping remain incomplete. |
| Show report states | The separate synthetic boundary fixture demonstrates `Affected` and `NoMatchInCheckedSources`; the saved PR snapshot demonstrates `Unknown`. JSON and escaped Markdown preserve evidence, SHA, UTC time, scope, and limitations. CI SHA is absent because CI is not collected. |
| Execution and reassessment decisions | No workflow is implemented. Trusted execution, triggers, permissions, stale-result handling, and CI aggregation require decisions and validation before any automation. |
| Recommend an approach | Prefer manual official-advisory review for now. This draft does not close Issue #80 or authorize production implementation. |

## Reproducible evidence

Run from `adapters/published-advisory` with Go 1.27 or later:

```sh
go test ./... -count=1
go vet ./...
go run . < testdata/pr53.json
go run . -format markdown < testdata/pr53-cve.json
```

The first snapshot yields `Unknown` for both versions, with zero matches and eight missing-field limitations each.
Do not remove incomplete records to get a no-match result.
The original snapshot was retrieved on `2026-11-14T14:13:04Z`, after the November release's publication.
Retrieval time cannot establish that a source includes recently published advisories.

The real-CVE example uses the [OpenJS CNA record](https://raw.githubusercontent.com/CVEProject/cvelistV5/main/cves/2026/123xxx/CVE-2026-123863.json).
Its structured intervals are `>=24.0.0 <24.22.3` and `>=25.0.0 <26.13.1`.
Both old `24.22.2` and proposed `26.13.0` match.
The [vendor advisory](https://nodejs.org/en/blog/vulnerability/november-2026-security-releases/) and release notes identify fixed versions `24.22.3` and `26.13.1`.
The CNA publication/update timestamp is `2026-11-11T18:16:12.733Z`.
The saved raw record's SHA-256 is recorded in the input fixture; the moving source URL is not a pinned upstream revision.

The narrative starts at `24.22.0` and `26.9.0`, unlike the structured intervals.
Both PR versions match either representation, but general equivalence is unverified.
This is a confirmed published-range comparison for those versions, not proof that the repository or deployment is exploitable.
See [fixture notes](README.md) for provenance, boundaries, and reproduction limits.
The current PR head differs from the saved SHA; reassess it separately before using the results for a current decision.

## What the spike establishes

**Demonstrated behavior:** Saved Node.js data can be converted through a source-specific boundary into shared assessments and JSON or Markdown.
Confirmed matches survive record/source failures, with limitations preserved.
Environment aliases require normalization: `windows` maps to `win32`, and `macos` maps to `darwin`.
Unknown inputs produce `Unknown`; source `all` does not extend distribution support.
The nested module requires separate tests; root `go test ./...` omits it.

**Documented upstream behavior:** Public vendor releases and CNA records describe published issues and affected versions.
They do not establish local exposure. The repository review guide describes Renovate security exceptions that need effective-configuration verification.

**Experimental findings:** A recently retrieved Security WG snapshot can omit evidence from a newer release.
Historical missing fields and inconsistent range representations prevent a simple comprehensive no-match claim.
The CNA example improves evidence for one CVE without fixing source completeness.

**Unverified hypotheses:** OSV, GitHub Advisory Database, Dependency Review, and Renovate may cover some relevant records.
Their exact official-runtime mapping, publication delays, completeness, and failure behavior were not established by this experiment.
An npm `node` or distribution `nodejs` record cannot substitute for official binaries without identity evidence.
Masterminds semver compatibility with every upstream range interpretation remains unverified.

## Decision proposal and remaining work

Do not integrate this spike into renodash or add a vulnerability gate.
Use a manual checklist with official advisory links, exact PR SHA, versions, distribution, environment, and UTC verification time for now.
This may meet the maintainer's needs without production renodash code; its practical effort should be evaluated with another real update.
Automatic matching works for selected explicit ranges, but reliable comprehensive matching remains unproven.

Before proposing automation, compare vendor/CNA, Security WG, OSV, GitHub Advisory Database, Dependency Review, and effective Renovate behavior for the same product and CVE.
Measure publication versus retrieval times and record missing, conflicting, failed, and unsupported cases.
Decide whether source selection and manual evidence entry meet review needs before building more adapters.

**Future proposals, not implemented:** Run trusted code from a trusted revision and treat PR changes only as data.
Use minimal read permissions for public evidence and PR metadata; define exact permissions after selecting acquisition methods.
Do not execute untrusted PR code with privileged tokens.
Reassess on head changes and new advisories; identify stale reports rather than reusing them silently.
Decide triggers and source freshness requirements before scheduling checks.
If CI aggregation is useful, preserve each checked SHA and explicitly handle mismatches, failed, skipped, and missing checks.
No CI acquisition, automatic advisory acquisition, trusted workflow, or automatic freshness enforcement exists in the spike.

A public Job Summary must contain only public advisory information and reviewed metadata.
Private Dependabot alerts require an explicit disclosure decision; unpublished findings and secrets must not be exposed.
Escape external text and URLs, distinguish proposed versions from deployment facts, and preserve unknowns and coverage limitations.
The maintainer remains responsible for the decision and merges manually.
The 14-day cooldown, Renovate configuration, rulesets, and merge settings remain unchanged.

Issue #80 remains open: broader source comparison, Renovate verification, execution decisions, and a final approach decision are outstanding.
No GitHub issues or comments were posted or edited for this draft.

## Additional validation before Job Summary integration

The `Test renodash offline` workflow renders offline CLI fixture JSON through `fake-actions-summary`.
It does not run this advisory adapter or establish advisory findings for the triggering PR's proposed version.
Treat that display experiment separately from the advisory assessment.
The following checks remain integration work; this list does not claim they passed.

| Priority | Check | Required evidence |
| --- | --- | --- |
| P1 | PR and assessment identity | Reject a saved #53 fixture as another PR's assessment, including a changed head or proposed version. |
| P1 | Execution trust | Execute assessment code from a trusted revision; inspect PR changes as data without executing them. |
| P2 | PR head and CI SHA | Preserve both identifiers and explain a merge-commit or other mismatch. |
| P2 | Failed CI | Exercise conditional summary execution, failed or skipped checks, and missing report files. Preserve advisory results or explicit unknown reasons. |
| P2 | Source provenance | Record origin, revision, retrieval and assessment times, acquisition failures, authenticity checks, and freshness limits. Caller labels alone are insufficient. |
| P2 | Public output | Review disclosure, hostile Markdown and URLs, and output size limits before publication. Escaping does not detect private information. |
| P3 | Reassessment | Mark old results stale after PR changes or new advisory evidence; define triggers and freshness rules before automation. |

### Assessment cases to verify

These cases guide additional tests and integration review.
Existing adapter tests cover selected ranges, environment handling, incomplete evidence, and preservation of matches.
They do not establish the integration guarantees above.

| Case | Expected result or boundary |
| --- | --- |
| Version within an affected range | `Affected` with the matching range and source. |
| Version clearly outside checked ranges | `NoMatchInCheckedSources` only when no unresolved limitations remain. |
| Product, OS, or distribution differs | Exclude an explicitly inapplicable record; return `Unknown` for unsupported or ambiguous target identity. |
| Snapshot omits a new CVE | Preserve snapshot scope and freshness limits; never claim comprehensive absence of vulnerabilities. |
| Sources publish conflicting ranges | Preserve both sources and ranges, and expose the conflict. Automatic conflict detection is not established by this spike. |
| One source fails while another matches | Keep `Affected` and the failure limitation. |
| Saved PR identity or head differs from the current PR | Detect the mismatch before publication; do not label the fixture as a current assessment. This check is not implemented. |

The saved CNA example illustrates false-positive risk: structured ranges start at `24.0.0` and `25.0.0`, while the narrative starts at `24.22.0` and `26.9.0`.
Older versions may therefore receive different results depending on the representation used.
Both #53 versions fall within both representations; this discrepancy does not change that example's matches.
Independent verification of actual impact remains outside scope.

The eight incomplete Security WG records produce `Unknown` rather than an unsupported negative result.
This conservative behavior can reduce practical usefulness; do not remove limitations merely to obtain a no-match result.
See the [assessment limitations contract](../../../docs/specs/adapters/published-advisory.md#false-positives-false-negatives-and-assessment-limitations).

Closing Issue #80 need not require zero false positives or false negatives.
The decision should record known error risks, cases reported as `Unknown`, and remaining unverified assumptions.
Keep PR #82 **NOT MERGE** and validate these boundaries before evaluating public summary integration.
