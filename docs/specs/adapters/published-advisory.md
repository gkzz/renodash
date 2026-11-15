# Published advisory evidence adapter spike

This report compares dependency versions against publicly disclosed security advisories.
It does not discover vulnerabilities, assess exploitability, or establish that a version is safe.

This NOT MERGE isolated Go experiment converts saved external evidence into a common report.
It does not extend the renodash CLI or change merge conditions.

## Input and validation

The input is one JSON object containing a PR URL, head SHA, inspected files, target identity, old and proposed versions, and source snapshots.
Each source identifies its adapter, URL, revision, UTC retrieval time, coverage, and raw payload or retrieval failure.
Reject missing provenance and malformed input before producing output.
Source URLs are provenance supplied by the caller, not proof that the payload is authentic.
No PR code, network requests, or external commands are executed by the tool.

## Behavior

An adapter converts source-specific data into assessments for each version independently.
The initial adapter supports the official Node.js runtime and the Node.js security working group's core index.
It interprets a version as affected when it matches `vulnerable` and does not match `patched`.
Use canonical input environments `linux`, `darwin`, or `win32`; normalize `macos` to `darwin` and `windows` to `win32`.
Other inputs, including `all`, produce `Unknown`; source `all` applies to every supported input.
Unrecognized source environments produce limitations rather than a confirmed non-match.
Unsupported product identities, prerelease versions, invalid ranges, and retrieval failures produce limitations.

Combine source results as follows:

- A confirmed match produces `Affected`, even when other records or sources fail.
- Without a match, any limitation produces `Unknown`.
- Otherwise, produce `NoMatchInCheckedSources` for the supplied snapshot and scope only.
- No sources produce `Unknown`.

Saved snapshots may be stale; completed queries do not establish comprehensive coverage.
Publication times can differ from retrieval times.
The adapter neither fetches nor authenticates data and does not independently identify PR products or versions.
Historical ranges may be incomplete or inconsistent; semver library behavior does not establish equivalence with every upstream interpretation.
Official binaries are distinct from npm and distribution-specific packages.
Only the declared supported distribution and environment are assessed; other lockfile artifacts require separate assessment.
This experiment does not determine exploitability or guarantee source completeness or freshness.
It does not infer that a newer release fixes an advisory without matching its published ranges.

## Output and failures

Emit JSON by default or escaped Markdown for a Job Summary file.
Preserve source provenance, ranges, advisory references, matches, and limitations for both versions.
An assessment result does not approve, block, or merge a PR.
Operational errors produce a nonzero exit code; vulnerability findings do not.
Follow the [public reporting boundaries](../../dependency-review.md#public-reporting-boundaries) before publishing output.
The maintainer remains responsible for deciding and manually merging.
Rendering escapes external text, including URLs, without creating Markdown links or fetching their destinations.
The tool cannot detect secrets or decide disclosure.
CI collection and workflow integration are outside this experiment.

## Implementation and test evidence

See the [isolated module](../../../adapters/published-advisory/) and its tests.
The module has its own dependency lock and must be tested separately from the root Go module.
