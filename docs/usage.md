# Usage

renodash parses saved Markdown or fetches and parses a GitHub issue. See the [JSON and parsing reference](specs/internal/dashboard/dashboard.md) for output fields and supported input.

## Choose a command

| Task | Command | Requirements |
| --- | --- | --- |
| Parse a saved Dashboard | `renodash parse < dashboard.md` | A local Markdown file; no network or authentication |
| Fetch and parse an issue | `renodash get OWNER/REPO#ISSUE_NUMBER` | GitHub authentication and issue read access |
| Include linked pull request (PR) states and checks | `renodash get --with-pr-metadata OWNER/REPO#ISSUE_NUMBER` | The `get` requirements, plus [`gh` on your `PATH`](#fetch-a-github-issue) and access to linked PR metadata and checks |

Replace `OWNER/REPO#ISSUE_NUMBER` with the Dashboard's repository and issue number.
Both commands extract information. They do not change checkboxes, create PRs, or apply updates.

The commands share a Dashboard parser. `get` fetches the issue body first; `parse` reads it from standard input.
Both write JSON, and only `get --with-pr-metadata` fetches linked PR states and checks.

```mermaid
flowchart LR
    Issue[GitHub issue] --> Get[get via REST]
    Get --> Parser[Dashboard parser]
    File[Saved Markdown] --> Parse[parse via standard input]
    Parse --> Parser
    Parser --> JSON[JSON output]
```

## Installation

To install from source, use Go 1.27 or later and run this from the repository root:

```sh
go install .
```

The binary is installed in `GOBIN`, or `GOPATH/bin` if `GOBIN` is unset. The default is usually `$HOME/go/bin`. Add the appropriate directory to your `PATH`; use `go env GOBIN GOPATH` to check the configured paths.

To install the latest release without checking out the repository:

```sh
go install github.com/gkzz/renodash@latest
renodash version
```

To install without Go, download the archive for your OS and CPU from [GitHub Releases](https://github.com/gkzz/renodash/releases), extract it, and place the binary on your `PATH`. Releases include a SHA-256 `checksums.txt` file.

## Fetch a GitHub issue

`get` needs credentials with read access to the issue. To use a saved login, install GitHub CLI (`gh`) and run:

```sh
gh auth login
gh auth status
renodash get OWNER/REPO#ISSUE_NUMBER
```

Issue fetching uses the GitHub REST API directly. It checks `GH_TOKEN`, then `GITHUB_TOKEN`, then the saved `gh` login.
With an environment token, you can fetch the issue even when `gh` is absent.
If GitHub rejects the selected token, the fetch fails.
The issue request connects directly to `api.github.com` and ignores proxy environment variables.

The `source` object records the issue URL, fetch time, and issue update time.

Add `--with-pr-metadata` to fetch states, mergeability, and CI checks of PRs linked from the Dashboard.
Linked PR requests require `gh` on your `PATH`.

```sh
renodash get --with-pr-metadata gkzz/renodash#5
```

The `pull_requests` array contains each linked PR in the same repository, in first-seen order.
PRs are fetched one at a time. Temporary failures may be retried.
Each PR includes `fetched_at`, the time its response was received.
Merge states and successful checks alone do not prove that a PR is ready to merge.

See [PR output fields](specs/internal/command/pr.md#linked-pull-request-metadata) for details.

### Keep results when a PR fails

Add `--allow-partial` to keep successful PRs and report failed requests:

```sh
renodash get --with-pr-metadata --allow-partial gkzz/renodash#5
```

| Result | `retrieval.status` | Exit code |
| --- | --- | --- |
| All linked PRs fetched, or no linked PRs | `complete` | `0` |
| Some or all PRs failed | `partial` | `0` |
| Dashboard failure, invalid input, cancellation, or command timeout | No result JSON | `1` |

Read `retrieval.errors` for failed resources and attempt counts.
Without `--allow-partial`, a final PR failure returns exit code `1` without result JSON.
Temporary PR failures get up to three attempts. The five-minute limit covers Dashboard fetching, PR requests, and retry waits.
It starts after target validation. Stdin reading waits for EOF and has no time limit.

### Read a target from stdin

Put one reference, such as `gkzz/renodash#5`, in `target.txt`:

```sh
renodash get --with-pr-metadata --allow-partial --stdin < target.txt
```

- Plain `renodash get --stdin` also works.
- Do not supply a positional target with `--stdin`.
- The input limit is 4096 bytes, including spaces and a final newline.
- Agents should send target data through their execution tool's stdin, without building a shell command from it.

See [Retries and partial results](specs/internal/command/retrieval.md) for input rules, retry limits, and errors.

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

## Run in GitHub Actions

The [offline workflow](../.github/workflows/renodash.yml) tests the current PR's
renodash source with `go install .`. It runs only on pull requests when opened,
synchronized, reopened, or marked ready for review. It ignores only issue template
changes and `.github/pull_request_template.md`.

The workflow runs `get --with-pr-metadata` using the [offline process adapter](../testdata/pr-metadata/README.md#run-locally).
See that reference for fixture isolation and local execution.
No GitHub token is passed to the tested code. Checkout does not persist credentials,
and the workflow declares `permissions: {}`.

The Job Summary displays fixture candidates, PR states, and checks. Empty checks
are unknown, not success. PR metadata reflects fixture capture time, not current
GitHub state; `fetched_at` reflects test execution time. The workflow does not
perform live dependency review or make merge decisions. The `renodash-offline`
artifact stores the output JSON for 1 day.

For local fixture execution and summary rendering, see the [fixture instructions](../testdata/pr-metadata/README.md#run-locally).
The renderer labels input as saved snapshots, so use it only with offline fixture results.

Fixture refreshes are deliberate. Review their provenance, source URLs, capture
time, and captured versus synthetic data together. See the fixture README for
local execution and coverage limitations. Live review would require a separate
workflow that runs trusted code.

## Parse saved Markdown

`parse` reads Markdown from standard input. It needs neither a network connection nor GitHub authentication.
Run these examples from the repository root, where `testdata/renodash.md` is available:

```sh
renodash parse < testdata/renodash.md
renodash parse --format json < testdata/renodash.md
```

`--format` is optional; `json` is the only supported format. To run from source, use the following from the repository root:

```sh
go run . parse < testdata/renodash.md
```

The shell's `<` operator sends the file contents to standard input. It does not pass the filename as an argument.
The saved fixture produces five items, including a repeated PR link. Its `bulk_actions` and `unknown_controls` arrays are empty.
For example, the second entry in `items` is:

```json
{
  "action": "rebase",
  "branch": "renovate/fixture-51",
  "title": "[chore(deps): update toolchain](../pull/51)",
  "section": "Open",
  "checked": false,
  "pr_link": "../pull/51"
}
```

This is one item, not the complete output. `checked: false` records the unchecked box in the input.
`action: "rebase"` describes the Dashboard control; parsing does not perform a rebase.
The fixture is saved test data, not a report of current GitHub state.

### Output and exit codes

See [output conventions](specs/internal/dashboard/dashboard.md#output-conventions) for JSON output, diagnostics, and exit codes.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| `renodash` is not found | Check that the installation directory is on your `PATH`. |
| `get` fails to fetch the issue | Check the issue reference and the selected token’s read permissions. For a saved login, run `gh auth status`. |
| `items` is empty | Check whether the Dashboard has candidates in a [supported section](specs/internal/dashboard/dashboard.md#supported-controls-and-sections). |
| An unsupported-control warning appears | Inspect `unknown_controls` and the original body. Do not treat an unknown control as a supported action. |
| Expected `research` is missing | Check `renodash version` and the [supported title formats](specs/internal/dashboard/research.md). |

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

A release prints `renodash v0.0.6`, using its installed version.
A development build prints `renodash dev` and includes a commit suffix when a SHA is available.

See [build information](development.md#build-information) for how the version, commit SHA, and `dirty` marker are determined.
