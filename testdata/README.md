# Dashboard fixtures

These three files are unmodified snapshots of GitHub issue bodies fetched on 2026-10-31:

| File | Source | Coverage |
| --- | --- | --- |
| `dotfiles.md` | [gkzz/dotfiles#3](https://github.com/gkzz/dotfiles/issues/3) | Three pending-approval entries, bulk approval, and pending status checks |
| `maestro-playground.md` | [gkzz/maestro-playground#4](https://github.com/gkzz/maestro-playground/issues/4) | Approval, schedule, and status-check controls |
| `renovate-config.md` | [gkzz/renovate-config#7](https://github.com/gkzz/renovate-config/issues/7) | A small Dashboard with two update candidates |

To update a snapshot, fetch the issue body again and record the capture date and changes. Compare control comments and counts with test expectations, and check for effects on parsing behavior. Tests never fetch fixtures automatically.

## Derived group-update fixture

`grouped-dotfiles.md` was created on 2026-11-03 from `dotfiles.md` above. Only the title for `renovate/toolchain` was replaced with ``chore(deps): update toolchain (`go`, `node`, `pnpm`, `python`)``, as presented in the [dotfiles PoC](../docs/ROADMAP.md#implemented-group-update-research-context). This is a derived fixture, not a newly fetched Dashboard body.

Current versions, available updates, managers, and files keep the original snapshot values. The fixture verifies that several Node.js candidates are not treated as a confirmed target for the group.
