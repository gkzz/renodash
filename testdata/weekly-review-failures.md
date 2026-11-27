# Weekly review failure examples

The test adapter returned saved GitHub responses and simulated lookup failures.
The fetch time shows when renodash read the saved data.

| Scenario | Result | Exit code | Update candidates | Pull requests (PRs) | Checks |
| --- | --- | ---: | ---: | ---: | ---: |
| All lookups succeed | Complete | 0 | 5 | 4 | 28 |
| One PR lookup fails | Partial | 0 | 5 | 3 | 22 |
| All PR lookups fail | Partial | 0 | 5 | 0 | 0 |
| First Dashboard fails | Failed | 1 | — | — | — |
| Next Dashboard succeeds | Complete | 0 | 5 | 4 | 28 |

## Check now

- One PR lookup failed for #51. Results for #53, #30, and #52 were available.
- When all PR lookups failed, renodash returned errors for #53, #51, #30, and
  #52. It kept all five update candidates.
- Each PR error recorded `resource: pull_request`, `repository: gkzz/renodash`,
  `code: fetch_failed`, `attempts: 1`, and `retryable: false`.
- Check the target and access for the failed Dashboard before retrying. The
  caller recorded the error below and continued to the next target.

```text
error: fetch example/failed#1: gh command failed: exit status 1
```

## Check later

- Fetch fresh data before deciding whether to merge. The saved checks apply to
  the recorded PR commits. Both merge fields were `UNKNOWN`.
- Current PR commits, required checks, branch protection, and merge readiness
  remain unverified.

## Run details

- Source: [saved Dashboard issue #5](https://github.com/gkzz/renodash/issues/5).
  The saved issue was last updated at 2026-11-12 23:51:41 UTC.
- Fetch time for the Dashboard and available PR records: 2026-11-27 16:54:04 UTC.
- CLI: `renodash v0.0.9-0.20261127164508-a052795d9ce8+dirty`.
  The working tree adds a failure mode to the test adapter.
- Skill: [weekly review skill](../examples/skills/renodash-weekly-review/SKILL.md)
  at `a052795`.

The caller ran this command once per target and passed one reference on
standard input. The test adapter was the only `gh` on the command search path.

```sh
renodash get --with-pr-metadata --allow-partial --stdin
```
