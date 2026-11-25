# Research context for agents

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

`provenance` records where values came from and how they were associated.

| Value | Meaning |
| --- | --- |
| `item_title` | Extracted from the update title |
| `detected_dependencies` | Taken from same-name Dashboard dependency records |
| `exact_package_name` | Associated using an exact package-name match |
| `unresolved` in `target_version_source` | No evidence establishes the group's target version |

`reference_urls` records URLs found in the title without verifying their destinations. `parse` cannot identify the source URL or fetch time of standard input, so it does not output them. `get` records them in `source`.

Package names are matched exactly. Repeated names in a title are deduplicated in order of appearance. If no record matches, `detected_dependencies` is `[]`. Group dependency records are deduplicated only when current version, manager, file, and available updates all match; other records keep input order. Group membership is not inferred from group or branch names.

An agent can use this JSON to identify upstream projects and investigate public release notes, commit diffs, compatibility, migrations, and deprecations. `get` accesses the network through `gh` to fetch the issue and, with `--with-pr-metadata`, linked PR metadata. The CLI does not perform security assessments. Dedicated vulnerability intelligence and security assessment are outside the product scope. Incidental security-related statements in public release documentation may be recorded with attribution; they do not establish a verified security fix or environmental applicability.

## Implementation and test evidence

Implementation: [research.go](../../../../internal/dashboard/research.go).
Tests: [research_test.go](../../../../internal/dashboard/research_test.go).
Tests use saved Markdown and do not verify linked destinations.
