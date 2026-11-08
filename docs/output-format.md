# JSON と解析の仕様

`parse` と `get` は、更新候補を `items`、一括操作を `bulk_actions`、未対応の制御コメントを `unknown_controls` に出力します。`get` は取得元の情報を `source` に追加します。実行方法は[使い方](usage.md)を参照してください。

## 出力の基本ルール

| 項目 | 方針 |
| --- | --- |
| 入力 | `parse`: 標準入力の Markdown、`get`: `OWNER/REPO#ISSUE_NUMBER` |
| 出力 | 標準出力へ JSON（デフォルト形式） |
| エラー・警告 | 標準エラー出力 |
| 終了コード | 成功時は `0`、失敗時は非ゼロ |
| 未知の Renovate 制御コメント | JSON に保持し、標準エラー出力へ警告する（終了コードは `0`） |

JSON は `items`、`bulk_actions`、`unknown_controls` の配列を持ち、空の場合も `[]` を出力します。各項目のキーは `snake_case`、配列内は入力順で、出力末尾に改行が付きます。`parse` の `--format` は省略でき、対応する形式は `json` のみです。

## Issue の取得元と日時

`get` の `source` には、次の情報を記録します。

| フィールド | 内容 |
| --- | --- |
| `repository` | GitHub が返した Issue URL に基づく `OWNER/REPO` 表記。入力の大文字・小文字は引き継ぎません |
| `issue_number` | Issue 番号 |
| `issue_url` | GitHub の Issue URL |
| `fetched_at` | 取得時刻（UTC、RFC3339） |
| `issue_updated_at` | GitHub API が返す Issue の最終更新時刻 |

取得時刻と Issue の最終更新時刻は異なります。`items`、`bulk_actions`、`unknown_controls` は `parse` と同じ構造です。エラーと未知の制御コメントの警告は標準エラーへ出力し、失敗時は JSON を出力しません。`parse` は `source` を出力しません。

## 解析する操作とセクション

`items` と `bulk_actions` には、対応するチェックボックスの操作と状態を格納します。内部の型は `internal/dashboard` で定義しています。

- `items` に `approve`・`unschedule`・`unpend` の操作種別、ブランチ識別子、タイトル、セクション名、チェック状態を保持します。`Open` セクションの `rebase-branch` は操作種別 `rebase` として保持し、タイトルに PR リンクがあれば `pr_link` に元のリンク先を格納します。
- `bulk_actions` に `approve-all-pending-prs`・`Open` セクションの `rebase-all-open-prs` とチェック状態を保持します。
- `unknown_controls` に、解析対象セクション内の未対応・不正なチェックボックス制御コメントの本文を前後の空白を除いて保持します。重複も残します。`unlimit`・`retry` などの未対応操作や未対応の一括操作もここに格納します。
- タイトルの Markdown はそのまま保持し、各配列内の順序は入力順です。

