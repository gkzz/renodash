# Command dispatch

Select a command and report its exit code. See [Markdown parsing](parse.md), [Dashboard fetching](get.md), and [Version output](version.md).

## Behavior

### Dispatch and help

The executable passes command-line arguments, stdin, stdout, and stderr to `command.Run`.
It uses the returned integer as the process exit code.
Command and option names are case-sensitive.

| Arguments | Behavior |
| --- | --- |
| No arguments | Fail with `expected subcommand: parse, get, or version (use --help for usage)` |
| `--help` or `-h` alone | Print top-level usage to stdout |
| `parse ...` | Parse flags, then read Markdown from stdin |
| `get ...` | Delegate to the fetching command; read a target from stdin only with `--stdin` |
| `version` | Print build information; do not read stdin |
| `version --help` or `version -h` | Print version usage to stdout |
| Other arguments after `version` | Fail with `unexpected argument: <first argument after version>` |
| Any other first argument | Fail with `unknown subcommand: <first argument>` |

Top-level help requires exactly one argument. For example, `--help extra` fails as an unknown subcommand.
`get` help also requires a lone help argument after the subcommand; see its specification.
Successful help writes no diagnostics and does not parse Markdown or fetch data.
A help output write error causes failure through the normal error path.

## Output and failures

### Exit codes

`Run` returns `0` when the selected command succeeds, including help and output with unsupported-control warnings.
For any returned error, it attempts to write `error: <message>` followed by a newline to stderr and returns `1`.
It does not print usage automatically after an error.
Failure to write that final diagnostic does not change the exit code or trigger another diagnostic.

## Implementation and test evidence

Implementation: [run.go](../../../../internal/command/run.go), with process exit wiring in [main.go](../../../../main.go).
Tests: `TestErrors` and `TestHelp` in [run_test.go](../../../../internal/command/run_test.go).
Command-specific tests remain beside each command implementation.
