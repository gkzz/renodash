# Dashboard JSON fields

`parse` and `get` extract Dashboard information as JSON. See [Usage](../../../usage.md) for commands and an output example.

| Top-level field | Contents | Included by |
| --- | --- | --- |
| `items` | Update candidates | Both commands |
| `bulk_actions` | Actions for several updates | Both commands |
| `unknown_controls` | Unsupported or malformed controls in recognized sections | Both commands |
| `source` | Issue metadata and timestamps | `get` |
| `pull_requests` | Linked pull request (PR) metadata and checks | `get --with-pr-metadata` |

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

## Behavior

`Dashboard` contains `Item`, `BulkAction`, and unknown-control collections.
See [parser rules](parser.md), [research fields](research.md), [Issue source](../command/get.md#issue-source-and-timestamps),
and [PR fields](../command/pr.md#linked-pull-request-metadata).
Partial retrieval fields are defined in [retrieval](../command/retrieval.md#output-and-failures).

## Implementation and test evidence

Types and JSON tags: [dashboard.go](../../../../internal/dashboard/dashboard.go).
Parser tests: [parser_test.go](../../../../internal/dashboard/parser_test.go).
JSON fixtures and output conventions: [parse_test.go](../../../../internal/command/parse_test.go) and [get_test.go](../../../../internal/command/get_test.go).
Serialization and command failures belong to [parse](../command/parse.md) and [get](../command/get.md).