解析対象は、[Renovate の生成処理](https://github.com/renovatebot/renovate/blob/main/lib/workers/repository/dependency-dashboard.ts)に基づく、次のレベル2見出しのセクションです。`## Pending Approval ##` のような末尾の閉じる `#` も認識します。セクション内にレベル3以下のカテゴリ見出しがあっても、解析を続けます。

- `Pending Approval`、`Awaiting Schedule`、`Pending Status Checks`
- `Rate-Limited`、`Errored`、`Open`、`Other Branches`
- `Group Size Not Met`、`PR Creation Approval Required`、`Pending Branch Automerge`
- `PR Edited (Blocked)`、`PR Closed (Blocked)`
- `Config Migration Needed`、`Config Migration Needed (Blocked)`

`pr_link` は Dashboard に記載された相対パスまたは URL をそのまま保持します。`parse` は入力元のリポジトリを知らず、`get` も絶対 URL への変換や PR の現状確認は行いません。入力元の `OWNER/REPO` を知る利用側で必要に応じてリンクを解決してください。`Detected Dependencies` や説明用のセクション、未知のセクション名のチェックボックスは対象外で、`unknown_controls` にも収集しません。Renovate が新しいセクション名を追加した場合は、解析対象の追加が必要です。

次のチェックボックスは解析しません。

- 行頭の `<!--` で始まる HTML コメントブロック内。行頭に0〜3個のスペースがある場合も含みます。
- バッククォートまたはチルダによるコードフェンス内。
- 4個以上のスペース、またはタブでインデントされた行。

HTML コメントは最初の `-->` で終了し、閉じ忘れた場合は以降を解析しません。未知のコメントを既知の操作として扱うことはありません。入力の読み取りに失敗した場合は、空の `Dashboard` とエラーを返します。

## エージェント向けの調査情報

対応する形式の単体更新や、パッケージ名が明示されたグループ更新には、`items` の各項目に `research` フィールドを追加します。PR が未作成でも、Dashboard 本文だけで調査材料を取得できます。

### 単体更新

次は、単体更新の `research` の例です。

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
- `detected_dependencies` は `Detected Dependencies` の manager・ファイルの2階層から、パッケージ名が完全一致する記録を収集します。現在版・manager・ファイルがすべて同じ記録は重複を除き、異なる記録は入力順に保持します。現在版は digest を含む元の表記を保持します。
- 名前の別名解決やブランチ名からの推測はしません。例えばタイトルの `node.js` と検出名の `node` は結び付けません。同名記録は調査候補であり、そのブランチで更新されることを保証しません。
- `reference_urls` はタイトルにある HTTP(S) の Markdown リンクの URL です。上流リポジトリやリリース URL は推測しません。入力にリンクがなければ空配列です。
- 記録が見つからない場合も配列は `[]` です。パッケージ名の一覧がないグループ更新・lock file maintenance・未対応形式では `research` 自体を省略し、元のタイトルを保持します。

### グループ更新

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

`current_version` は Dashboard が示す現在版、`available_updates` は同じ記録に記載された更新候補です。候補には別のメジャー更新向けの版も含まれるため、候補が1件でもグループの更新先とは断定せず、`target_version` は `"unknown"` にします。候補の記載がなければ `available_updates` は `[]` です。

### 値の出所と関連付け

`provenance` は値の出所と関連付けの方法を示します。`item_title` は更新候補のタイトル、`detected_dependencies` は Dashboard の同名依存記録、`exact_package_name` は名前の完全一致を意味します。グループの `target_version_source: "unresolved"` は、更新先を示す根拠がないことを表します。`reference_urls` はタイトルに記載された URL であり、リンク先の正しさを検証したものではありません。`parse` は標準入力の取得元 URL や取得日時を知ることができないため、これらは出力しません。`get` は `source` に取得元と取得日時を記録します。

パッケージ名は完全一致で関連付けます。タイトル内の名前は重複を除いて記載順に保持し、一致する記録がなければ `detected_dependencies` は `[]` です。現在版・manager・file・更新候補がすべて同じ記録は重複を除き、それ以外は入力順で残します。グループ名やブランチ名からメンバーを推測しません。

この JSON をローカルのエージェントに渡し、パッケージの上流を特定したうえでリリースノート、コミット差分、GHSA、間接依存を調査できます。`get` は Issue 取得のためだけに `gh` 経由でネットワークにアクセスします。CLI はセキュリティ評価を行いません。調査結果では「更新にセキュリティ修正が含まれること」と「利用環境で悪用可能であること」を分けて扱ってください。

抽出から調査までの実行例、エージェント向けプロンプト、結果の期待形式は [エージェントによるセキュリティ調査](agent-security-triage.md) を参照してください。

既存 CLI とローカルエージェントで更新の保留理由を調べた例は、[Minimum Release Age Triage PoC](minimum-release-age-triage/README.md) を参照してください。日時の出典を区別し、ログで確認できない原因は未確定として記録します。
