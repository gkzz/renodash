# Labeler Permission Smoke Test

This pull request verifies whether `actions/labeler` can apply
an existing label to a pull request with the following permissions:

- `contents: read`
- `issues: write`
- `pull-requests: read`

The expected label is determined by the pull request's branch name.

This pull request is for verification only and will not be merged.