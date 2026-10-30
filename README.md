# renodash

[![Lint](https://github.com/gkzz/renodash/actions/workflows/lint.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/lint.yml)
[![Test](https://github.com/gkzz/renodash/actions/workflows/test.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/test.yml)

Renovate の Dependency Dashboard を解析し、更新候補を JSON で扱うための小さな Go CLI を開発しています。

現在は Go プロジェクトの初期設定と、実際の Dashboard を使ったテストデータを用意した段階です。パーサーと CLI はまだ実装していません。

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
| 未知の Renovate 制御コメント | 既知の操作として扱わず、警告する |

JSON の構造はパーサーの実装時に定め、テストで仕様を固定します。

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

現在のテストは、3 件の fixture が読み込めることと、想定する制御コメントが含まれることを確認します。解析処理の検証はパーサーの実装時に追加します。外部モジュールへの依存はありません。

GitHub Actions でも main への push、pull request、手動実行で同じコマンドを実行します。

## テストデータ

`testdata/` には Issue 本文をそのまま保存しています。取得日・出典・各 fixture の検証対象は [testdata/README.md](testdata/README.md) に記載しています。テスト中に GitHub API へアクセスしたり、fixture を自動更新したりはしません。

## 実装の順序

1. Go モジュール、README、fixture、CI を用意する（現在）。
2. `Parse(io.Reader)` と更新候補のデータモデル、パーサーの単体テストを追加する。
3. `renodash parse`、JSON 出力、CLI のエラーハンドリングを実装し、使用例を検証する。

## ライセンス

[MIT License](LICENSE)
