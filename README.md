# renodash

[![Lint](https://github.com/gkzz/renodash/actions/workflows/lint.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/lint.yml)
[![Test](https://github.com/gkzz/renodash/actions/workflows/test.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/test.yml)

Renovate の Dependency Dashboard から更新候補を抽出する Go CLI です。操作種別、ブランチ、タイトル、チェック状態に加え、パッケージ名や版などの調査材料を JSON で出力します。

## インストール

Go 1.27 以降を用意し、リポジトリのルートで実行してください。

```sh
go install .
```

インストール先を PATH に追加すると、`renodash` を実行できます。公開済みバイナリは [GitHub Releases](https://github.com/gkzz/renodash/releases) から取得できます。導入方法の詳細は[使い方](docs/usage.md#インストールする)を参照してください。

## 基本的な使い方

GitHub Issue を取得するには、認証済みの GitHub CLI `gh` が必要です。

```sh
renodash get gkzz/maestro-playground#4
```

保存済みの Markdown は、ネットワークに接続せず解析できます。次の例はリポジトリのルートで実行します。

```sh
renodash parse < testdata/dotfiles.md
```

どちらも更新候補を `items`、一括操作を `bulk_actions`、未対応の制御コメントを `unknown_controls` に出力します。`get` は取得元と日時を `source` に追加します。Issue のチェック操作や PR 作成、更新の適用は行いません。

```sh
renodash version
renodash --help
```

## ドキュメント

- [使い方](docs/usage.md)：導入、コマンドの使い方、エラーの確認
- [JSON と解析の仕様](docs/output-format.md)：出力フィールド、対応する操作、調査情報の抽出条件
- [エージェントによるセキュリティ調査](docs/agent-security-triage.md)：抽出結果を使う調査手順
- [開発ガイド](docs/development.md)：開発環境、テスト、バージョン情報
- [リリースガイド](docs/release.md)：公開手順と配布物
- [文書一覧](docs/README.md)：ロードマップ、PoC、調査の引き継ぎ記録

## ライセンス

[MIT License](LICENSE)
