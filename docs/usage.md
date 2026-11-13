# Usage

renodash parses saved Markdown or fetches and parses a GitHub issue. See the [JSON and parsing reference](output-format.md) for output fields and supported input.

## Installation

To install from source, use Go 1.27 or later and run this from the repository root:

```sh
go install .
```

The binary is installed in `GOBIN`, or `GOPATH/bin` if `GOBIN` is unset. The default is usually `$HOME/go/bin`. Add the appropriate directory to your `PATH`; use `go env GOBIN GOPATH` to check the configured paths.

To install v0.0.5 without checking out the repository, use the `cmd/renodash` package path:

```sh
go install github.com/gkzz/renodash/cmd/renodash@v0.0.5
renodash version
```

Releases through v0.0.5 use `cmd/renodash` as the CLI entry point. Starting with v0.0.6, the CLI entry point is the repository root:

```sh
go install github.com/gkzz/renodash@v0.0.6
renodash version
```

To install without Go, download the archive for your OS and CPU from [GitHub Releases](https://github.com/gkzz/renodash/releases), extract it, and place the binary on your `PATH`. Releases include a SHA-256 `checksums.txt` file.

## Fetch a GitHub issue

`get` requires GitHub CLI (`gh`) on your `PATH` and authentication with read access to the issue.

```sh
gh auth login
gh auth status
renodash get OWNER/REPO#ISSUE_NUMBER
```

Specify the issue as `OWNER/REPO#ISSUE_NUMBER`. `get` fetches its body through `gh api` and uses the same parser as `parse`. The `source` object records the issue URL, fetch time, and issue update time.

Add `--with-pr-metadata` to fetch the state and CI checks of PRs linked from the Dashboard:

```sh
renodash get --with-pr-metadata gkzz/renodash#5
```

The result includes a `pull_requests` array. Each linked PR in the same repository is fetched once.
The Dashboard request and each unique PR request have separate 30-second timeouts.
PRs are fetched sequentially, so total runtime can exceed 30 seconds and grows with the number of linked PRs.
If a link is unsupported or a PR request fails, the command returns an error without writing JSON.
See [linked pull request metadata](output-format.md#linked-pull-request-metadata) for fields and limits.

### GitHub Actions permissions

For `--with-pr-metadata`, start with these read permissions and set `GH_TOKEN` to `${{ github.token }}` in the step environment:

```yaml
permissions:
  contents: read
  issues: read
  pull-requests: read
  checks: read
  statuses: read
```

`issues` covers the Dashboard, `pull-requests` covers PR metadata, and `checks` and `statuses` cover the two kinds of CI results.
See GitHub's [workflow permissions reference](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions).
A successful plain `get` does not verify access to PR checks.
The workflow token is scoped to its repository; use a token with access to the target repository for cross-repository requests.

This permission set has not yet been verified by this project in Actions against a private repository.
Before relying on it, run both commands below in the target repository's workflow with the permissions above:

```sh
renodash get OWNER/REPO#ISSUE_NUMBER > dashboard.json
renodash get --with-pr-metadata OWNER/REPO#ISSUE_NUMBER > dashboard-with-prs.json
```

Use a Dashboard with linked PRs that have both Actions checks and external commit statuses.
Verify that both commands succeed and that `pull_requests[].checks` contains the expected results.
Repeat in a private repository and record the run URL, repository visibility, permissions, and result.
If the second command fails, check token permissions and repository access before using its output.

### Filter candidates

With `jq`, you can display just the candidates:

```bash
set -o pipefail
renodash get OWNER/REPO#ISSUE_NUMBER | jq '.items[]'
```

Run this pipeline in Bash. `pipefail` makes a fetch or parse failure produce a nonzero pipeline exit code. Dashboards change over time and may have no candidates.

## Parse saved Markdown

`parse` reads Markdown from standard input. It needs neither a network connection nor GitHub authentication.

```sh
renodash parse < testdata/renodash.md
renodash parse --format json < testdata/renodash.md
```

`--format` is optional; `json` is the only supported format. To run from source, use the following from the repository root:

```sh
go run . parse < testdata/renodash.md
```

Both commands follow these rules:

- Leave issues and files unchanged.
- Write JSON to standard output.
- Write errors and warnings to standard error.
- Return exit code `0` on success and a nonzero code on failure.
- Keep unknown controls in the JSON and report warnings without failing the command.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| `renodash` is not found | Check that the installation directory is on your `PATH`. |
| `get` fails to fetch the issue | Check the `gh` installation, `gh auth status`, the issue reference, and read permissions. |
| `items` is empty | Check whether the Dashboard has candidates in a [supported section](output-format.md#supported-controls-and-sections). |
| An unsupported-control warning appears | Inspect `unknown_controls` and the original body. Do not treat an unknown control as a supported action. |
| Expected `research` is missing | Check `renodash version` and the [supported title formats](output-format.md#research-context-for-agents). |

Do not use output from a failed command for research. Resolve the error reported on standard error, then rerun the command. Saved Markdown can be parsed again with `parse`.

Command help is also available:

```sh
renodash --help
renodash parse --help
renodash get --help
```

## Check the version

```sh
renodash version
go run . version
```

Run `go run . version` from the repository root. The command needs no standard input, makes no network requests, and changes no files.

See [build information](development.md#build-information) for how the version, commit SHA, and `dirty` marker are determined.
