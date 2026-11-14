# Roadmap

[Summary in Japanese](ROADMAP.ja.md)

## Current focus: make everyday use easier

For now, we are focusing on making renodash easier to use in our own development workflows.
[Issue #45](https://github.com/gkzz/renodash/issues/45) describes the starting point: try the existing CLI in consuming repositories through a minimal GitHub Actions workflow, including private repositories.

Start small, use it regularly, and improve the setup based on what we learn.
The first step is to install renodash with `go install` and extract Dashboard update candidates.
We will use this experience to identify friction in setup and everyday use.

## Direction

Keep the **Agent-first, Human-verifiable** approach.
Agents use the CLI output to investigate updates, and humans can check the supporting evidence and make update decisions.
Renovate continues to manage dependency updates and update PRs.
The [dependency review guide](dependency-review.md) describes the release cooldown, human approval stages, and constraints for proposed vulnerability reports.

Further work will take shape through actual use.
Feature scope, design, and timing remain open.

For current capabilities and output, see the [usage guide](usage.md) and [JSON and parsing reference](specs/internal/dashboard/output-format.md).
