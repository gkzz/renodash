# Command dispatch and parse

This specification defines command selection, help, parse arguments, diagnostics, and exit codes.
See [Dashboard fetching](get.md) for `get` arguments and [Version and build information](version.md) for version output.
See [JSON and parsing](../dashboard/output-format.md) for Dashboard recognition and output fields.

## Dispatch and help

The executable passes command-line arguments, stdin, stdout, and stderr to `command.Run`.
It uses the returned integer as the process exit code.
Command and option names are case-sensitive.

| Arguments | Behavior |
| --- | --- |
| No arguments | Fail with `expected subcommand: parse, get, or version (use --help for usage)` |
| `--help` or `-h` alone | Print top-level usage to stdout |
| `parse ...` | Parse flags, then read Markdown from stdin |
| `get ...` | Delegate to the fetching command; do not read stdin |
| `version` | Print build information; do not read stdin |
| `version --help` or `version -h` | Print version usage to stdout |
| Other arguments after `version` | Fail with `unexpected argument: <first argument after version>` |
| Any other first argument | Fail with `unknown subcommand: <first argument>` |

Top-level help requires exactly one argument. For example, `--help extra` fails as an unknown subcommand.
`get` help also requires a lone help argument after the subcommand; see its specification.
Successful help writes no diagnostics and does not parse Markdown or fetch data.
A help output write error causes failure through the normal error path.

## Parse arguments

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

## Parse output and diagnostics

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
Warnings alone do not cause failure, and JSON preserves the original unknown control strings.

Normalize nil collections to empty arrays, then encode JSON directly to stdout with two-space indentation and a final newline.
Unlike `get`, `parse` does not prepare the entire JSON document in an intermediate buffer.

| Failure | Result |
| --- | --- |
| Argument validation or stdin parsing | Error; no JSON output |
| Warning write | Error prefixed with `write warning: `; no JSON output; earlier warnings may remain |
| JSON output write | Error prefixed with `write JSON: `; partial output may remain |

## Exit codes

`Run` returns `0` when the selected command succeeds, including help and output with unsupported-control warnings.
For any returned error, it attempts to write `error: <message>` followed by a newline to stderr and returns `1`.
It does not print usage automatically after an error.
Failure to write that final diagnostic does not change the exit code or trigger another diagnostic.

## Verified examples

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

Implementation: [run.go](../../../../internal/command/run.go), with process exit wiring in [main.go](../../../../main.go).

| Tests | Representative evidence |
| --- | --- |
| `TestFixtures`, `TestEmptyDashboard` in [run_test.go](../../../../internal/command/run_test.go) | Default and explicit JSON, fixture output, empty arrays |
| `TestUnknownControls` in the same file | Literal parse warnings, retained unknown controls, successful output |
| `TestErrors`, `TestWriteError` in the same file | Dispatch, invalid flags and arguments, input and JSON write errors |
| `TestHelp` in the same file | Top-level and parse help without reading stdin |
| `TestVersionCommand` in [version_test.go](../../../../internal/command/version_test.go) | Version output, help, and rejected arguments |
| `TestOfflineCapturedMetadata`, `TestOfflineLookupErrors` in [offline_test.go](../../../../internal/integration/offline_test.go) | Real CLI entry point, offline fetching output, and process failures |

The integration tests exercise `get` through a fake `gh` process; they do not test every dispatch or parse flag rule.
The verified examples above also exercise the built CLI directly.
This evidence describes current behavior and does not establish exhaustive test coverage or cross-platform support.
