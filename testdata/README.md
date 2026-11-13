# Dashboard fixtures

`renodash.md` is the synthetic renodash Dashboard from
[the offline PR metadata fixtures](pr-metadata/README.md). It exercises several
linked PRs, duplicate references, and grouped toolchain titles. This fixture does not represent a live
Dashboard observation. `internal/command/testdata/renodash.json` is its expected
parser output. Parser unit tests cover approval, schedule, status-check, bulk,
and unsupported controls separately.

See the fixture provenance before updating these files. Tests never refresh
GitHub data automatically.

## renovate-config Dashboard snapshot

`renovate-config.md` is the unmodified issue body from
[gkzz/renovate-config#7](https://github.com/gkzz/renovate-config/issues/7).
Captured at 2026-11-13T00:31:18Z;
the issue's `updatedAt` was 2026-11-12T23:53:26Z. Capture command:

```sh
gh issue view 7 --repo gkzz/renovate-config --json body,url,updatedAt
```

The Markdown preserves the returned body without adding a trailing newline.
It covers a pending status check, nested dependency details with counts and
blockquotes, pinned SHA and digest versions, update annotations, and a custom
review policy section after the dependencies. This snapshot records a historical
observation, not the current Dashboard or PR state.
`internal/command/testdata/renovate-config.json` records the expected parser output.
The fixture test runs offline; refreshing this snapshot is a deliberate action.

## Synthetic research regression fixtures

`grouped-research.md` and `single-research.md` are small synthetic fixtures derived
from the former dotfiles, grouped-dotfiles, and maestro-playground fixtures in
the main branch before PR #46. They are test inputs, not live Dashboard observations.

The grouped fixture preserves four packages, unresolved target versions, several
available updates, exact package association (`node.js` does not match `node`),
duplicate dependency evidence, and research provenance. The single fixture covers
explicit target versions, a `cli-` version, a Docker digest, scheduled and pending
controls, and lock file maintenance without research.

Their expected JSON files live in `internal/command/testdata/`. Expectations keep
the corresponding earlier fixture output. Command tests compare the complete JSON
for both `parse` and `parse --format json`, including research and bulk actions.
