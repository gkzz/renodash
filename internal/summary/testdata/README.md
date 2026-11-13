# Summary regression fixtures

The `.golden` files were captured with the original
`.github/scripts/renodash-summary.jq` from PR #46 (commit `6d6f2f0`):

```sh
jq -r -f .github/scripts/renodash-summary.jq INPUT.json > OUTPUT.golden
```

`offline.json` comes from the fixture-only CLI integration test. Its
`source.fetched_at` is fixed to `2026-11-13T00:00:00Z`.
`empty.json` and `escaping.json` are synthetic display cases.
Do not regenerate these expectations from the Go renderer.
