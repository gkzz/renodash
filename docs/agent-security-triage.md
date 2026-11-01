# エージェントによるセキュリティ調査

renodash の JSON をローカルのコーディングエージェントへ渡し、更新候補にセキュリティ修正が含まれるかと、更新の優先度を調べます。専用コマンドは不要です。調査だけを行い、更新の承認や適用は人が判断します。

## 更新候補を抽出する

Go は [README](../README.md#開発とテスト) の指定版、GitHub からの取得には認証済みの `gh`、絞り込みには `jq` を使います。リポジトリのルートで次を実行してください。

```sh
set -o pipefail
gh api repos/gkzz/dotfiles/issues/3 --jq '.body' > /tmp/renodash-dashboard.md
go run ./cmd/renodash parse < /tmp/renodash-dashboard.md \
  | jq '{items: [.items[] | select(.research.package == "pnpm")]}' \
  > /tmp/renodash-pnpm.json
cat /tmp/renodash-pnpm.json
```

インストール済みの `renodash parse` も使えますが、古いバイナリには `research` がなく、絞り込み結果が空になることがあります。空だった場合は、まず絞り込み前の JSON と Dashboard 本文を確認し、上記の `go run` で再実行してください。候補が消えた場合と CLI の版が古い場合を区別します。取得や解析が失敗した場合は、出力を調査に使わずエラーを解消してください。

2026-11-02 の PoC では次の出力を確認しました。Dashboard は変化するため、再実行時に同じ候補が残っているとは限りません。

```json
{
  "items": [
    {
      "action": "approve",
      "branch": "renovate/toolchain",
      "title": "chore(deps): update dependency pnpm to v12.14.0",
      "section": "Pending Approval",
      "checked": false,
      "research": {
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
    }
  ]
}
```

`reference_urls` が空でも調査できます。エージェントがパッケージの公式配布元と上流を確認します。`detected_dependencies` は同名パッケージの記録であり、対象ブランチで更新されることや、手元で実行中の版を保証しません。

## エージェントへ渡す

次のプロンプトに `/tmp/renodash-pnpm.json` の内容を添付します。再現できるよう、入力元 URL、取得日時、利用した CLI の版またはソースのコミット SHA も記録してください。ソースから実行した場合は `git rev-parse HEAD` で SHA を確認できます。

```text
以下は renodash が抽出した Renovate の更新候補です。

各候補について以下を調査してください。
1. パッケージの公式配布元と上流リポジトリを特定する。
2. 公式リリースノートを確認する。
3. 現在版と更新先のコミット差分、特に依存と lockfile の差分を確認する。
4. CVE・GHSA・RustSec などの公開アドバイザリを確認する。
5. 本体の修正、直接依存・間接依存の修正、別コンポーネントの変更を区別する。
6. セキュリティ修正が含まれることと、この環境で脆弱性が悪用可能であることを区別する。

各判断について証拠 URL を示してください。
確認できない項目は unknown とし、不足している証拠も示してください。
同じ脆弱性の CVE・GHSA・RustSec ID は別々に件数へ数えないでください。

結果には調査日時、対象の版、確認した差分のタグまたは SHA、
修正内容、影響を受ける条件、環境への該当状況を記載してください。
最後に更新の推奨度、理由、追加で確認すべきことをまとめてください。
推奨度と緊急度は分けてください。

GitHub Issue のチェックボックス操作や PR 作成、
リポジトリの変更は行わないでください。

入力元 URL:
取得日時:
CLI の版またはソースのコミット SHA:
調査対象環境（OS、導入方法、用途。未確認なら unknown）:

対象 JSON:
（ここに出力を貼り付ける）
```

## 調査結果の期待形式

候補ごとに次の形式で回答を求めます。`unknown` は「影響なし」を意味しません。

```markdown
調査日時:
入力元・取得日時・CLI の版または SHA:
対象パッケージ・現在版 → 更新先:
公式配布元・上流 URL:
比較したタグまたは SHA・差分 URL:
調査対象環境:

| 修正・アドバイザリ ID | 本体／直接依存／間接依存／別コンポーネント | 影響版・修正版 | 成立条件 | 環境への該当（yes/no/unknown）と根拠 | 証拠 URL |
| --- | --- | --- | --- | --- | --- |
| 確認した内容 | 分類 | 版 | 条件 | 判断と不足する証拠 | 一次情報の URL |

セキュリティ修正の包含: yes/no/unknown と根拠
実環境での悪用可能性: yes/no/unknown と根拠
更新の推奨度: 推奨／保留／不要 と理由
緊急度: 高／中／低／unknown と理由
追加確認: 判断に必要なファイル、ビルド設定、実行経路など
調査の限界: 未確認の項目と理由
```

## pnpm 12.13.0 → 12.14.0 で確認すること

この組み合わせは、2026-11-02 に調査した PoC の例です。[pnpm の公式リリースノート](https://github.com/pnpm/pnpm/releases/tag/v12.14.0)と[バージョン間の差分](https://github.com/pnpm/pnpm/compare/v12.13.0...v12.14.0)から、次を確認します。

- [rustls 更新コミット](https://github.com/pnpm/pnpm/commit/ad6310387968bc6e4e29770ef12ca5d8652a4bd9)の `Cargo.lock` に、間接依存の `0.23.45 → 0.23.46` がある。
- [rustls 0.23.46 の公式リリースノート](https://github.com/rustls/rustls/releases/tag/v/0.23.46)に、以下の4件のセキュリティ修正がある。
- [pnpr の修正](https://github.com/pnpm/pnpm/commit/583f02ac6d3731c45305779c8f936159013e290e)は別コンポーネントの変更として扱う。認証ファイルの権限や bcrypt の切り詰め対策を pnpm CLI の修正として報告しない。
- lockfile の更新だけでは、導入済み pnpm CLI での悪用可能性は確定しない。配布バイナリのビルド設定や実行経路を未確認なら `unknown` とする。

| rustls の修正 | 主な成立条件と証拠 |
| --- | --- |
| Brotli 証明書展開時の過大なメモリ確保 | Brotli の有効化が必要。[GHSA-h2v8-394m-p7vv](https://github.com/advisories/GHSA-h2v8-394m-p7vv)／[RUSTSEC-2026-0348](https://rustsec.org/advisories/RUSTSEC-2026-0348.html) |
| セッション値の Debug 出力による秘密情報漏えい | 独自セッションストアなどで対象の値をログ出力する場合。[GHSA-c8m5-c952-6cm4](https://github.com/advisories/GHSA-c8m5-c952-6cm4)／[RUSTSEC-2026-0346](https://rustsec.org/advisories/RUSTSEC-2026-0346.html) |
| 未認証の ServerHello による後続接続の鍵交換方式の誘導 | セッション保存とサーバーの鍵交換選択方針などに依存。[GHSA-w44r-93j7-qv28](https://github.com/advisories/GHSA-w44r-93j7-qv28)／[RUSTSEC-2026-0349](https://rustsec.org/advisories/RUSTSEC-2026-0349.html) |
| ECH 拒否時のバッファ済みアプリケーションデータ送信 | ECH 有効化、ハンドシェイク完了前の書き込み、ECH 拒否が条件。rustls 0.23.44 と 0.23.45 が影響対象。[GHSA-vp88-798v-h67f](https://github.com/advisories/GHSA-vp88-798v-h67f)／[RUSTSEC-2026-0347](https://rustsec.org/advisories/RUSTSEC-2026-0347.html) |

この例の期待判断は「間接依存にセキュリティ修正を含むため更新を推奨するが、実環境への影響と緊急度は unknown」です。次に、mise が導入するバイナリ、rustls の機能と TLS 設定、該当する呼び出し経路を確認します。

上記を証拠付きで報告し、別コンポーネントと実環境の影響を区別できれば、このケースの合格とします。期待結果を先に読んだ調査は、未知の候補に対する性能の検証にはなりません。独立した評価では、エージェントに抽出 JSON とプロンプトだけを渡し、調査後にこの節と照合してください。
