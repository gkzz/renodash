This issue lists Renovate updates and detected dependencies. Read the [Dependency Dashboard](https://docs.renovatebot.com/key-concepts/dashboard/) docs to learn more.<br>[View this repository on the Mend.io Web Portal](https://developer.mend.io/github/gkzz/dotfiles).

## Pending Approval

The following branches are pending approval. To create them, click on a checkbox below.

 - [ ] <!-- approve-branch=renovate/toolchain -->chore(deps): update dependency pnpm to v12.14.0
 - [ ] <!-- approve-branch=renovate/major-github-actions-major -->chore(deps): update dependency ubuntu to v26
 - [ ] <!-- approve-branch=renovate/lock-file-maintenance-toolchain -->chore(deps): lock file maintenance
 - [ ] <!-- approve-all-pending-prs -->🔐 **Create all pending approval PRs at once** 🔐

## Pending Status Checks

The following updates await pending status checks. To force their creation now, click on a checkbox below.

 - [ ] <!-- unpend-branch=renovate/major-toolchain -->chore(deps): update node.js to v26

## Detected Dependencies

<details><summary>github-actions (3)</summary>
<blockquote>

<details><summary>.github/actions/setup-mise/action.yml (1)</summary>

 - `actions/cache v6.1.0@55cc8345863c7cc4c66a329aec7e433d2d1c52a9`

</details>

<details><summary>.github/workflows/lint.yml (7)</summary>

 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `rhysd/actionlint 1.7.12@sha256:b1934ee5f1c509618f2508e6eb47ee0d3520686341fec936f3b79331f9315667`
 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `ubuntu 24.04` → [Updates: `26.04`]
 - `ubuntu 24.04` → [Updates: `26.04`]
 - `ubuntu 24.04` → [Updates: `26.04`]

</details>

<details><summary>.github/workflows/test.yml (2)</summary>

 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`

</details>

</blockquote>
</details>

<details><summary>mise (1)</summary>
<blockquote>

<details><summary>mise.toml (9)</summary>

 - `biome 2.5.14`
 - `node 24.22.1` → [Updates: `24.22.2`, `26.13.0`]
 - `python 3.15.0` → [Updates: `3.15.2`]
 - `go 1.27.2` → [Updates: `1.27.3`]
 - `java 27.0.0`
 - `pnpm 12.13.0` → [Updates: `12.14.0`]
 - `ghq 1.11.2`
 - `peco 0.6.0`
 - `direnv 2.38.2`

</details>

</blockquote>
</details>

<details><summary>renovate-config (1)</summary>
<blockquote>

<details><summary>renovate.json5</summary>


</details>

</blockquote>
</details>

---
## Update review policy

Use the dashboard as a review queue. Branches and PRs are created only after approving updates from the dashboard. Merge candidates should satisfy the stability window, CI, release-note/changelog checks, and an explicit need/risk judgment. See [gkzz/renovate-config](https://redirect.github.com/gkzz/renovate-config) for the shared Renovate review policy.
