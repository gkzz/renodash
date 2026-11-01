# renodash

[![Lint](https://github.com/gkzz/renodash/actions/workflows/lint.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/lint.yml)
[![Test](https://github.com/gkzz/renodash/actions/workflows/test.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/test.yml)

Renovate の Dependency Dashboard を解析し、更新候補を JSON で扱うための小さな Go CLI を開発しています。

`Parse(io.Reader) (Dashboard, error)` で Markdown から更新候補と一括操作を抽出できます。CLI はまだ実装していません。

## v0.1 で実現すること

- 標準入力から Dashboard の Markdown を読み取る。
- Renovate のチェックボックスから操作種別・ブランチ・タイトル・状態を抽出する。
- 解析結果を JSON で標準出力へ書き出す。
- 3 件の Dashboard を使い、ネットワークに依存しないテストで解析結果を検証する。

GitHub API クライアント、TUI、PR 作成機能は v0.1 の対象外です。必要になるまでパッケージの分割や抽象化は行いません。

## 使用例（実装予定）

以下は完成後のインターフェースです。現時点では実行できません。

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
| 未知の Renovate 制御コメント | パーサーで保持し、CLI 実装時に標準エラー出力へ警告する |

JSON の構造は CLI を実装する際に定めます。

## パーサー

ルートパッケージ `github.com/gkzz/renodash` の `Parse` は `io.Reader` を受け取り、`Dashboard` を返します。

- `Items` に `approve`・`unschedule`・`unpend` の操作種別、ブランチ識別子、タイトル、セクション名、チェック状態を保持します。
- `BulkActions` に `approve-all-pending-prs` とチェック状態を保持します。
- `UnknownControls` に、未対応・不正な制御コメントの本文を前後の空白を除いて保持します。重複も残します。
- タイトルの Markdown はそのまま保持し、各スライス内の順序は入力順です。

解析対象は `Pending Approval`・`Awaiting Schedule`・`Pending Status Checks` の3セクションです。`Detected Dependencies` や説明用のセクション、バッククォートまたはチルダによるコードフェンス内は解析しません。未知のコメントを既知の操作として扱うことはありません。入力の読み取りに失敗した場合は、空の `Dashboard` とエラーを返します。

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
3. `renodash parse`、JSON 出力、CLI のエラーハンドリングを実装し、使用例を検証する。

## ライセンス

[MIT License](LICENSE)
