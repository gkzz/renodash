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
