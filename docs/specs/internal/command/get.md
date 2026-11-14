# Dashboard fetching

This specification describes issue fetching, response validation, and output ordering for `get`.
See [Usage](../../../usage.md) for invocation examples, [Input validation](validation.md) for issue reference rules,
and [JSON and parsing](../dashboard/output-format.md) for output fields and Dashboard parsing rules.

## Input and validation

- Accept exactly one `OWNER/REPO#ISSUE_NUMBER` argument after removing `--with-pr-metadata` options.
- Recognize `get --help` and `get -h` when the help option is the only argument. Print help to stdout without fetching.
- Check the reference before invoking `gh`.

### Response validation

Decode the response as JSON before parsing its Markdown body. Ignore unrelated JSON fields.

| Field | Required behavior |
| --- | --- |
| `number` | Must equal the requested integer issue number |
| `html_url` | Must identify the requested issue using the URL rules below |
| `updated_at` | Must be nonempty and parse as RFC 3339 |
| `body` | Must be a non-null string containing at least one non-whitespace character |

An empty or whitespace-only body fails. A nonempty body with no supported update candidates can succeed with empty arrays.
Pass the original body to the Dashboard parser without trimming it.

### Issue URL rules

- Require the `https` scheme and host `github.com`; compare the host without case sensitivity.
- Reject user information, explicit ports, query strings (including an empty `?`), and nonempty fragments.
- Require the escaped path to match `/OWNER/REPO/issues/NUMBER` without a trailing slash.
- Compare the repository name without case sensitivity. Require the literal lowercase `/issues/` segment.
- Use the parsed integer's decimal representation for `NUMBER`. For example, input `#0005` requires URL suffix `/issues/5`, not `/issues/05`.
- Do not decode escaped path characters to make a repository match.

Set `source.repository` from the accepted URL, preserving its repository casing.
Preserve the returned URL and `updated_at` string in `source.issue_url` and `source.issue_updated_at`.
Record `source.fetched_at` in UTC RFC 3339 after the successful request and before response validation.

## Behavior

### Fetching

- Fetch with `gh api repos/OWNER/REPO/issues/NUMBER`, using the input repository casing and the parsed integer number.
- Apply the [request budget and cancellation rules](github-client.md#deadlines-and-cancellation).
- Read the Dashboard from the fetched body, without reading stdin.
- Fetch linked PR metadata only when `--with-pr-metadata` is present. See [Linked pull requests](pr.md) for link resolution and response validation.

## Output and failures

On success, emit one indented JSON document followed by a newline.
Represent missing item, bulk-action, and unknown-control collections as empty arrays.
Include `pull_requests` only when requested.

Prepare the JSON in memory before writing warnings or stdout. Write all unsupported-control warnings to stderr before writing JSON to stdout.
Each warning uses `warning: unsupported Renovate control: ` followed by the control formatted with Go `%q` and a newline.
Escape embedded newlines and terminal control characters in warnings. Preserve the original control strings as JSON values.
Unsupported controls alone do not cause a failure.

| Failure | Result |
| --- | --- |
| Invalid arguments, fetch failure, or timeout | Error; no JSON output |
| Invalid JSON, mismatched metadata, invalid timestamp, or empty body | Error; no JSON output |
| Dashboard parse failure or requested PR metadata failure | Error; no JSON output |
| Warning write failure | Error; stdout remains empty; earlier warnings may have been written |
| Stdout write failure | Error; the destination may contain partial JSON |

`Run` returns exit code `0` on success. On failure it reports `error: <message>` to stderr and returns `1`.
Output buffering prevents fetch and validation failures from publishing partial results; it cannot make destination writes atomic.

### GitHub CLI errors

See [GitHub CLI client](github-client.md) for command execution, error normalization, and diagnostic handling.

## Implementation and test evidence

Implementation: `executeGet`, `fetchIssue`, and `issueRepositoryFromURL` in [get.go](../../../../internal/command/get.go).
Exit handling: `Run` in [run.go](../../../../internal/command/run.go).

The following tests are in [get_test.go](../../../../internal/command/get_test.go):

| Tests | Covered behavior |
| --- | --- |
| `TestGet` | Endpoint, source fields, UTC fetch time, JSON arrays, and warnings |
| `TestGetIssueURLCase`, `TestMatchesIssueURL` | Repository casing and accepted or rejected URL forms |
| `TestGetErrors`, `TestGetParseError` | Invalid references, fetch and parse failures, bad JSON, empty body, and mismatched issue number |
| `TestGetResponseValidationBoundaries` | Missing, null, empty, and malformed timestamps; missing/null bodies; empty query markers and URL user information; failure without stdout |
| `TestGetStdoutWriteFailure` | Wrapped destination errors, exit diagnostics, and failures before or after partial JSON output |
| `TestGetTimeoutLeavesStdoutEmpty` | Presence of a request deadline and handling of a timeout error |
| `TestGetEscapesWarningsBeforeJSON`, `TestGetEscapesNewlineInWarning` | Warning escaping, write order, and original JSON control values |
| `TestGetWarningWriteFailureLeavesStdoutEmpty` | No stdout output after a warning write failure |

These tests use injected responses or a fake `gh`; they do not establish live GitHub availability.
The timeout test checks the remaining budget; it does not wait for the 30-second duration.
Transport and late-response tests are described in [GitHub CLI client](github-client.md).

These tests provide representative evidence rather than exhaustive coverage of every validation branch.
