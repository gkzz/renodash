# Offline Actions Summary

`RenderOffline` converts saved renodash JSON into Markdown for the offline Actions Summary.
It does not fetch GitHub data or decide whether a pull request (PR) can merge.
The output describes the saved-snapshot test scenario, including its constructed Dashboard and capture-time limitations.
This is not a general renderer for live Dashboard observations.

See [JSON and parsing](../dashboard/output-format.md) for command output fields.
The renderer validates only the subset described below; successful rendering does not check the complete command output contract.

## Input and validation

Read exactly one JSON value followed by end of input. Trailing whitespace is accepted.
Ignore fields outside the renderer's input structure.
The value must decode into the expected object structure; incompatible field types fail decoding.
Missing fields and JSON null values leave zero values in the fresh input structure.
Apply these validation rules before writing any output:

| Field | Rule |
| --- | --- |
| Root value and `source` | Must be non-null |
| `source.issue_number` | Must be positive |
| `source.issue_url` | Must pass the URL rules below |
| `source.fetched_at` | Must be nonempty; no timestamp parsing or whitespace-only rejection |
| Each `pull_requests` entry | Require a positive `number` and a URL passing the rules below |

No other nonempty requirements apply to candidate text, PR titles, or PR states.
PR states are displayed without checking a state enumeration.
Missing or null `is_draft` becomes `false`.
Missing or null collections have length zero.
The renderer does not match PR numbers to URL paths, require repository agreement, or deduplicate PRs.
Bulk actions and unknown controls are decoded as raw JSON entries and used only for counts.

## URLs

Apply the same URL validator to the issue URL and every PR URL:

- Reject any literal Unicode whitespace or control character before URL parsing.
- Require a parsed URL with scheme `http` or `https`, a nonempty hostname, and no user information.
- Reject URL parsing errors, including malformed escapes.
- Do not require GitHub as the host or check issue and PR path shapes.
- Do not reject queries, fragments, or ports solely because they are present.

Serialize the parsed URL, then encode these Markdown delimiters:

| Character | Replacement |
| --- | --- |
| `(`, `)` | `%28`, `%29` |
| `<`, `>` | `%3C`, `%3E` |
| Backslash | `%5C` |
| Pipe | `%7C` |
| `[`, `]` | `%5B`, `%5D` |
| Backtick | `%60` |

Existing percent-encoded whitespace, such as `%20`, is accepted.
These rules differ from PR link resolution during [Dashboard fetching](../command/get.md), which restricts fetched links to the Dashboard's GitHub repository.

## Display and escaping

Write a heading identifying saved GitHub snapshots and explain that observations reflect capture time rather than current GitHub state.
The introduction identifies the Dashboard body as constructed test input and points to `testdata/pr-metadata/README.md` for provenance.

Show the issue link, `fetched_at` as **Test execution**, and counts for candidates, bulk actions, unknown controls, and linked PRs.
The **Test execution** label belongs to the offline test scenario.
The renderer displays the supplied `source.fetched_at` value without obtaining or verifying when the test ran.
Counts use the full input collections, even when candidate rows are truncated.
Bulk action and unknown control details are not displayed.

Render candidate section, action, and title in input order, with at most 20 rows.
For no candidates, show a `No candidates` row.
For more than 20 candidates, show the first 20 and a notice directing readers to the artifact for full results.

Render all PRs in input order, without a display limit.
Each row contains a numbered link, state, lowercase Boolean draft value, title, and checks.
With no PRs, state that no linked PRs were found and metadata verification was skipped.
With PRs, include the notice that check observations are not merge approval.

Escape candidate text, PR state and title, check names and states, and `fetched_at` using these replacements:

| Input | Output |
| --- | --- |
| `&`, `<`, `>` | `&amp;`, `&lt;`, `&gt;` |
| Backslash | `&#92;` |
| `*`, `_` | `&#42;`, `&#95;` |
| `[`, `]` | `&#91;`, `&#93;` |
| Pipe | `&#124;` |
| Backtick | `&#96;` |

Then replace each consecutive run of carriage returns and line feeds with one space.
Existing entity text is escaped again: `&amp;` becomes `&amp;amp;`.
This function does not otherwise normalize whitespace or remove control characters from display text.
URL destinations use the separate URL rules above.

