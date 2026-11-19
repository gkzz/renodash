# GitHub Actions dependency updates

This repository uses the shared `actions-lock` preset and records action dependencies in `.github/workflows/actions.lock`.
See the [shared guide](https://github.com/gkzz/renovate-config/blob/main/docs/actions-lock.md) for setup, updates, and verification.

- Before merging the first Renovate action update, confirm that it also updates the lockfile and passes verification.
- Review workflow changes and resolved commits together.
- Local composite actions use `./` references and contain only `run:` steps.
