# renodash

[![Lint](https://github.com/gkzz/renodash/actions/workflows/lint.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/lint.yml)
[![Test](https://github.com/gkzz/renodash/actions/workflows/test.yml/badge.svg)](https://github.com/gkzz/renodash/actions/workflows/test.yml)

Renovate の Dependency Dashboard を解析し、更新候補を JSON で扱うための小さな Go CLI を開発しています。

`renodash parse` で標準入力の Markdown を解析し、`renodash get` で GitHub Issue を取得して解析します。どちらも更新候補と一括操作を JSON として出力します。Go パッケージの `Parse(io.Reader) (Dashboard, error)` からも利用できます。

## v0.1 で実現すること

- 標準入力から Dashboard の Markdown を読み取る。
- Renovate のチェックボックスから操作種別・ブランチ・タイトル・状態を抽出する。
- 解析結果を JSON で標準出力へ書き出す。
- 3 件の Dashboard を使い、ネットワークに依存しないテストで解析結果を検証する。

GitHub SDK、TUI、PR 作成機能は v0.1 の対象外です。必要になるまでパッケージの分割や抽象化は行いません。

## 使用例

まず CLI をインストールします。

```sh
go install ./cmd/renodash
```

インストール先を PATH に追加したうえで実行してください。

```sh
renodash get gkzz/maestro-playground#4 | jq '.items[]'

gh api repos/gkzz/dotfiles/issues/3 --jq '.body' \
  | renodash parse --format json

renodash parse < testdata/dotfiles.md
```

`get` には PATH 上の GitHub CLI `gh` と `gh auth login` 等による認証が必要です。`get` は `gh api` を使って Issue 本文を取得し、既存の `Parse` で解析します。`parse` は標準入力だけを解析するため、オフライン解析や保存済み Markdown の再解析に使えます。どちらも Issue やファイルを変更しません。

`get` の JSON には `source` が加わります。`repository` は GitHub が返した Issue URL に基づく `OWNER/REPO` 表記（入力の大文字・小文字は引き継ぎません）、`issue_number` は Issue 番号、`issue_url` は GitHub の Issue URL、`fetched_at` は取得時刻（UTC、RFC3339）、`issue_updated_at` は GitHub API が返す Issue 最終更新時刻です。取得時刻と最終更新時刻は異なります。`items`、`bulk_actions`、`unknown_controls` は `parse` と同じ構造です。エラーと未知の制御コメントの警告は標準エラーへ出力し、失敗時は JSON を出力しません。`parse` の JSON 仕様は変わりません。

| 項目 | 方針 |
| --- | --- |
| 入力 | `parse`: 標準入力の Markdown、`get`: `OWNER/REPO#ISSUE_NUMBER` |
| 出力 | 標準出力へ JSON（デフォルト形式） |
| エラー・警告 | 標準エラー出力 |
| 終了コード | 成功時は `0`、失敗時は非ゼロ |
| 未知の Renovate 制御コメント | JSON に保持し、標準エラー出力へ警告する（終了コードは `0`） |

JSON は `items`、`bulk_actions`、`unknown_controls` の配列を持ち、空の場合も `[]` を出力します。各項目のキーは `snake_case`、配列内は入力順で、出力末尾に改行が付きます。`--format` は省略可能で、対応する形式は `json` のみです。

```sh
renodash --help
renodash parse --help
renodash get --help
go run ./cmd/renodash parse --format json < testdata/dotfiles.md
```

## バージョンの確認

```sh
renodash version
# renodash v0.0.4 (commit: 0123456789abcdef...)

go run ./cmd/renodash version
# renodash dev (commit: 0123456789abcdef...)

# 未コミットの変更を含むローカルビルドの例:
# renodash dev (commit: 0123456789abcdef..., dirty)

mise ls "go:github.com/gkzz/renodash/cmd/renodash"
# go:github.com/gkzz/renodash/cmd/renodash  0.0.4-0.20261103085119-d4ec06cd2d6d  ...
# この版をインストールした場合: renodash v0.0.4-0.20261103085119-d4ec06cd2d6d (commit: unknown)
```

上記は表示形式の例であり、その版の公開を示すものではありません。標準入力は不要で、ネットワークアクセスやファイルの変更も行いません。

バージョンは、リリース用ビルドで `-ldflags -X` により埋め込んだ値、Go のビルド情報にあるメインモジュールのバージョン、`dev` の順で決めます。`go install github.com/gkzz/renodash/cmd/renodash@v0.0.4` では `v0.0.4` を表示します。mise の Go バックエンドで擬似バージョンを指定した場合も、そのバージョンをそのまま表示します。ローカルの `go run` は `dev` と表示します。

コミット SHA はリリースビルド時の埋め込み値、次に Go の `vcs.revision` を使い、どちらもなければ `unknown` と表示します。Go の擬似バージョンから SHA を推測しません。VCS 情報を含まないビルド（`-buildvcs=false` など）や、タグ指定の `go install` でも SHA を取得できない場合があります。情報が不明でも成功として終了します。

Go のビルド情報にある `vcs.modified` が `true` の場合だけ、コミット表示の末尾に `dirty` を追加します。リリース用の埋め込み値を使う場合や SHA が `unknown` の場合も同様です。`vcs.modified` がない場合は変更の有無を判断できないため、`dirty` は表示しません。

実行時に `git` は呼び出しません。表示する SHA はビルド元コミットを示し、`dirty` はビルド時に未コミットの変更があったことだけを示します。変更内容は表さないため、同じ版と SHA のビルドでも内容が同じとは限りません。

## パーサー

ルートパッケージ `github.com/gkzz/renodash` の `Parse` は `io.Reader` を受け取り、`Dashboard` を返します。

- `Items` に `approve`・`unschedule`・`unpend` の操作種別、ブランチ識別子、タイトル、セクション名、チェック状態を保持します。`Open` セクションの `rebase-branch` は操作種別 `rebase` として保持し、タイトルに PR リンクがあれば `pr_link` に元のリンク先を格納します。
- `BulkActions` に `approve-all-pending-prs`・`Open` セクションの `rebase-all-open-prs` とチェック状態を保持します。
- `UnknownControls` に、解析対象セクション内の未対応・不正なチェックボックス制御コメントの本文を前後の空白を除いて保持します。重複も残します。`unlimit`・`retry` などの未対応操作や未対応の一括操作もここに格納します。
- タイトルの Markdown はそのまま保持し、各スライス内の順序は入力順です。

解析対象は、[Renovate の生成処理](https://github.com/renovatebot/renovate/blob/main/lib/workers/repository/dependency-dashboard.ts)に基づく、次のレベル2見出しです。`## Pending Approval ##` のような末尾の閉じる `#` も認識します。レベル3以下のカテゴリ見出しでも対象セクションを保持します。

- `Pending Approval`、`Awaiting Schedule`、`Pending Status Checks`
- `Rate-Limited`、`Errored`、`Open`、`Other Branches`
- `Group Size Not Met`、`PR Creation Approval Required`、`Pending Branch Automerge`
- `PR Edited (Blocked)`、`PR Closed (Blocked)`
- `Config Migration Needed`、`Config Migration Needed (Blocked)`

`pr_link` は Dashboard に記載された相対パスまたは URL をそのまま保持します。`parse` は入力元のリポジトリを知らず、`get` も絶対 URL への変換や PR の現状確認は行いません。入力元の `OWNER/REPO` を知る利用側で必要に応じてリンクを解決してください。`Detected Dependencies` や説明用のセクション、未知のセクション名のチェックボックスは対象外で、`UnknownControls` にも収集しません。Renovate が新しいセクション名を追加した場合は、解析対象の追加が必要です。

行頭（0〜3スペースのインデントを含む）の `<!--` で始まる HTML コメントブロック、バッククォートまたはチルダによるコードフェンス内と、4スペース以上またはタブでインデントされたチェックボックスは解析しません。HTML コメントは最初の `-->` で終了し、閉じ忘れた場合は以降を解析しません。未知のコメントを既知の操作として扱うことはありません。入力の読み取りに失敗した場合は、空の `Dashboard` とエラーを返します。

## エージェント向けの調査情報

単体更新、またはパッケージ名が明示されたグループ更新のタイトルには、任意の `research` フィールドを追加します。PR が未作成でも、Dashboard 本文だけで調査材料を取得できます。

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
  "reference_urls": [],
  "provenance": {
    "package_source": "item_title",
    "target_version_source": "item_title",
    "detected_dependencies_source": "detected_dependencies",
    "association": "exact_package_name",
    "reference_url_source": "item_title"
  }
}
```

- `package` と `target_version` は `Update dependency NAME to VERSION`、`Update NAME to VERSION`、Docker の `Update NAME docker tag to VERSION` 形式から抽出します。任意のコミット接頭辞と、名前・版のバッククォート、Markdown リンクに対応します。版の `v` 接頭辞は保持します。
- `detected_dependencies` は `Detected Dependencies` の manager・ファイルの2階層から、パッケージ名が完全一致する記録を収集します。同じ現在版・manager・ファイルの組み合わせは重複を除き、異なる記録は入力順に保持します。現在版は digest を含む元の表記を保持します。
- 名前の別名解決やブランチ名からの推測はしません。例えばタイトルの `node.js` と検出名の `node` は結び付けません。同名記録は調査候補であり、そのブランチで更新されることを保証しません。
- `reference_urls` はタイトルにある HTTP(S) の Markdown リンクの URL です。上流リポジトリやリリース URL は推測しません。入力にリンクがなければ空配列です。
- 記録が見つからない場合も配列は `[]` です。パッケージ名の一覧がないグループ更新・lock file maintenance・未対応形式では `research` 自体を省略し、元のタイトルを保持します。

グループ更新は ``chore(deps): update toolchain (`go`, `node`, `pnpm`, `python`)`` のように、末尾の括弧内にバッククォート付きの名前をカンマ区切りで列挙した形式に対応します。単体更新の JSON は従来の形式を維持し、グループ更新では `research.packages` に各パッケージの情報を格納します。

```json
{
  "packages": [
    {
      "package": "node",
      "target_version": "unknown",
      "detected_dependencies": [
        {
          "current_version": "24.22.1",
          "manager": "mise",
          "file": "mise.toml",
          "available_updates": ["24.22.2", "26.13.0"]
        }
      ]
    }
  ],
  "reference_urls": [],
  "provenance": {
    "package_source": "item_title",
    "target_version_source": "unresolved",
    "detected_dependencies_source": "detected_dependencies",
    "available_updates_source": "detected_dependencies",
    "association": "exact_package_name",
    "reference_url_source": "item_title"
  }
}
```

`current_version` は Dashboard が示す現在版、`available_updates` は同じ記録に記載された更新候補です。候補には別の major 更新向けの版も含まれるため、候補が1件でもグループの更新先とは断定せず、`target_version` は `"unknown"` にします。候補の記載がなければ `available_updates` は `[]` です。

`provenance` は値の出所と関連付けの方法を示します。`item_title` は更新候補のタイトル、`detected_dependencies` は Dashboard の同名依存記録、`exact_package_name` は名前の完全一致を意味します。グループの `target_version_source: "unresolved"` は、更新先を示す根拠がないことを表します。`reference_urls` はタイトルに記載された URL であり、リンク先の正しさを検証したものではありません。`parse` は標準入力の取得元 URL や取得日時を知ることができないため、これらは出力しません。`get` は `source` に取得元と取得日時を記録します。

パッケージ名は完全一致で関連付けます。タイトル内の名前は重複を除いて記載順に保持し、一致する記録がなければ `detected_dependencies` は `[]` です。現在版・manager・file・更新候補がすべて同じ記録は重複を除き、それ以外は入力順で残します。グループ名やブランチ名からメンバーを推測しません。

この JSON をローカルのエージェントに渡し、パッケージの上流を特定したうえでリリースノート、コミット差分、GHSA、間接依存を調査できます。`get` は Issue 取得のためだけに `gh` 経由でネットワークにアクセスします。CLI はセキュリティ評価を行いません。調査結果では「更新にセキュリティ修正が含まれること」と「利用環境で悪用可能であること」を分けて扱ってください。

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

## Documentation

詳しい設計方針、開発計画、調査手順は [Documentation](docs/README.md) を参照してください。

- [Roadmap](docs/ROADMAP.md)
- [Agent Security Triage](docs/agent-security-triage.md)
- [Dashboard Agent Skill のサンプル](examples/skills/renodash-dashboard/SKILL.md)

Agent Skill はサンプルとして `examples/` に置いています。利用する際は、対応するエージェントのスキルディレクトリへコピーしてください。例えば `.agents/skills/renodash-dashboard/SKILL.md` に配置します。

## ライセンス

[MIT License](LICENSE)