## Check fallbacks

Preserve check order within each PR. Format each check as `name: state` and join checks with literal `<br>` separators.

1. For the name, use non-null `name`, then non-null `context`, then `Unnamed check`.
   An empty but non-null name wins over context.
2. For the state, use the first nonempty value among `conclusion`, `status`, and `state`, then `Unknown`.
   These values are not checked against an enumeration.
3. If the checks collection is missing, null, or empty, show `Unknown (no checks returned)`.

The renderer decodes check fields into its own typed structure.
Unlike the command's raw check output, incompatible check field types can fail summary decoding.

## Failures and output

| Failure | Error |
| --- | --- |
| Initial JSON decoding or reader failure | `decode offline summary: <error>` |
| A second valid JSON value | `offline summary: expected one JSON value` |
| Invalid trailing input or trailing reader failure | `decode trailing input: <error>` |
| Missing required source metadata | `offline summary: missing source metadata` |
| Invalid issue URL | `issue URL: <error>` |
| Nonpositive PR number | `PR %d: missing number`, substituting the zero-based entry index for `%d` |
| Invalid PR URL | `PR %d URL: <error>`, substituting the supplied PR number for `%d` |
| Writer failure | `write offline summary: <error>` |

For example, a nonpositive number in the first PR entry returns `PR 0: missing number`.
PR 7 with a relative URL returns `PR 7 URL: expected an absolute HTTP(S) URL`.

Check the input, including every PR URL, before constructing the Markdown in a buffer.
On validation or input errors, do not call the output writer.
After rendering, call the writer once with the complete buffer.
If it reports fewer bytes than the buffer length without an error, return a wrapped `io.ErrShortWrite`.
Writer failures can leave partial output; the renderer does not roll back bytes already written.
Wrapped reader and writer errors remain available through Go error unwrapping.

## Implementation and test evidence

Implementation: `RenderOffline`, `linkURL`, and `safe` in [render.go](../../../../internal/summary/render.go).
The offline adapter in [main.go](../../../../internal/tools/fake-actions-summary/main.go) calls the renderer with stdin and stdout.
The [offline workflow](../../../../.github/workflows/renodash.yml) uses that adapter to generate summary Markdown.

These tests are in [render_test.go](../../../../internal/summary/render_test.go):

| Test | Representative evidence |
| --- | --- |
| `TestGolden` | Exact empty, escaping, and offline Markdown against independent expectations |
| `TestCandidateLimits` | Counts, order, empty output, and 0/1/20/21 candidate boundaries |
| `TestCheckFallbacks` | Name and state precedence, null fields, empty names, and unnamed checks |
| `TestInvalidInput` | Invalid roots and field types, extra values, malformed trailing input, and accepted trailing whitespace |
| `TestURLs` | Invalid issue/PR URLs, literal whitespace, user information, malformed escapes, delimiter encoding, and preserved `%20` |
| `TestIOErrors` | Reader failure, writer failure, short writes, and error unwrapping |
| `TestSafe` | Text escaping, existing entities, and newline-run replacement |

The [golden fixture notes](../../../../internal/summary/testdata/README.md) record their origin in the original jq renderer.
Do not regenerate those expectations from the Go renderer.

In [summary_test.go](../../../../internal/integration/summary_test.go), `TestSummaryEmpty`, `TestSummaryMissingChecks`, and `TestSummaryEscapingAndChecks` exercise summary behavior across fixture-shaped input.
`TestOfflineSummaryAdapter` builds and runs the adapter, checking successful output, invalid input, and unexpected arguments.
`TestOfflineSummaryGolden` compares offline CLI results, after fixing the execution timestamp, against the independent golden output.
`TestOfflineEmptyChecks` in [offline_test.go](../../../../internal/integration/offline_test.go) connects missing command checks to the summary fallback.

This evidence is representative, not exhaustive.
Some validation details, including nonpositive numbers and the absence of timestamp or repository matching, follow from implementation review rather than dedicated boundary tests.
Offline tests do not establish live GitHub state or support on other operating systems.
