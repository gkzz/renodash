# renodash

[![Lint](https://github.com/gkzz/renodash/actions/workflows/lint.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/lint.yml)
[![Test](https://github.com/gkzz/renodash/actions/workflows/test.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/test.yml)

renodash is a CLI that extracts update candidates from Renovate's Dependency Dashboard. It outputs JSON with actions, branches, titles, checkbox states, and research context such as package names and versions.

[日本語](readme_ja.md)

## Installation

With Go 1.27 or later, run this from the repository root:

```sh
go install .
```

Add the installation directory to your `PATH`. Prebuilt binaries are available on [GitHub Releases](https://github.com/gkzz/renodash/releases). See the [usage guide](docs/usage.md#installation) for details.

## Quick start

Fetch and parse a GitHub issue using an authenticated GitHub CLI (`gh`):

```sh
renodash get gkzz/maestro-playground#4
```

Parse saved Markdown offline. Run this example from the repository root:

```sh
renodash parse < testdata/dotfiles.md
```

Both commands output update candidates in `items`, bulk actions in `bulk_actions`, and unsupported controls in `unknown_controls`. `get` also records the source and timestamps in `source`. Neither command changes checkboxes, creates PRs, or applies updates.

```sh
renodash version
renodash --help
```

## Documentation

- [Usage](docs/usage.md): installation, commands, and troubleshooting
- [JSON and parsing reference](docs/output-format.md): output fields, supported controls, and research extraction
- [Agent security triage](docs/agent-security-triage.md): investigating extracted candidates
- [Development](docs/development.md): setup, tests, and build information
- [Releasing](docs/release.md): publishing releases and distribution archives
- [Documentation index](docs/README.md): roadmap, proofs of concept, and research handoffs

## License

[MIT License](LICENSE)
