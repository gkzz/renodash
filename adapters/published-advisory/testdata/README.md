# Saved evidence inputs

These files support offline reproduction. They are not current vulnerability assessments.

## PR snapshot

`pr53.json` combines these read-only observations:

- PR: https://github.com/gkzz/renodash/pull/53
- Head SHA: `cc4ccda827937e4d7b9ce164899d3781acd04085`
- Inspected diff: `.config/mise/config.toml` and `.config/mise/mise.lock`
- Old and proposed Node.js versions: `24.22.2` and `26.13.0`
- Source: https://github.com/nodejs/security-wg/blob/f643f84d4977d0ba379dc7c5805c9b65206369b5/vuln/core/index.json
- Source revision: `f643f84d4977d0ba379dc7c5805c9b65206369b5`

The payload contains all 204 records from that saved index.
The target covers official Linux artifacts only, not every artifact in the lockfile.
The lockfile also includes musl artifacts from unofficial-builds, macOS, and Windows.
Those artifacts require separate scope decisions.

The source's latest indexed records concern October 2026 releases.
This does not establish coverage of November security releases.
Missing fields in eight older records currently produce `Unknown` for both PR versions.
Do not remove those records merely to get a no-match result.

## Boundary example

`boundary.json` uses the real record `99` from the same saved source revision.
Its CVE is `CVE-2022-32222`.
The target versions `18.9.0` and `18.9.1` are synthetic test inputs, not PR #53's diff.
The PR reference and file list are reused as input metadata for the test harness.
This fixture does not claim that PR #53 changes either of these versions.

The record declares `14.x || 16.x || 18.x` vulnerable and `^14.20.0 || ^16.20.0 || ^18.9.1` patched.
Its affected environment is Linux.
The expected results are `Affected` for `18.9.0` and `NoMatchInCheckedSources` for `18.9.1` within this single-record scope.
The source description contains historical inconsistencies; this experiment reports the structured ranges without resolving them.

Unit tests add malformed and failed-source inputs to exercise `Unknown` and partial coverage.
No live API calls occur during tests.

## Reproduced PR results and November example

`TestPRSnapshot` expects zero matches and eight limitations for each version, yielding `Unknown`.
The incomplete record IDs are `2`, `27`, `28`, `29`, `33`, `34`, `35`, and `38`.
It checks identity, inspected files, source revision, record count, and the November coverage warning.
This proves reproducible processing of the saved snapshot, not complete assessment of the November release.

Read-only verification of the saved commit confirmed the version change and official Linux download URLs:
https://github.com/gkzz/renodash/commit/cc4ccda827937e4d7b9ce164899d3781acd04085
The current PR head observed on 2026-11-15 differs (`fce9674eff75ad287b0adb7fdc9d7b5665779f1f`).
The fixture describes the saved commit only; it must not be relabeled as a current assessment.

`CVE-2026-123863.json` preserves the publicly disclosed OpenJS CNA record retrieved on 2026-11-15 at 00:52:16 UTC.
Source: https://raw.githubusercontent.com/CVEProject/cvelistV5/main/cves/2026/123xxx/CVE-2026-123863.json
Its publication and update timestamp is `2026-11-11T18:16:12.733Z`.
The SHA-256 of the saved bytes appears in `pr53-cve.json` as the snapshot identifier; no upstream commit was pinned.

The structured affected intervals are `>=24.0.0 <24.22.3` and `>=25.0.0 <26.13.1`.
The narrative instead gives starts of `24.22.0` and `26.9.0` for the corresponding release lines.
Both `24.22.2` and `26.13.0` fall within either representation, so this discrepancy does not change these two matches.
Do not generalize the transcription to versions where these representations disagree.

Official release evidence identifies fixes in `24.22.3` and `26.13.1`:

- https://nodejs.org/en/blog/vulnerability/november-2026-security-releases/
- https://nodejs.org/en/blog/release/v24.22.3/
- https://nodejs.org/en/blog/release/v26.13.1/

`pr53-cve.json` manually transcribes these explicit CNA intervals into the existing adapter's input shape.
The fixture is a single-CVE experiment with manual transcription.
Its `provider` and `transformation` distinguish OpenJS CNA from the core-shaped parser.
The `nodejs-core-range-transcription` adapter preserves the CVE record ID without an NSWG prefix.
Automatic CNA support and upstream Security WG index changes remain outside its scope.
The `patched` exclusion expresses the published upper boundaries; the transcription was not inferred from a later release alone.
`TestPR53PublishedCVE` reproduces `Affected` for both actual PR versions within this single-record scope.
This does not amend the original 204-record fixture or establish exploitability, deployment exposure, or coverage of other November CVEs.

The saved environment vocabulary contains `all` (193 records), `win32` (9), `linux` (2), and six other identifiers.
Those identifiers are `aix`, `darwin`, `freebsd`, `openbsd`, `sunos`, and `android`, each appearing once.
One record can list several environments; these counts describe labels, not separate advisories.
Only Linux, macOS, and Windows inputs are supported; no other distribution support is inferred.

## Test-only data

Inline environment records in `TestEnvironments` are fabricated and use `example.invalid` URLs.
They test matching behavior and do not describe published vulnerabilities.
Synthetic test times may use August or September 2026; they are not retrieval or verification evidence.
The boundary example combines a real saved record with synthetic target versions, as described above.
The PR snapshots and saved CNA record remain separate evidence with their actual provenance dates.
