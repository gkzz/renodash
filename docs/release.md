# Releasing

## Publish a release

The release workflow checks the code, builds archives, and publishes a GitHub Release.
Merge changes into main through a pull request (PR) before publishing.

### Publish from a tag

Run these commands from the repository root to update local main, create a tag, and push it.
`v0.1.0` is an example; replace it with the version you intend to publish.

```sh
git switch main
git pull --ff-only
git tag -a v0.1.0 -m 'Release v0.1.0'
git push origin v0.1.0
```

Tags must use `vMAJOR.MINOR.PATCH`. Prerelease tags are not supported. CI verifies that the tagged commit is contained in main.

### Publish manually

1. Open Actions → Release → Run workflow on GitHub.
2. Select `main`.
3. Enter the intended tag, such as `v0.1.0`, in `Release tag` and start the workflow.

The workflow tags the main commit selected when the run starts.
Running from another branch or reusing a tag that points to a different commit fails.

### Workflow and retries

The [Release workflow](../.github/workflows/release.yml) runs `make check`, builds distribution archives, attaches them to a draft GitHub Release, then publishes it.
No extra secrets are required.

| State | Retry behavior |
| --- | --- |
| Run failed before publication | Rerun the same run |
| Manual run reuses a draft | The draft must have a matching target commit |
| Release is already published | Rerunning fails; the workflow cannot overwrite it |

## Archives and build validation

Releases provide six binaries: amd64 and arm64 builds for Linux, macOS, and Windows. Archives include README and LICENSE, with SHA-256 hashes in `checksums.txt`. Linux and macOS use `.tar.gz`; Windows uses `.zip`.

PRs also run checks and build all six targets. The Linux binary is smoke-tested, including its `version` output. The displayed version must match the release tag; release output omits the embedded commit SHA.

To check builds locally, use Go from the [development environment](development.md#setup-and-tests), plus tar, zip, and sha256sum. Run from the repository root and specify an output directory that does not yet exist:

```sh
bash setup/release-build.sh v0.1.0 /tmp/renodash-dist
```

All binaries embed the supplied version and the SHA from `git rev-parse HEAD`. Builds include uncommitted changes, so use a clean working tree for release builds.

## PR labels and release notes

Start PR titles with `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `ci:`, or `chore:`. An optional scope, such as `fix(parser):`, is allowed.

`actions/github-script` adds `Type: ...` labels based on branch names using the [classification rules](../.github/labeler.yml). Recognized prefixes are `feat/`, `fix/`, `docs/`, `refactor/`, `test/`, `ci/`, `chore/`, and `renovate/`. `chore/deps` and `build/deps` are classified as dependency updates.

The [release-notes configuration](../.github/release.yml) groups PRs by change type, with unmatched PRs under `Other Changes`. Apply breaking-change and security-fix labels manually after reviewing the changes. Breaking Changes and Security Fixes appear first.

## Maintain automatic labeling

Labeling uses two workflows:

- [Label PR Trigger](../.github/workflows/label-pr-trigger.yml) runs on `pull_request` events `opened`, `reopened`, and `synchronize`. It uses `permissions: {}` and neither fetches nor executes PR code.
- [Label PR](../.github/workflows/labeler.yml) runs on a successful trigger through `workflow_run`, using the default branch's workflow definition. Only the labeling job receives `pull-requests: write` and `contents: read` for fetching configuration.

The second workflow validates the originating run's name, path, event, and success status. When exactly one PR is associated with the run, it fetches the PR's current state through the API. It proceeds only for an open PR in the same repository with a matching head SHA and branch. Forks and missing, ambiguous, or mismatched associations are skipped. API failures stop processing. PR numbers are never inferred from branch names.

Labels are added through the GitHub API using the validated PR number and head branch. Rules live only in `.github/labeler.yml`, stored as JSON-compatible YAML and fetched at the `workflow_run` commit (`GITHUB_SHA`). Only `head-branch` regular-expression strings are supported; other rule formats fail explicitly. Existing labels are not removed. PR code, artifacts, and caches are not fetched.

When this documentation was written, the two-workflow design had not been verified on GitHub. An [earlier successful run](https://github.com/gkzz/renodash/actions/runs/42447391752) used a different design with write permissions directly on `pull_request`.

Check the workflow definitions locally with:

```sh
actionlint .github/workflows/label-pr-trigger.yml .github/workflows/labeler.yml
```

After merging both workflows into the default branch, verify that:

1. Creating or updating a same-repository `ci/` PR adds `Type: CI` after the trigger succeeds. Check other categories such as `renovate/` as well.
2. Fork PRs, failed trigger runs, and runs with no associated PR do not change labels.
3. Closing a PR or adding a commit after the trigger completes causes a rerun of the old run to be skipped.
4. Logs show no write permissions in the trigger and no checkout, artifact retrieval, or PR-code execution in the labeling workflow.

A PR can still change between the state check and label update, so labels must not determine authorization or deployment. This design also does not prevent someone with push access from requesting permissions in a different PR workflow.
