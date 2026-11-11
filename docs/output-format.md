# JSON and parsing reference

`parse` and `get` output update candidates in `items`, bulk actions in `bulk_actions`, and unsupported controls in `unknown_controls`. `get` adds issue metadata in `source`. See [Usage](usage.md) for commands.

## Output conventions

| Property | Behavior |
| --- | --- |
| Input | `parse`: Markdown on standard input; `get`: `OWNER/REPO#ISSUE_NUMBER` |
| Output | JSON on standard output |
| Errors and warnings | Standard error |
| Exit code | `0` on success; nonzero on failure |
| Unknown Renovate controls | Retained in JSON with a warning; exit code remains `0` |

The output follows these conventions:

- `items`, `bulk_actions`, and `unknown_controls` are always arrays. Empty arrays use `[]`.
- Keys use `snake_case`.
- Entries keep input order within each array.
- Output ends with a newline.
- The optional `parse --format` flag supports only `json`.

## Issue source and timestamps

`get` includes these fields in `source`:

| Field | Meaning |
| --- | --- |
| `repository` | `OWNER/REPO` from the issue URL returned by GitHub; does not preserve the input's letter case |
| `issue_number` | Issue number |
| `issue_url` | GitHub issue URL |
| `fetched_at` | Fetch time in UTC, formatted as RFC3339 |
| `issue_updated_at` | Issue update time returned by the GitHub API |

Fetch time and issue update time are distinct. The other top-level arrays have the same structure as `parse` output. Errors and unsupported-control warnings go to standard error; fetch and parse failures produce no JSON. `parse` does not output `source`.

## Supported controls and sections

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

`pr_link` preserves the relative path or URL from the Dashboard. `parse` does not know the source repository, and `get` neither converts links to absolute URLs nor checks current PR status. Consumers can resolve links when they know the source `OWNER/REPO`.

Checkboxes in `Detected Dependencies`, explanatory sections, and unknown sections are excluded, including from `unknown_controls`. New Renovate section names require explicit parser support.

The parser also excludes checkboxes:

- Inside HTML comment blocks beginning with `<!--` at the start of a line, optionally preceded by up to three spaces.
- Inside backtick or tilde code fences.
- On lines indented with four or more spaces, or a tab.

An HTML comment ends at the first `-->`. An unclosed comment prevents parsing of the remaining input. Unknown comments are never treated as known actions. At the internal API level, an input read failure returns an empty `Dashboard` and an error.

## Research context for agents

Supported single-package updates and groups with explicit package names receive an optional `research` field on each item. Research context can be extracted from the Dashboard even before a PR exists.

### Single-package updates

Example `research` value:

```json
{
  "package": "pnpm",
  "target_version": "v12.14.0",
  "detected_dependencies": [
    {
      "current_version": "12.13.0",
      "manager": "mise",
      "file": "mise.toml"
    }
  ],
  "reference_urls": [],
  "provenance": {
    "package_source": "item_title",
    "target_version_source": "item_title",
    "detected_dependencies_source": "detected_dependencies",
    "association": "exact_package_name",
    "reference_url_source": "item_title"
  }
}
```

- `package` and `target_version` are extracted from `Update dependency NAME to VERSION`, `Update NAME to VERSION`, and Docker's `Update NAME docker tag to VERSION` forms. Optional commit prefixes, backticks around names and versions, and Markdown links are supported. A version's `v` prefix is preserved.
- `detected_dependencies` collects exact package-name matches from the manager/file hierarchy in `Detected Dependencies`. Records with the same current version, manager, and file are deduplicated; distinct records keep input order. Current versions preserve their original representation, including digests.
- There is no alias resolution or inference from branch names. For example, `node.js` in a title is not matched to `node` in detected dependencies. A matching name is research context, not proof that the branch updates that dependency record.
- `reference_urls` contains HTTP(S) Markdown link URLs from the title. Upstream repositories and release URLs are not inferred. If the title has no links, the array is empty.
- Missing records produce empty arrays. Groups without explicit package lists, lock file maintenance, and unsupported title formats omit `research` and keep the original title.

### Group updates

Supported groups end with a comma-separated list of backtick-quoted package names in parentheses, such as ``chore(deps): update toolchain (`go`, `node`, `pnpm`, `python`)``. Single-package JSON keeps its existing structure; groups place per-package context in `research.packages`.

```json
{
  "packages": [
    {
      "package": "node",
      "target_version": "unknown",
      "detected_dependencies": [
        {
          "current_version": "24.22.1",
          "manager": "mise",
          "file": "mise.toml",
          "available_updates": ["24.22.2", "26.13.0"]
        }
      ]
    }
  ],
  "reference_urls": [],
  "provenance": {
    "package_source": "item_title",
    "target_version_source": "unresolved",
    "detected_dependencies_source": "detected_dependencies",
    "available_updates_source": "detected_dependencies",
    "association": "exact_package_name",
    "reference_url_source": "item_title"
  }
}
```

`current_version` is the version reported by the Dashboard. `available_updates` contains candidates from the same dependency record. Candidates can include versions intended for a separate major update, so even a single candidate does not establish the group's target. `target_version` remains `"unknown"`. If no candidates are listed, `available_updates` is `[]`.

### Provenance and matching

`provenance` records where values came from and how they were associated. `item_title` means the update title, `detected_dependencies` means same-name Dashboard records, and `exact_package_name` means an exact name match. For groups, `target_version_source: "unresolved"` means there is no evidence establishing the target version.

`reference_urls` records URLs found in the title without verifying their destinations. `parse` cannot identify the source URL or fetch time of standard input, so it does not output them. `get` records them in `source`.

Package names are matched exactly. Repeated names in a title are deduplicated in order of appearance. If no record matches, `detected_dependencies` is `[]`. Group dependency records are deduplicated only when current version, manager, file, and available updates all match; other records keep input order. Group membership is not inferred from group or branch names.

An agent can use this JSON to identify upstream projects and investigate public release notes, commit diffs, compatibility, migrations, and deprecations. `get` accesses the network through `gh` only to fetch the issue. The CLI does not perform security assessments. Dedicated vulnerability intelligence and security assessment are outside the [product scope](ROADMAP.md#intentional-vulnerability-intelligence-exclusions). Incidental security-related statements in public release documentation may be recorded with attribution; they do not establish a verified security fix or environmental applicability.

The [historical agent security triage](agent-security-triage.md) preserves external-agent examples, prompts, and expected reports outside the current product scope. The [release-age triage PoC](minimum-release-age-triage/README.md) (Japanese) shows an investigation of pending updates using the existing CLI and a local agent. Keep timestamp sources distinct and leave causes unconfirmed when execution logs are unavailable.
