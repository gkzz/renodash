# Markdown parsing command

Read Dashboard Markdown from stdin and write JSON without fetching GitHub data.
See [Dashboard parsing](../dashboard/parser.md) and [Dashboard fields](../dashboard/dashboard.md).

## Input and validation

### Parse arguments

`parse` uses Go's `flag` parser with errors returned to the caller.
Its default flag diagnostics and usage output are suppressed so `Run` reports each error once.

| Input | Rule |
| --- | --- |
| No flags | Select JSON output |
| `--format json`, `--format=json` | Select JSON output explicitly |
| `-format json`, `-format=json` | Accept the single-dash forms provided by Go's flag parser |
| Repeated format flags | Use the last value |
| Any format value other than `json` | Fail with `unsupported output format: <value>` |
| Format flag without a value | Fail with `flag needs an argument: -format` |
| Unknown flag | Fail with `flag provided but not defined: <flag>` |
| Positional argument | Fail with `unexpected argument: <first positional argument>` |
| `--` | Stop flag parsing; reject any remaining positional argument |

Flags must precede positional arguments. The parser stops at the first positional argument and does not process later flags.
Argument errors occur before reading stdin. Positional arguments are checked before the selected output format.
A separate token after `--format` is consumed as its value even if it looks like a flag.
For example, `parse --format --help` fails with `unsupported output format: --help`.

`--help` or `-h`, when encountered as a flag, prints parse usage to stdout and returns success without reading stdin.
Help stops flag processing: `parse --help extra` succeeds, while `parse extra --help` fails on `extra`.
An earlier flag parsing error still fails, as in `parse --unknown --help`.

## Behavior

Delegate Markdown recognition to the Dashboard parser after argument validation.

## Output and failures

### Parse output and diagnostics

After successful argument validation, read Markdown from stdin using the Dashboard parser.
Do not invoke GitHub commands or add a `source` field.
An empty input succeeds with empty `items`, `bulk_actions`, and `unknown_controls` arrays.

Write one warning per unknown control to stderr, in collection order, before writing JSON:

```text
warning: unsupported Renovate control: <control>
```

For `parse`, substitute the control directly using Go `%s`, without quotes or escaping.
Embedded newlines or terminal control characters remain literal in the diagnostic.
By comparison, `get` uses Go `%q` to quote and escape the control; see [fetching diagnostics](get.md#output-and-failures).
Warnings alone do not cause failure.
Unknown controls are stored in `Dashboard.UnknownControls` after the parser trims leading and trailing whitespace.
The stored values do not necessarily match the original comment text.
See [Dashboard output format](../dashboard/dashboard.md) for the output specification.

Normalize nil collections to empty arrays, then encode JSON directly to stdout with two-space indentation and a final newline.
Unlike `get`, `parse` does not prepare the entire JSON document in an intermediate buffer.

| Failure | Result |
| --- | --- |
| Argument validation or stdin parsing | Error; no JSON output |
| Warning write | Error prefixed with `write warning: `; no JSON output; earlier warnings may remain |
| JSON output write | Error prefixed with `write JSON: `; partial output may remain |

### Examples

The following examples illustrate expected behavior based on the implementation.
These commands have not been executed as part of this documentation change.

Build and run from the repository root with the development Go version:

```sh
go build -o /tmp/renodash-command-spec .
/tmp/renodash-command-spec parse < /dev/null
```

Expected stdout, with exit code `0` and empty stderr:

```json
{
  "items": [],
  "bulk_actions": [],
  "unknown_controls": []
}
```

Run an unsupported format example:

```sh
/tmp/renodash-command-spec parse --format yaml < /dev/null
```

Expect exit code `1`, empty stdout, and this stderr line:

```text
error: unsupported output format: yaml
```

These examples use a POSIX shell and `/dev/null`. They require no GitHub credentials or network access.

## Implementation and test evidence

Implementation: [parse.go](../../../../internal/command/parse.go).
Tests: [parse_test.go](../../../../internal/command/parse_test.go) cover fixtures, empty arrays, warnings, arguments, help, and I/O failures.
Exit codes follow [command dispatch](run.md#exit-codes).
Tests are offline and do not establish exhaustive platform support.
