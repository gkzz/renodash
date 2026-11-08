# Dashboard のテストデータ

次の3ファイルは、2026-10-31 に GitHub から取得した Issue 本文のスナップショットです。本文には手を加えていません。

| ファイル | 出典 | 検証対象 |
| --- | --- | --- |
| `dotfiles.md` | [gkzz/dotfiles#3](https://github.com/gkzz/dotfiles/issues/3) | 承認待ち 3 件、一括承認、ステータスチェック待ち |
| `maestro-playground.md` | [gkzz/maestro-playground#4](https://github.com/gkzz/maestro-playground/issues/4) | 承認待ち、スケジュール待ち、ステータスチェック待ちの 3 種類の操作 |
| `renovate-config.md` | [gkzz/renovate-config#7](https://github.com/gkzz/renovate-config/issues/7) | 更新候補が 2 件の小さな Dashboard |

更新する場合は Issue 本文を取得し直し、取得日と変更内容を記録してください。制御コメントや件数の変更はテストの期待値と照らし合わせ、解析仕様への影響を確認します。テストから自動取得はしません。

## グループ更新の派生 fixture

`grouped-dotfiles.md` は、2026-11-03 に上記の `dotfiles.md` から作成したテスト用データです。`renovate/toolchain` のタイトルだけを、[dotfiles の PoC](../docs/ROADMAP.md#実装済みグループ更新の調査情報)で提示された ``chore(deps): update toolchain (`go`, `node`, `pnpm`, `python`)`` に置き換えています。新たに取得した Dashboard 原文ではありません。依存関係の現在版・更新候補・manager・file は元のスナップショットを保持し、node の複数候補をグループの更新先と断定しないことを検証します。
