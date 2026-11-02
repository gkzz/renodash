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

## バージョンの確認

```sh
renodash version
# renodash v0.0.2 (commit: 0123456789abcdef...)

go run ./cmd/renodash version
# renodash dev (commit: unknown)
```

上記は表示形式の例であり、その版の公開を示すものではありません。標準入力は不要で、ネットワークアクセスやファイルの変更も行いません。

バージョンとコミット SHA はそれぞれ、リリースビルド時に `-ldflags -X` で埋め込んだ値を優先します。不足する情報は標準ライブラリの `runtime/debug.ReadBuildInfo` で補います。`go install github.com/gkzz/renodash/cmd/renodash@vX.Y.Z` のようにタグを指定したインストールでは、取得できるモジュールバージョンを表示します。

ローカルの `go run` や通常の `go build` など、タグ付きモジュールのバージョンを取得できない場合は `dev` と表示します。Go の擬似バージョンや `+dirty` 付きのモジュールバージョンも `dev` として扱い、正式リリースとは見なしません。コミットは埋め込み値、次に Go の `vcs.revision` を使い、どちらもなければ `unknown` と表示します。擬似バージョンから SHA を推測しません。VCS 情報を含まないビルド（`-buildvcs=false` など）や、タグ指定の `go install` でも SHA を取得できない場合があります。情報が不明でも成功として終了します。

実行時に `git` は呼び出しません。表示する SHA はビルド元コミットを示しますが、ビルド時の未コミットの変更内容は表しません。`dev` であっても、同じ SHA のビルドが同じ内容であるとは限りません。

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

`Items` に追加する操作は引き続き `approve`・`unschedule`・`unpend` の3種類です。`Detected Dependencies` や説明用のセクション、未知のセクション名のチェックボックスは対象外で、`UnknownControls` にも収集しません。Renovate が新しいセクション名を追加した場合は、解析対象の追加が必要です。

行頭（0〜3スペースのインデントを含む）の `<!--` で始まる HTML コメントブロック、バッククォートまたはチルダによるコードフェンス内と、4スペース以上またはタブでインデントされたチェックボックスは解析しません。HTML コメントは最初の `-->` で終了し、閉じ忘れた場合は以降を解析しません。未知のコメントを既知の操作として扱うことはありません。入力の読み取りに失敗した場合は、空の `Dashboard` とエラーを返します。

## エージェント向けの調査情報

単体更新と判別できるタイトルには、任意の `research` フィールドを追加します。PR が未作成でも、Dashboard 本文だけで調査材料を取得できます。

```json
{
  "package": "pnpm",
  "target_version": "v12.14.0",
  "detected_dependencies": [
    {
      "current_version": "12.13.0",
      "manager": "mise",
      "file": "mise.toml"
    }
  ],
  "reference_urls": []
}
```

- `package` と `target_version` は `Update dependency NAME to VERSION`、`Update NAME to VERSION`、Docker の `Update NAME docker tag to VERSION` 形式から抽出します。任意のコミット接頭辞と、名前・版のバッククォート、Markdown リンクに対応します。版の `v` 接頭辞は保持します。
- `detected_dependencies` は `Detected Dependencies` の manager・ファイルの2階層から、パッケージ名が完全一致する記録を収集します。同じ現在版・manager・ファイルの組み合わせは重複を除き、異なる記録は入力順に保持します。現在版は digest を含む元の表記を保持します。
- 名前の別名解決やブランチ名からの推測はしません。例えばタイトルの `node.js` と検出名の `node` は結び付けません。同名記録は調査候補であり、そのブランチで更新されることを保証しません。
- `reference_urls` はタイトルにある HTTP(S) の Markdown リンクの URL です。上流リポジトリやリリース URL は推測しません。入力にリンクがなければ空配列です。
- 記録が見つからない場合も配列は `[]` です。グループ更新・lock file maintenance・未対応形式では `research` 自体を省略し、元のタイトルを保持します。

この JSON をローカルのエージェントに渡し、パッケージの上流を特定したうえでリリースノート、コミット差分、GHSA、間接依存を調査できます。CLI はネットワークにアクセスせず、セキュリティ評価も行いません。調査結果では「更新にセキュリティ修正が含まれること」と「利用環境で悪用可能であること」を分けて扱ってください。

抽出から調査までの実行例、エージェント向けプロンプト、結果の期待形式は [エージェントによるセキュリティ調査](docs/agent-security-triage.md) を参照してください。

既存 CLI とローカルエージェントで更新の保留理由を調べた例は、[Minimum Release Age Triage PoC](docs/minimum-release-age-triage/README.md) を参照してください。日時の出典を区別し、ログで確認できない原因は未確定として記録します。

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

## リリース

`.github/workflows/release.yml` は `v0.1.0` のようなタグの push または Actions の GUI からの手動実行を契機に、`make check`、配布用ビルド、GitHub Release の公開を実行します。`.github/release.yml` は、自動生成するリリースノートの分類を設定します。

Linux・macOS・Windows の amd64/arm64 向けバイナリを配布します。アーカイブには README と LICENSE を同梱し、SHA-256 の `checksums.txt` を添付します。Linux・macOS は `.tar.gz`、Windows は `.zip` です。

リリースする変更を PR 経由で main にマージしてから、最新の main にタグを付けて push します。

```sh
git switch main
git pull --ff-only
git tag -a v0.1.0 -m 'Release v0.1.0'
git push origin v0.1.0
```

タグは `vMAJOR.MINOR.PATCH` 形式のみ対応し、main に含まれるコミットであることを CI で確認します。プレリリースタグは未対応です。GUI から実行する場合は、Actions → Release → Run workflow でブランチに `main` を選び、`Release tag` に `v0.0.1` などを入力します。実行開始時の main のコミットを対象としてタグを作成し、リリースを公開します。main 以外からの手動実行と、別のコミットを指す既存タグの再利用はエラーになります。

PR でも検証・6種類のビルド・Linux バイナリの動作確認を行います。タグの push と GUI からの手動実行で公開し、追加のシークレットは不要です。成果物をドラフトに添付してから公開し、途中で失敗した場合は同じワークフローを再実行するとドラフトの処理を再開します。手動実行でドラフトを再利用する場合は対象コミットの一致も確認します。公開済みのリリースは上書きせず、再実行はエラーになります。

ビルドだけをローカルで確認する場合は、Go・tar・zip・sha256sum が使える環境で以下を実行します。出力先には未作成のディレクトリを指定します。

```sh
bash setup/release-build.sh v0.1.0 /tmp/renodash-dist
```

このスクリプトは指定されたバージョンと、ビルド時の `git rev-parse HEAD` の SHA を6種類すべてのバイナリに埋め込みます。ローカルでは未コミットの変更もビルドに含まれるため、公開用ビルドは変更のない作業ツリーで実行してください。CI は Linux アーカイブから取り出したバイナリの `version` 出力が、指定バージョンと checkout したコミットに一致することも検証します。

## テストデータ

`testdata/` には Issue 本文をそのまま保存しています。取得日・出典・各 fixture の検証対象は [testdata/README.md](testdata/README.md) に記載しています。テスト中に GitHub API へアクセスしたり、fixture を自動更新したりはしません。

## 実装の順序

1. Go モジュール、README、fixture、CI を用意する（完了）。
2. `Parse(io.Reader)` と更新候補のデータモデル、パーサーの単体テストを追加する（完了）。
3. `renodash parse`、JSON 出力、CLI のエラーハンドリングを実装し、使用例を検証する（完了）。

## ライセンス

[MIT License](LICENSE)
