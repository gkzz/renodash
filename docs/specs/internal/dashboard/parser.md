# Dashboard parsing

Recognize Renovate checkbox operations from Markdown without network access.

## Input and validation

Read Markdown through an `io.Reader`.

## Behavior

### Supported controls and sections

`items` and `bulk_actions` keep checkbox actions and states. Internal types are defined in `internal/dashboard`.

- `items` records the action (`approve`, `unschedule`, or `unpend`), branch identifier, title, section, and checked state. In `Open`, `rebase-branch` becomes action `rebase`; a PR link in the title is retained as `pr_link`.
- `bulk_actions` records `approve-all-pending-prs` and, in `Open`, `rebase-all-open-prs`, along with their checked states.
- `unknown_controls` keeps unsupported or malformed checkbox control comment bodies from recognized sections, trimming surrounding whitespace and preserving duplicates. Unsupported actions such as `unlimit` and `retry`, and unsupported bulk actions, are included here.
- Titles keep their Markdown, and entries keep input order within each array.

The parser recognizes the following level-two sections, based on [Renovate's Dashboard generator](https://github.com/renovatebot/renovate/blob/main/lib/workers/repository/dependency-dashboard.ts). Closing hashes, as in `## Pending Approval ##`, are supported. Level-three or deeper category headings do not end the current section.

- `Pending Approval`, `Awaiting Schedule`, `Pending Status Checks`
- `Rate-Limited`, `Errored`, `Open`, `Other Branches`
- `Group Size Not Met`, `PR Creation Approval Required`, `Pending Branch Automerge`
- `PR Edited (Blocked)`, `PR Closed (Blocked)`
- `Config Migration Needed`, `Config Migration Needed (Blocked)`

`pr_link` preserves the relative path or URL from the Dashboard. `parse` does not know the source repository, and `get` preserves these links in its default output. Consumers can resolve links when they know the source `OWNER/REPO`.

Checkboxes in `Detected Dependencies`, explanatory sections, and unknown sections are excluded, including from `unknown_controls`. New Renovate section names require explicit parser support.

The parser also excludes checkboxes:

- Inside HTML comment blocks beginning with `<!--` at the start of a line, optionally preceded by up to three spaces.
- Inside backtick or tilde code fences.
- On lines indented with four or more spaces, or a tab.

An HTML comment ends at the first `-->`. An unclosed comment prevents parsing of the remaining input. Unknown comments are never treated as known actions. At the internal API level, an input read failure returns an empty `Dashboard` and an error.

## Output and failures

Return the [Dashboard fields](dashboard.md), with optional [research context](research.md).
On an input read failure, return an empty Dashboard and an error.

## Implementation and test evidence

Implementation: [parser.go](../../../../internal/dashboard/parser.go).
Tests: [parser_test.go](../../../../internal/dashboard/parser_test.go) cover controls, sections, ignored blocks, order, and read errors.
Saved fixtures provide offline examples; they do not establish support for future Renovate formats.
