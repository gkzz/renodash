# renodash

[![Lint](https://github.com/gkzz/renodash/actions/workflows/lint.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/lint.yml)
[![Test](https://github.com/gkzz/renodash/actions/workflows/test.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/test.yml)

Renovate の Dependency Dashboard を解析し、更新候補を JSON で扱うための小さな Go CLI を開発しています。

`renodash parse` で標準入力の Markdown を解析し、更新候補と一括操作を JSON として出力します。Go パッケージの `Parse(io.Reader) (Dashboard, error)` からも利用できます。

## v0.1 で実現すること

- 標準入力から Dashboard の Markdown を読み取る。
- Renovate のチェックボックスから操作種別・ブランチ・タイトル・状態を抽出する。
- 解析結果を JSON で標準出力へ書き出す。
- 3 件の Dashboard を使い、ネットワークに依存しないテストで解析結果を検証する。

GitHub API クライアント、TUI、PR 作成機能は v0.1 の対象外です。必要になるまでパッケージの分割や抽象化は行いません。

## 使用例

まず CLI をインストールします。

```sh
go install ./cmd/renodash
```

インストール先を PATH に追加したうえで実行してください。

```sh
gh api repos/gkzz/dotfiles/issues/3 --jq '.body' \
  | renodash parse --format json

renodash parse < testdata/dotfiles.md
```

GitHub からの取得は外部の `gh` コマンドで行います。`renodash` 自体は GitHub にアクセスせず、ファイルも変更しません。

| 項目 | 方針 |
| --- | --- |
| 入力 | 標準入力の Markdown |
| 出力 | 標準出力へ JSON（デフォルト形式） |
| エラー・警告 | 標準エラー出力 |
| 終了コード | 成功時は `0`、失敗時は非ゼロ |
| 未知の Renovate 制御コメント | JSON に保持し、標準エラー出力へ警告する（終了コードは `0`） |

JSON は `items`、`bulk_actions`、`unknown_controls` の配列を持ち、空の場合も `[]` を出力します。各項目のキーは `snake_case`、配列内は入力順で、出力末尾に改行が付きます。`--format` は省略可能で、対応する形式は `json` のみです。

```sh
renodash --help
renodash parse --help
go run ./cmd/renodash parse --format json < testdata/dotfiles.md
```

## パーサー

ルートパッケージ `github.com/gkzz/renodash` の `Parse` は `io.Reader` を受け取り、`Dashboard` を返します。

- `Items` に `approve`・`unschedule`・`unpend` の操作種別、ブランチ識別子、タイトル、セクション名、チェック状態を保持します。
- `BulkActions` に `approve-all-pending-prs` とチェック状態を保持します。
- `UnknownControls` に、解析対象セクション内の未対応・不正なチェックボックス制御コメントの本文を前後の空白を除いて保持します。重複も残します。`unlimit`・`retry`・`rebase` などの未対応操作や未対応の一括操作もここに格納します。
- タイトルの Markdown はそのまま保持し、各スライス内の順序は入力順です。

解析対象は、[Renovate の生成処理](https://github.com/renovatebot/renovate/blob/main/lib/workers/repository/dependency-dashboard.ts)に基づく、次のレベル2見出しです。`## Pending Approval ##` のような末尾の閉じる `#` も認識します。レベル3以下のカテゴリ見出しでも対象セクションを保持します。

- `Pending Approval`、`Awaiting Schedule`、`Pending Status Checks`
- `Rate-Limited`、`Errored`、`Open`、`Other Branches`
- `Group Size Not Met`、`PR Creation Approval Required`、`Pending Branch Automerge`
- `PR Edited (Blocked)`、`PR Closed (Blocked)`
- `Config Migration Needed`、`Config Migration Needed (Blocked)`

`Items` に追加する操作は引き続き `approve`・`unschedule`・`unpend` の3種類です。`Detected Dependencies` や説明用のセクション、未知のセクション名は対象外で、`UnknownControls` にも収集しません。Renovate が新しいセクション名を追加した場合は、解析対象の追加が必要です。

行頭（0〜3スペースのインデントを含む）の `<!--` で始まる HTML コメントブロック、バッククォートまたはチルダによるコードフェンス内と、4スペース以上またはタブでインデントされたチェックボックスは解析しません。HTML コメントは最初の `-->` で終了し、閉じ忘れた場合は以降を解析しません。未知のコメントを既知の操作として扱うことはありません。入力の読み取りに失敗した場合は、空の `Dashboard` とエラーを返します。

## 開発とテスト

Go 1.27 以降を使用します。

開発用の Go 1.27.2 は mise で管理します。以下の `make` コマンドを使うには、事前に mise をインストールしてください。

`make install` でインストール内容を確認し、`make install-apply` でツールと依存関係をインストールできます。

```sh
make go-fmt  # Go のソースコードを整形
make lint    # 整形状態と go vet を検査
make test    # テストを実行
make check   # 整形状態・静的解析・テストを検証
```

Go を直接使う場合は、次のコマンドでもテストできます。

```sh
go test ./...
```

テストでは3件の fixture の解析結果と、チェック状態、未知・不正な制御コメントの保持、コードフェンスと操作対象外セクションの除外、読み取りエラーなどを検証します。外部モジュールへの依存はありません。

GitHub Actions でも main への push、pull request、手動実行で同じコマンドを実行します。

## テストデータ

`testdata/` には Issue 本文をそのまま保存しています。取得日・出典・各 fixture の検証対象は [testdata/README.md](testdata/README.md) に記載しています。テスト中に GitHub API へアクセスしたり、fixture を自動更新したりはしません。

## 実装の順序

1. Go モジュール、README、fixture、CI を用意する（完了）。
2. `Parse(io.Reader)` と更新候補のデータモデル、パーサーの単体テストを追加する（完了）。
3. `renodash parse`、JSON 出力、CLI のエラーハンドリングを実装し、使用例を検証する（完了）。

## ライセンス

[MIT License](LICENSE)
