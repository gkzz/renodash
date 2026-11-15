# SDK inputs and saved output

`target.json` provides a historical Go module update pair without advisory IDs or URLs.
It is a synthetic target, not an actual PR diff or a recommended update.

`live-20261115` preserves output acquired through the public Go API at module version `v1.9.0`.
`old.sdk.json` and `proposed.sdk.json` are the original SDK JSON streams.
`report.json` records module identity, queries, API configuration, completion times, digests, matches, and limitations.
`sha256.json` records the captured file digests for offline verification.
The recorded assessment completed on 2026-11-15 at 03:08:54 UTC.
SDK query configuration records the database's last modification time separately.
The SDK output is not a capture of raw upstream HTTP bytes.

The target is `golang.org/x/text`, ecosystem `Go`, distribution `upstream-module`.
Old version `v0.3.6` matched four GO records; proposed version `v0.3.7` matched three.
The official API, not local range evaluation, produced these matches.
`TestSavedOfficialSDKOutput` replays the streams without network access.
Saved advisory IDs are expected outputs, not inputs to discovery.
This does not establish complete coverage or applicability to a deployed application.

Synthetic tests use fabricated `example.org/library` records and explicitly synthetic IDs.
Their minimal records exercise stream conversion, not the official version-matching algorithm.
The canceled-context test exercises the actual API without network access.

## Data attribution and license

The database entries in `live-20261115` originate from The Go Authors / Go Vulnerability Database at [vuln.go.dev](https://vuln.go.dev/).
They are used under [Creative Commons Attribution 4.0 International (CC BY 4.0)](https://creativecommons.org/licenses/by/4.0/).
See the [upstream license statement](https://github.com/golang/vulndb/blob/master/README.md#license).
Existing record references, aliases, publication times, and modification times remain in the saved data.

The SDK streams preserve the captured API output.
`report.json` selects and arranges SDK results with adapter metadata; its database entries remain subject to CC BY 4.0.
The surrounding notes summarize the results; they do not imply endorsement by the original authors.
Database entries are provided without warranties under the license.
The repository's MIT license does not replace the database license.
