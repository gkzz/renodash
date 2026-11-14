# renodash

[![Lint](https://github.com/gkzz/renodash/actions/workflows/lint.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/lint.yml)
[![Test](https://github.com/gkzz/renodash/actions/workflows/test.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/test.yml)

renodash extracts update candidates from Renovate's Dependency Dashboard and outputs JSON.

- Fetch a Dashboard from GitHub, or parse saved Markdown offline.
- Read update actions, branches, titles, and checkbox states.
- Get package names and versions for further research.

[Japanese](readme_ja.md)

## Installation

With Go 1.27 or later, run this from the repository root:

```sh
go install .
```

Add the installation directory to your `PATH`. Prebuilt binaries are available on [GitHub Releases](https://github.com/gkzz/renodash/releases). See the [usage guide](docs/usage.md#installation) for details.

## Quick start

Fetch and parse a GitHub issue using an authenticated GitHub CLI (`gh`):

```sh
renodash get OWNER/REPO#ISSUE_NUMBER
```

Replace `OWNER/REPO#ISSUE_NUMBER` with your Dependency Dashboard issue.

Include states and continuous integration (CI) checks for pull requests (PRs) linked from the Dashboard:

```sh
renodash get --with-pr-metadata OWNER/REPO#ISSUE_NUMBER
```

Parse a saved Dashboard Markdown file offline:

```sh
renodash parse < dashboard.md
```

Both commands output JSON with these fields:

- `items`: update candidates
- `bulk_actions`: actions for several updates
- `unknown_controls`: unsupported controls

`get` also records the issue source and timestamps in `source`. Neither command changes checkboxes, creates PRs, or applies updates.

```sh
renodash version
renodash --help
```

## Documentation

- [Usage](docs/usage.md): installation, commands, and troubleshooting
- [JSON and parsing reference](docs/specs/internal/dashboard/output-format.md): output fields, supported controls, and research extraction
- [Development](docs/development.md): setup, tests, and build information
- [Writing documentation](docs/writing.md): plain English guidelines and textlint checks
- [Releasing](docs/release.md): publishing releases and distribution archives
- [Documentation index](docs/README.md): guides, reference, and roadmap

## License

[MIT License](LICENSE)
