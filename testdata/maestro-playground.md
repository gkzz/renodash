This issue lists Renovate updates and detected dependencies. Read the [Dependency Dashboard](https://docs.renovatebot.com/key-concepts/dashboard/) docs to learn more.<br>[View this repository on the Mend.io Web Portal](https://developer.mend.io/github/gkzz/maestro-playground).

## Pending Approval

The following branches are pending approval. To create them, click on a checkbox below.

 - [ ] <!-- approve-branch=renovate/major-github-actions-major -->chore(deps): update dependency ubuntu to v26

## Awaiting Schedule

The following updates are awaiting their schedule. To get an update now, click on a checkbox below.

 - [ ] <!-- unschedule-branch=renovate/lock-file-maintenance-toolchain -->chore(deps): lock file maintenance

## Pending Status Checks

The following updates await pending status checks. To force their creation now, click on a checkbox below.

 - [ ] <!-- unpend-branch=renovate/toolchain -->chore(deps): update dependency maestro to cli-2.12.0
 - [ ] <!-- unpend-branch=renovate/github-actions -->chore(deps): update ghcr.io/renovatebot/renovate docker tag to v44.198.2

## Detected Dependencies

<details><summary>github-actions (2)</summary>
<blockquote>

<details><summary>.github/workflows/e2e.yml (5)</summary>

 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `actions/upload-artifact v7.0.2@cf430e030ddbb5b0abf93d22962f4752f3646cd9`
 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `actions/upload-artifact v7.0.2@cf430e030ddbb5b0abf93d22962f4752f3646cd9`
 - `macos 26`

</details>

<details><summary>.github/workflows/lint.yml (10)</summary>

 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `rhysd/actionlint 1.7.12@sha256:b1934ee5f1c509618f2508e6eb47ee0d3520686341fec936f3b79331f9315667`
 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `actions/checkout v7.0.1@3d3c42e5aac5ba805825da76410c181273ba90b1`
 - `ghcr.io/renovatebot/renovate 44.17.1@sha256:90759c92793143ae7db68cd14e690109d8ace3527e17c7d188c96a1ad628282e` → [Updates: `44.198.2`]
 - `ubuntu 24.04` → [Updates: `26.04`]
 - `ubuntu 24.04` → [Updates: `26.04`]
 - `ubuntu 24.04` → [Updates: `26.04`]
 - `ubuntu 24.04` → [Updates: `26.04`]

</details>

</blockquote>
</details>

<details><summary>mise (1)</summary>
<blockquote>

<details><summary>.config/mise/config.toml (1)</summary>

 - `maestro cli-2.11.0` → [Updates: `cli-2.12.0`]

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
