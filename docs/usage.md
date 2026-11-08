# 使い方

renodash は、保存済み Markdown の解析と、GitHub Issue の取得・解析に対応しています。出力するフィールドと解析対象は [JSON と解析の仕様](output-format.md)を参照してください。

## インストールする

ソースからインストールするには Go 1.27 以降が必要です。リポジトリのルートで実行します。

```sh
go install .
```

インストール先は `GOBIN`、未設定なら `GOPATH/bin` です。通常は `$HOME/go/bin` なので、利用環境に合わせて PATH に追加してください。配置先は `go env GOBIN GOPATH` で確認できます。

公開済みの版は、リポジトリを取得せずにインストールできます。`VERSION` を公開済みタグに置き換えてください。

```sh
go install github.com/gkzz/renodash@VERSION
renodash version
```

Go を使わずに導入する場合は、[GitHub Releases](https://github.com/gkzz/renodash/releases) から OS と CPU に合うアーカイブを取得し、展開したバイナリを PATH 上に配置します。配布物には SHA-256 の `checksums.txt` が付属します。

## GitHub Issue を取得して解析する

`get` には PATH 上の GitHub CLI `gh` と、対象 Issue を読み取れる認証が必要です。

```sh
gh auth login
gh auth status
renodash get gkzz/maestro-playground#4
```

対象は `OWNER/REPO#ISSUE_NUMBER` 形式で指定します。`get` は `gh api` で本文を取得し、`parse` と同じパーサーで解析します。取得元 URL・取得日時・Issue の最終更新日時は `source` に出力します。

`jq` があれば、更新候補だけを表示できます。

```bash
set -o pipefail
renodash get gkzz/maestro-playground#4 | jq '.items[]'
```

このパイプラインは Bash で実行してください。`pipefail` により、取得や解析の失敗をパイプラインの終了コードで検出できます。Dashboard は変化するため、候補がない場合もあります。

## 保存済み Markdown を解析する

`parse` は標準入力から Markdown を読み取ります。ネットワーク接続や GitHub の認証は不要です。

```sh
renodash parse < testdata/dotfiles.md
renodash parse --format json < testdata/dotfiles.md
```

`--format` は省略でき、対応する形式は `json` のみです。チェックアウトしたソースから実行する場合は、リポジトリのルートで次を使います。

```sh
go run . parse < testdata/dotfiles.md
```

`parse` と `get` は Issue やファイルを変更しません。解析結果は標準出力、エラーと警告は標準エラーに出力します。成功時の終了コードは `0`、失敗時は非ゼロです。未知の制御コメントは警告と JSON に残し、成功として終了します。

## エラーや空の結果を確認する

| 状況 | 確認すること |
| --- | --- |
| `renodash` が見つからない | インストール先が PATH に含まれているか確認する |
| `get` で取得に失敗する | `gh` のインストール、`gh auth status`、対象の表記と読み取り権限を確認する |
| `items` が空 | 元の Dashboard に更新候補があるか、[解析対象のセクション](output-format.md#解析する操作とセクション)かを確認する |
| 未知の制御コメントの警告が出る | `unknown_controls` と元の本文を確認する。既知の操作として扱わない |
| 期待する `research` がない | `renodash version` と[対応するタイトル形式](output-format.md#エージェント向けの調査情報)を確認する |

失敗時の出力は調査に使わず、標準エラーの原因を解消して再実行してください。保存済み Markdown は `parse` で再解析できます。

使い方は CLI からも確認できます。

```sh
renodash --help
renodash parse --help
renodash get --help
```

## バージョンの確認

```sh
renodash version
go run . version
```

`go run . version` はリポジトリのルートで実行してください。標準入力は不要で、CLI はネットワークアクセスやファイルの変更も行いません。

表示する版・コミット SHA・`dirty` の決め方は、[開発ガイド](development.md#バージョン情報の決め方)を参照してください。
