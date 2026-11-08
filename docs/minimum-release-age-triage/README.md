# 更新の保留理由と公開日時を調べる PoC

`gkzz/renovate-config#7` の GHCR 更新候補を既存の `renodash parse` で解析できた。設定には14日の待機期間と `internalChecksFilter: "strict"` がある。ただし、Renovate の実行ログを取得できていないため、保留原因は未確定である。

この PoC では、更新候補の保留理由と日時の出典を調べ、人間が更新可否を判断するための材料を整理した。Go の機能追加や対象リポジトリへの変更は行っていない。

## 実際に確認した内容

取得日時は `2026-11-01T23:17:08Z`（日本時間11月2日）。Dashboard の最終更新は `2026-11-01T20:40:24Z`。取得した本文を [dashboard.md](dashboard.md)、CLI の出力を [dashboard.json](dashboard.json) に保存した。解析に使用した renodash のコミットは `495c6617e4973758247307ef27d091658648f32e`。

| 確認事項 | 結果・根拠 |
| --- | --- |
| 対象 | [Dashboard #7](https://github.com/gkzz/renovate-config/issues/7) の `ghcr.io/renovatebot/renovate` → `v44.199.2` |
| 状態 | `Pending Status Checks`、ブランチ識別子 `renovate/github-actions` |
| 現在版・manager | `44.17.1`（digest 付き）、`github-actions`。datasource と manager は別の情報 |
| 設定 | [renovate.json5](https://github.com/gkzz/renovate-config/blob/77fd5c225f704c84d37402025fa7f670d24b7055/renovate.json5) に `minimumReleaseAge: "14 days"`、`internalChecksFilter: "strict"`。`minimumReleaseAgeBehaviour` の明示指定なし |
| ほかの制約 | 同ファイルに月曜の schedule、Dashboard 事前承認、GitHub Actions のグループ化。取得した設定だけでは実行時の最終設定まで確定しない |
| 更新対象 | [validate.yml](https://github.com/gkzz/renovate-config/blob/77fd5c225f704c84d37402025fa7f670d24b7055/.github/workflows/validate.yml) の `docker://ghcr.io/renovatebot/renovate:44.17.1@sha256:…`。datasource は `docker` と推定し、ログでの確認を残す |
| 上流公開日時 | [GitHub Release 44.199.2](https://github.com/renovatebot/renovate/releases/tag/44.199.2) の REST API `published_at` は `2026-11-01T19:12:13Z`。取得時点では14日未満 |
| Renovate の日時 | 未観測。ログがないため、欠落が実証されたわけではない |

GitHub Release の公開は日本時間11月2日04:12、Dashboard 更新は05:40、取得は08:17だった。公開から Dashboard 更新までは約1時間28分で、仮に正しい公開日時を取得できていても14日の待機期間には達していない。そのため、今回の保留は通常の待機期間内でも日時欠落でも説明でき、現在の更新候補だけでは両者を切り分けられない。GHCR 成果物の公開日時も未確認のため、評価は `undetermined` とする。

対象 workflow の `renovate-config-validator` は設定検証を行う。そこに固定された `44.17.1` は、Dashboard を生成した Renovate の稼働バージョンを示さない。Actions の workflow 一覧からも依存更新ジョブは確認できなかった。Dashboard が案内する [Mend portal](https://developer.mend.io/github/gkzz/renovate-config) の実行ログは今回取得できていない。

## 調査時点の公式仕様との照合

- [Docker datasource](https://docs.renovatebot.com/modules/datasource/docker/) は日時対応を Docker Hub の `tag_last_pushed` に限定している。GHCR の日時欠落を疑う根拠になるが、対象実行の観測とは分ける。
- [minimumReleaseAgeBehaviour](https://docs.renovatebot.com/configuration-options/#minimumreleaseagebehaviour) の既定値は `timestamp-required`。日時がない版は安定扱いにならない。`timestamp-optional` は日時なしの版を安定扱いにするため、待機期間を保証しない。
- [Minimum Release Age](https://docs.renovatebot.com/key-concepts/minimum-release-age/) は Renovate 42 の既定動作変更、`strict` での保留、発行者が変更できる OCI 作成日時の問題を説明している。同資料では `github-tags` の `committedDate` とリリース公開日時を区別している。lockfile maintenance も、package manager 側の適用状況を別途確認する必要がある。

今回の GitHub Release の公開日時は GHCR のタグ・digest の公開日時を証明しない。日時を代用して「14日経過済み」「欠落だけが原因」と結論づけない。

## 再現手順

前提は Bash、Go（[開発ガイド](../development.md#開発とテスト)の指定版）、`gh`、`jq` と対象情報への読み取り権限。リポジトリのルートで Bash を使って実行する。取得結果は変化するため、今回の記録と一致するとは限らない。取得失敗、本文の欠落・空文字、CLI の失敗、対象候補の未検出時は処理を停止する。

```bash
set -euo pipefail

triage_dir=$(mktemp -d)
gh api repos/gkzz/renovate-config/issues/7 > "$triage_dir/issue.json"
jq -er '.body | select(type == "string" and length > 0)' \
  "$triage_dir/issue.json" > "$triage_dir/dashboard.md"
go run . parse < "$triage_dir/dashboard.md" > "$triage_dir/dashboard.json"
target_version=$(
  jq -er '
    [
      .items[]
      | select(.research.package == "ghcr.io/renovatebot/renovate")
      | .research.target_version
    ]
    | if length == 1
      then .[0] | select(type == "string" and length > 0)
      else error("expected exactly one candidate")
      end
  ' "$triage_dir/dashboard.json"
)
gh api repos/gkzz/renovate-config/commits/main --jq .sha
# 得られた SHA を ref に指定し、default.json、renovate.json5、更新対象ファイルを取得する。
release_tag="${target_version#v}"
gh api "repos/renovatebot/renovate/releases/tags/$release_tag" \
  --jq '{html_url,tag_name,published_at,created_at}'
```

リリース日時は Dashboard から取得した更新先の版で照会する。`renovatebot/renovate` の GitHub Release タグは `v` なしのため、先頭の `v` を除去している。ほかのパッケージではタグの命名規則を確認する。

保存済み本文だけを解析する場合は、次を実行する。`dashboard.json` は調査当時の CLI 出力であり、現在の出力には `research.provenance` が加わるため、完全には一致しない。更新候補と、パッケージ名・版・依存関係の情報が保存済みの記録と一致することを確認する。

```sh
go run . parse < docs/minimum-release-age-triage/dashboard.md
```

## ローカルエージェントに渡す調査指示

`dashboard.json` と対象設定を渡し、次の順に調べる。実行ログを使う場合は、非公開のローカル作業領域で扱い、エージェントへの入力も必要な範囲に絞る。

1. 設定の extends と packageRules、manager、update type、グループ構成を追い、実行時の待機期間・behaviour・filter を確認する。ファイル設定と最終設定を区別する。
2. Dashboard 更新に対応する実行日時・run ID・Renovate バージョンを特定する。Mend portal で対象リポジトリのログを取得できなければ、未取得と記録する。
3. 対象パッケージ・版・ブランチの前後で `releaseTimestamp`、`minimumReleaseAge`、`internalChecksFilter`、`isPending`、ブランチ作成を止めた判断を探す。ログに値が出ないことだけで日時欠落と断定しない。
4. GHCR のタグと digest に対応する公式メタデータを確認する。GitHub Release、Git コミット、イメージ内の作成日時はそれぞれ別の観測として保存する。
5. 同じ成果物の日時だと確認できた場合に限って差分を計算する。通常の待機期間内、日時欠落、schedule、承認、グループ内の別依存などを切り分ける。
6. 公開情報の根拠 URL、取得日時、確定事項、仮説、追加調査を出力する。非公開ログで確認した事実には、その旨と確認範囲を記す。公開情報だけでは再検証できない点も明示する。日時不足やログ未取得なら更新可否は `undetermined` とする。

Mend portal の実行ログは、原文・抜粋ともパブリックリポジトリや Issue、PR に掲載しない。トークンを除去したログも公開対象にはしない。ログファイル、実行識別子、非公開ログへの直接リンクは非公開の作業領域にとどめる。公開するのは、内容を確認したうえでの必要最小限の事実の要約と確認状況に限る。

調査のために Dashboard のチェックボックスを操作したり、設定や PR を変更したりしない。

## 日時を保存する際の区別

各観測に `kind`、`value`、`source_url`、`observed_at`、対象の版・digest、信頼できる範囲を付ける。

| kind | 意味・判断の限界 |
| --- | --- |
| `renovate_release_timestamp` | 対象実行で実際に使った値。ログ未取得と、明示的な欠落を区別する |
| `registry_published_at` | レジストリの日時。タグか digest か、再 push を反映するか確認する |
| `git_committed_at` | コミット日時。タグの作成・push・公開日時を保証しない |
| `github_release_published_at` | GitHub Release の公開日時。コンテナ公開日時とは別 |
| `image_created_at` | 発行者が設定する値。待機期間の証明に使わない |
| `agent_estimated_at` | 推定値。根拠と不確実性を残し、自動判定に使わない |

[assessment.json](assessment.json) は今回の観測結果であり、CLI の出力仕様ではない。`renovate_release_timestamp: null` は、`timestamp_observation: "not_observed"` と併せて未観測を表す。日時欠落をログで確認した場合は別の状態として記録する。

## 検証項目と今回の到達点

今回の調査では、日時の出典と設定を確認し、保留原因を確定するには実行ログが必要だと分かった。

- [x] 設定ファイルから14日ルールを取得する。
- [x] Pending Status Checks の原因を調査し、確認できた範囲と残る仮説を示す。
- [x] 日時の取得元と信頼できる範囲を区別する。
- [x] 日時不足時に更新可否を決めない。
- [x] 根拠 URL と追加調査項目を出力する。
- [ ] 対象実行のログで、日時欠落が保留原因か確定する。

次は非公開の作業領域で対象実行のログを確認し、日時とフィルター判断を突き合わせる。公開文書には必要な事実の要約と確認状況だけを反映する。繰り返し必要になる情報が分かるまでは既存 CLI を使い、Go の実装が必要になった段階で別の `feat/` ブランチに分ける。

可能であれば、14日以上前に公開された GHCR 成果物の更新候補が保留されているケースでも調査する。比較する日時が対象のタグ・digest に対応する成果物の公開日時であることを確認し、通常の待機期間と日時欠落を切り分ける。
