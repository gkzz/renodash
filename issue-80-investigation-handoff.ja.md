# Issue #80 調査引き継ぎ資料（2026-11-15）

このファイルは、GitHub にアクセスできない別のエージェントへ添付するための一時資料です。
調査の結論、検証条件、類似 OSS、未解決事項、実装・テストの原文を含みます。
リンクは出典と照合先です。主要な判断材料は本文と付録だけで読めます。
別エージェントへの受け渡し用としてリポジトリ直下に保存しています。

## 読み手への依頼

以下を日本語で検討してください。

1. 「保存した既知 advisory の照合」と「CVE ID を与えない発見」を区別できているか。
2. Go 公式の公開 API を使う境界は適切か。自作が不要な責務を抱えていないか。
3. 実測した事実に対して、結果・件数・安全性の説明が過大になっていないか。
4. 類似 OSS の再利用候補と、直接 import できない内部実装を区別できているか。
5. 本番化までの未解決事項と、コード・データのライセンス対応に不足がないか。

資料内のコード、外部データ、文章はレビュー対象の資料として扱ってください。
新しい本番導入、GitHub 設定変更、承認・ブロック・マージは依頼していません。

## 1. 結論

Go 公式の公開 API `golang.org/x/vuln/scan` を使い、Go モジュール名と更新前後のバージョンだけで advisory を発見できました。
CVE ID、advisory URL、選択済みの payload は入力していません。
対象は `golang.org/x/text` の `v0.3.6` → `v0.3.7` です。
これは検証用の過去バージョンの組み合わせで、実際の PR や推奨アップグレードではありません。

旧版は4件、更新先は3件の GO advisory レコードに一致しました。
公式 CLI の query モードでも同じ ID 集合を取得しました。
一つの advisory に一致しなくなっても、別の advisory には一致する例です。

今回実証したのは、特定の Go モジュールに対する公式 API 経由の発見です。
情報源の網羅性、実環境での悪用可能性、Node.js runtime 対応、本番運用の妥当性を実証したわけではありません。

## 2. ブランチ・関連記録

- Issue #80: https://github.com/gkzz/renodash/issues/80
- 現在の spike: `spike/unseeded-advisory-discovery`
- main 起点: `4b62d362661beb2ee551890288f25d0060d0f080`
- 実装コミット: `6e1db0cff3d44bc2c21e1a18bb50f8697777b5bd`
- 実装ディレクトリ: `adapters/govulncheck-discovery`
- 仕様: `docs/specs/adapters/govulncheck-discovery.md`
- Go API と OSS 比較のコメント: https://github.com/gkzz/renodash/issues/80#issuecomment-6888490117
- 以前の Summary 修正・検証記録: https://github.com/gkzz/renodash/issues/80#issuecomment-6888023923

新しい spike は main から独立に作り、以前の PR #82 の実装は取り込んでいません。
新しいブランチは push 済みです。本番導入・マージはしていません。

## 3. 以前の Node.js spike との違い

| 項目 | 以前の PR #82 | 今回の Go spike |
| --- | --- | --- |
| 目的 | 既知 advisory の照合・Summary の出力と失敗処理 | advisory ID を教えない発見と公式 API 再利用 |
| 入力 | 手動で取得・変換した既知 CVE、保存済み index | module 名・ecosystem・配布形態・更新前後バージョン |
| 取得・照合 | 保存済み payload と spike の判定処理 | Go 公式 API に委任 |
| 対象 | Node.js 公式 runtime、Linux の保存 fixture | Go upstream module の検証用バージョン対 |
| 実際の PR 評価 | 現在の PR を評価していない | PR diff の抽出・同一性検証をしていない |
| GitHub Summary | ローカル再現 Markdown をレビュー | 新しい Summary の本番統合はしていない |

以前の spike は、既知の `CVE-2026-123863` を手動で取得・変換して、保存した PR #53 の Node.js `24.22.2` / `26.13.0` と照合しました。
その段階では、CVE を知らない状態からの発見は実証していませんでした。
`conflicting` は独立した二つの情報源ではなく、合成した範囲を使ったケースです。

### 以前の表示上の修正

PR #82 のコミット `2654fc92c0bb4c900b1d23dd5ae6af3caf4b6870` で次を修正しました。

- 対象バージョン変更後も残る coverage の結論を除去。
- OpenJS CNA、手動変換、解析アダプターを区別。
- 手動変換レコードへ不適切な NSWG 識別子を付けない。
- 欠損・不正・不一致の実際の原因を表示。
- 意図的な CI 失敗とジョブの成功を区別。
- 長い checked ranges を、全文を保持して折りたたみ表示。

修正後の実行: https://github.com/gkzz/renodash/actions/runs/43668103622
通常の review と9通りの fixture ジョブ、計10ジョブが成功しました。
ただし、GitHub に保存された Summary 本文の直接取得はできていません。
レビューしたものは実行コミット・成果物からの再現出力であり、GitHub の実表示との完全一致やレイアウトは未確認です。

## 4. 今回採用した公式 API

- 公開 API: `golang.org/x/vuln/scan`
- 固定した module version: `v1.9.0`
- 対応する upstream commit: `e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5`
- データベース: `https://vuln.go.dev`
- 呼び方: `scan.Command(ctx, "-mode=query", "-json", "-db", database, "module@version")` → `Start()` → `Wait()`

これは govulncheck の公開 Go API であり、汎用の OSV HTTP client SDK とは別です。
取得と影響バージョンの照合を公式実装へ委任し、advisory 用 HTTP client・semver 比較・範囲判定は自作していません。
CLI 風の引数を取りますが、adapter は別プロセスの govulncheck や対象プロジェクトのコードを起動しません。
upstream の `internal` package は import・コピーしていません。

adapter が担当するのは、対象の選別、更新前後の照会、SDK JSON の変換、失敗の保持、証拠の保存です。
対象 identity は呼び出し側の申告であり、実際の PR から検証したものではありません。
標準ライブラリや Go runtime は今回の選別対象に含めていません。

## 5. 検証入力と実測結果

入力全文:

```json
{
  "ecosystem": "Go",
  "distribution": "upstream-module",
  "module": "golang.org/x/text",
  "old_version": "v0.3.6",
  "proposed_version": "v0.3.7"
}
```

評価完了日時: `2026-11-15T03:08:54Z`（UTC）。Go API とデータベースの時刻を混同しないでください。

| 対象 | 判定 | 一致した GO advisory ID |
| --- | --- | --- |
| 更新前 `v0.3.6` | Affected | GO-2021-0113, GO-2022-1059, GO-2026-5970, GO-2026-6629 |
| 更新先 `v0.3.7` | Affected | GO-2022-1059, GO-2026-5970, GO-2026-6629 |

`GO-2021-0113` は更新先の一致から外れ、他の3件は残りました。
件数は advisory レコード数です。異なるデータベースの ID を alias で束ねた件数や、実際に悪用可能な脆弱性数ではありません。
取得時の両 assessment の `limitations` 配列は空でした。
これは既知の制約がないという意味ではなく、今回の adapter が照会・出力変換エラーを記録しなかったという意味です。

各バージョンについて、SDK が報告した設定と保存した元ストリームの SHA-256 は次のとおりです。
これらの digest は元 SDK 出力に対するもので、この資料の抜粋 JSON への digest ではありません。

### old: `golang.org/x/text@v0.3.6`

```json
{
  "retrieved_at": "2026-11-15T03:08:53Z",
  "evidence_file": "old.sdk.json",
  "sha256": "cb6be6375d79e75c5917a2cea9d86e6f806da0f904c25ef39fe023aa821f153c",
  "sdk_config": {
    "protocol_version": "v1.0.0",
    "scanner_name": "govulncheck-discovery",
    "scanner_version": "v0.0.0",
    "db": "https://vuln.go.dev",
    "db_last_modified": "2026-11-11T02:51:11Z",
    "scan_level": "symbol",
    "scan_mode": "query"
  }
}
```

### proposed: `golang.org/x/text@v0.3.7`

```json
{
  "retrieved_at": "2026-11-15T03:08:54Z",
  "evidence_file": "proposed.sdk.json",
  "sha256": "9434f914b75c3d90dafd2b6154aae55864d81893aea7727d424c53f140a636da",
  "sdk_config": {
    "protocol_version": "v1.0.0",
    "scanner_name": "govulncheck-discovery",
    "scanner_version": "v0.0.0",
    "db": "https://vuln.go.dev",
    "db_last_modified": "2026-11-11T02:51:11Z",
    "scan_level": "symbol",
    "scan_mode": "query"
  }
}
```

### advisory の照合情報（保存レコードからの抜粋）

以下は判定の根拠を読めるよう、SDK が返したレコードのフィールドを選んだものです。
独立した追加照会や、adapter 独自の範囲再評価はしていません。
説明文は省き、ID・alias・時刻・package・range・reference を残しています。

```json
[
  {
    "id": "GO-2021-0113",
    "aliases": [
      "CVE-2021-38561",
      "GHSA-ppp9-7jff-5vj2"
    ],
    "published": "2021-10-06T17:51:21Z",
    "modified": "2024-05-20T16:03:47Z",
    "affected": [
      {
        "package": {
          "name": "golang.org/x/text",
          "ecosystem": "Go"
        },
        "ranges": [
          {
            "type": "SEMVER",
            "events": [
              {
                "introduced": "0"
              },
              {
                "fixed": "0.3.7"
              }
            ]
          }
        ],
        "ecosystem_specific": {
          "imports": [
            {
              "path": "golang.org/x/text/language",
              "symbols": [
                "MatchStrings",
                "MustParse",
                "Parse",
                "ParseAcceptLanguage"
              ]
            }
          ]
        }
      }
    ],
    "references": [
      {
        "type": "FIX",
        "url": "https://go.dev/cl/340830"
      },
      {
        "type": "FIX",
        "url": "https://go.googlesource.com/text/+/383b2e75a7a4198c42f8f87833eefb772868a56f"
      }
    ]
  },
  {
    "id": "GO-2022-1059",
    "aliases": [
      "CVE-2022-32149",
      "GHSA-69ch-w2m2-3vjp"
    ],
    "published": "2022-10-11T18:16:24Z",
    "modified": "2024-05-20T16:03:47Z",
    "affected": [
      {
        "package": {
          "name": "golang.org/x/text",
          "ecosystem": "Go"
        },
        "ranges": [
          {
            "type": "SEMVER",
            "events": [
              {
                "introduced": "0"
              },
              {
                "fixed": "0.3.8"
              }
            ]
          }
        ],
        "ecosystem_specific": {
          "imports": [
            {
              "path": "golang.org/x/text/language",
              "symbols": [
                "MatchStrings",
                "ParseAcceptLanguage"
              ]
            }
          ]
        }
      }
    ],
    "references": [
      {
        "type": "REPORT",
        "url": "https://go.dev/issue/56152"
      },
      {
        "type": "FIX",
        "url": "https://go.dev/cl/442235"
      },
      {
        "type": "WEB",
        "url": "https://groups.google.com/g/golang-announce/c/-hjNw559_tE/m/KlGTfid5CAAJ"
      }
    ]
  },
  {
    "id": "GO-2026-5970",
    "aliases": [
      "CVE-2026-56852"
    ],
    "published": "2026-07-14T17:29:56Z",
    "modified": "2026-08-10T22:42:36Z",
    "affected": [
      {
        "package": {
          "name": "golang.org/x/text",
          "ecosystem": "Go"
        },
        "ranges": [
          {
            "type": "SEMVER",
            "events": [
              {
                "introduced": "0"
              },
              {
                "fixed": "0.39.0"
              }
            ]
          }
        ],
        "ecosystem_specific": {
          "imports": [
            {
              "path": "golang.org/x/text/unicode/norm",
              "symbols": [
                "Form.Append",
                "Form.AppendString",
                "Form.Bytes",
                "Form.FirstBoundary",
                "Form.FirstBoundaryInString",
                "Form.IsNormal",
                "Form.IsNormalString",
                "Form.LastBoundary",
                "Form.NextBoundary",
                "Form.NextBoundaryInString",
                "Form.Properties",
                "Form.PropertiesString",
                "Form.QuickSpan",
                "Form.QuickSpanString",
                "Form.Span",
                "Form.SpanString",
                "Form.String",
                "Form.Transform",
                "Iter.Init",
                "Iter.InitString",
                "Iter.Next",
                "Iter.Seek",
                "nextComposed",
                "normReader.Read",
                "normWriter.Write"
              ]
            }
          ]
        }
      }
    ],
    "references": [
      {
        "type": "REPORT",
        "url": "https://go.dev/issue/80142"
      },
      {
        "type": "FIX",
        "url": "https://go.dev/cl/794100"
      }
    ]
  },
  {
    "id": "GO-2026-6629",
    "aliases": [
      "CVE-2026-56851"
    ],
    "published": "2026-10-07T14:09:43Z",
    "modified": "2026-10-07T14:09:43Z",
    "affected": [
      {
        "package": {
          "name": "golang.org/x/text",
          "ecosystem": "Go"
        },
        "ranges": [
          {
            "type": "SEMVER",
            "events": [
              {
                "introduced": "0"
              },
              {
                "fixed": "0.41.0"
              }
            ]
          }
        ],
        "ecosystem_specific": {
          "imports": [
            {
              "path": "golang.org/x/text/secure/precis",
              "symbols": [
                "Profile.Append",
                "Profile.AppendCompareKey",
                "Profile.Bytes",
                "Profile.Compare",
                "Profile.CompareKey",
                "Profile.NewTransformer",
                "Profile.String",
                "Transformer.Bytes",
                "Transformer.String",
                "Transformer.Transform",
                "nickAdditionalMapping.Transform"
              ]
            }
          ]
        }
      }
    ],
    "references": [
      {
        "type": "FIX",
        "url": "https://go.dev/cl/793360"
      },
      {
        "type": "REPORT",
        "url": "https://go.dev/issue/80112"
      }
    ]
  }
]
```

## 6. 動作確認の結果と範囲

以下は実装コミットに対して実行済みの結果です。今回の資料作成時に再実行した、という意味ではありません。

| 確認 | 結果・範囲 |
| --- | --- |
| 公式 CLI query（固定 v1.9.0） | 両バージョンで実行。API 経由の結果と同じ ID 集合 |
| 公開 Go API を直接呼ぶ adapter | 両バージョンをライブ照会し、出力と report を保存 |
| isolated module の `go test ./... -count=1` | 成功。保存された出力の ID 集合と digest を照合 |
| isolated module の `go vet ./...` | 成功 |
| ルートの `make check` | 成功 |
| `git diff --check` | 成功 |
| 新しい isolated module の GitHub Actions | 未実施。root の Go テストは nested module を走査しない |

テストでは SDK 出力を注入し、不正な protocol・設定・identity、失敗、部分出力、出力制限、重複・相違レコード、withdrawn、未対応入力を確認しました。
実際の公開 API に対しては、キャンセル済み context のテストもあります。
SDK の範囲判定そのものを、自作アルゴリズムと比較して正当化したわけではありません。
テストは外部 API にアクセスしませんが、初回の Go 依存取得にはネットワークが必要になる場合があります。

## 7. 再現手順

Go 1.27 以降を使います。リポジトリの root から次を実行します。

```sh
cd adapters/govulncheck-discovery
go test ./... -count=1
go vet ./...
```

ライブ照会には `vuln.go.dev` への接続が必要です。
root で親ディレクトリを用意し、未使用の出力先を選んでください。

```sh
mkdir -p local/advisory-discovery
cd adapters/govulncheck-discovery
go run . -input testdata/target.json \
  -evidence-dir ../../local/advisory-discovery/handoff-run
```

現在のデータベースを再照会すると、後日の情報追加・修正によって結果が変わり得ます。
当時の再現には、`TestSavedOfficialSDKOutput` と保存済み SDK stream を使います。
元 stream は `testdata/live-20261115/old.sdk.json` と `proposed.sdk.json` です。
この資料には、それらの全説明文を含めず、照合用のフィールドを抜粋しています。

## 8. 未解決事項と判断の限界

- PR diff からの正しい製品・配布形態・バージョンの抽出と検証。
- 評価対象 PR head、CI で試験した SHA、実行コード SHA の紐づけ。
- 新 advisory、レコード修正、PR 更新に対する再評価。
- 信頼できる実行環境、権限、PR 側で変更可能な assessor の扱い。
- 情報源の収録遅延・網羅性、複数情報源の alias・相違レコードの扱い。
- 実際の部署・利用コード・環境における到達性や影響。
- Job Summary の本番統合と表示方法。
- 標準ライブラリ、Go runtime、Node.js runtime、他 ecosystem への対応。
- 保存データと配布物における第三者ライセンスの表示。

`NoMatchInCheckedSources` は安全の保証ではありません。
`Affected` は今回の query モードが返した module/version の一致で、実環境での exploitability を確認したものではありません。
どの結果も、自動承認・自動ブロック・自動マージの根拠として採用していません。

## 9. OSS 利用とライセンス（2026-11-15 に一次資料を確認）

コードとデータは別のライセンスです。

| 対象 | ライセンス | OSS での利用・配布時の主な対応 |
| --- | --- | --- |
| 採用した `golang.org/x/vuln` の Go コード | BSD 3-Clause 相当の LICENSE | source 配布では著作権・条件・免責を保持。binary 配布では文書等へ収録。名称を無断の推薦・承認表示に使わない |
| `vuln.go.dev` の database entries | CC BY 4.0 | 出典・指定されている帰属情報・ライセンス・免責表示等を保持し、加工を明示。許諾された再利用を妨げる追加制限を付けない |
| renodash 独自コード | root の MIT License | 第三者のコード・データを一括で MIT に変更した扱いにはしない |

両者とも、条件を守れば OSS で利用できます。CC BY 4.0 は商用利用や加工・再配布も許可しています。
CC BY 4.0 には ShareAlike 条件がなく、データの利用だけで renodash の独自コード全体を CC BY に変更する必要はありません。
SDK を組み込む binary 配布では、SDK 本体に加えて配布される推移的依存の通知も確認する必要があります。
これは確認したライセンスに基づく整理であり、全利用形態についての法的適合性を保証するものではありません。

**表示を補足済み:** `adapters/govulncheck-discovery/testdata/README.md` に、保存データの帰属・CC BY 4.0・加工内容・免責を追記しました。
`THIRD_PARTY_NOTICES.md` に SDK の元 LICENSE と配布時の注意を記載しています。
SDK の採用方法と保存データの判定結果は変更していません。
バイナリを配布する場合は、推移的依存も含めた通知の確認が別途必要です。

### この資料に含めたデータの帰属・変更表示

出典: The Go Authors / Go Vulnerability Database (`https://vuln.go.dev/`)。
Database entries は [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/) に基づき使用しています。
各 record の原資料への reference と識別子は上の抜粋に保持しています。
この資料では SDK 出力からフィールドを抜粋・再配置し、結果を日本語で要約しました。
元データや第三者が、この調査や renodash を推薦・承認しているとは主張していません。
元データはライセンスの免責条件に従い提供され、正確性等の保証はありません。

確認した一次資料:

- https://raw.githubusercontent.com/golang/vuln/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5/LICENSE
- https://github.com/golang/vulndb/blob/master/README.md#license
- https://creativecommons.org/licenses/by/4.0/
- https://creativecommons.org/licenses/by/4.0/legalcode.en （特に Section 3）

## 10. 類似 OSS の調査

以下は今回の調査記録です。実行済みのものと文書・実装を閲覧しただけのものを区別しています。

## 類似 OSS と公式 Go API の比較（2026-11-15）

今回の Go module/version 照会には、Go 公式 `golang.org/x/vuln/scan` の query モードを採用した。
取得・範囲判定を自作せず、更新前後の照会と証拠保存だけを adapter が担当する。
これは Go モジュールに限定した採用判断であり、Node.js runtime 対応を意味しない。

### 比較

| 候補 | 確認した機能 | 今回との関係 | 検証の範囲 |
| --- | --- | --- | --- |
| Go `govulncheck` / 公開 `scan` API | `module@version` の query、Go Vulnerability Database、JSON 出力 | 公開 Go API を直接 adapter として採用 | 固定 `v1.9.0` の API と CLI を両バージョンで実行 |
| OSV の公式 Go bindings | `osvdev` client と、experimental package の pagination helper | OSV を直接使う場合の候補。Go 公式 API と別の選択肢 | リポジトリと利用側コードの確認。組み込み実行は未実施 |
| OSV-Scanner | 旧新の結果差分、alias によるグループ化 | 比較・重複表示の参考。差分だけでは旧新両方の全一致は表示できない | 実装閲覧。scanner 本体の実行は未実施 |
| OSV-SCALIBR | pluggable な抽出・enrichment、注入可能な OSV client、取得タイムアウト | source adapter / capability 設計の参考。今回の対象には全体導入は不要と判断 | 実装・README 閲覧。ライブラリ組み込みは未実施 |
| Dependency Review Action | base/head の dependency graph compare、Summary、warn-only、snapshot 警告 | PR と SHA に紐づいた報告の比較候補。任意の module/version を直接引く API の代替ではない | 実装・設定閲覧。実リポジトリでの適用は未実施 |

### 固定して確認した一次資料

#### Go 公式

- [公開 API](https://pkg.go.dev/golang.org/x/vuln@v1.9.0/scan)
- [scan.Command / Start / Wait の実装](https://github.com/golang/vuln/blob/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5/scan/scan.go)
- [query モード](https://github.com/golang/vuln/blob/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5/internal/scan/query.go)
- [query モードの制約](https://github.com/golang/vuln/blob/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5/internal/scan/flags.go)
- [Go Vulnerability Database](https://go.dev/doc/security/vuln/database)

公開 `scan` package だけを import する。`internal/client` や `internal/semver` は外部からの再利用 API ではない。
query モードは module/version に対する照会であり、source モードの到達性解析を行ったとは扱わない。
CLI 風の引数を受け取るが、公開 API は内部で Go 実装を呼び、adapter は subprocess を起動しない。

#### OSV 公式 Go bindings

- [bindings](https://github.com/google/osv.dev/tree/e15de6dfb89d532a870731e9e78ca6b2db19a4ed/bindings/go)
- [experimental pagination](https://github.com/google/osv.dev/blob/e15de6dfb89d532a870731e9e78ca6b2db19a4ed/bindings/go/osvdevexperimental/paging.go)

`osvdevexperimental` は名称と説明どおり experimental な helper であり、安定した API と断定しない。
OSV-SCALIBR が `osvdev` と組み合わせて利用していることを確認した。
依存を採用する場合は、対象 API の安定性・バージョン・タイムアウト・partial result の取り扱いを検証する必要がある。

#### OSV-Scanner

- [旧新差分](https://github.com/google/osv-scanner/blob/80fe50567a5d6e0ec408353e2ceacca1561b91be/internal/ci/vulnerability_result_diff.go)
- [alias grouping](https://github.com/google/osv-scanner/blob/80fe50567a5d6e0ec408353e2ceacca1561b91be/internal/grouper/build.go)

旧新差分の実装は主に新たに現れた結果を返す。今回必要な「旧版と更新先のそれぞれの状態」とは用途が異なる。
これらは `internal` package なので、外部から直接 import する設計にはしない。

#### OSV-SCALIBR

- [README / plugins](https://github.com/google/osv-scalibr/blob/3fcb96d9989b68907927f55036754cc34896c315/README.md)
- [client interface](https://github.com/google/osv-scalibr/blob/3fcb96d9989b68907927f55036754cc34896c315/enricher/vulnmatch/osvdev/client.go)
- [OSV enrichment](https://github.com/google/osv-scalibr/blob/3fcb96d9989b68907927f55036754cc34896c315/enricher/vulnmatch/osvdev/osvdev.go)

capability による plugin 選択と、注入可能な client が責務分離の参考になる。
「package identity の抽出」と「公開 advisory の照会」は別の工程として扱える。

#### Dependency Review Action

- [設定と warn-only](https://github.com/actions/dependency-review-action/blob/907a6e9569212033966bc0b69d32473b046dd66b/README.md)
- [dependency graph compare](https://github.com/actions/dependency-review-action/blob/907a6e9569212033966bc0b69d32473b046dd66b/src/dependency-graph.ts)
- [Summary と artifact 出力](https://github.com/actions/dependency-review-action/blob/907a6e9569212033966bc0b69d32473b046dd66b/src/main.ts)

GitHub に依存情報が収録されていることが前提になる。
warn-only を設定できても、採用済み・実環境で期待どおり動くという結論にはしていない。

### ライブ検証

対象は `golang.org/x/text`、`v0.3.6` → `v0.3.7`。CVE ID や advisory URL は入力しなかった。
`scan.Command` を使った adapter の評価完了は 2026-11-15 03:08:54 UTC。
旧版は 4 GO レコード、更新先は 3 GO レコードに一致した。
SDK の CLI query 結果でも同じ ID 集合を取得した。
いずれも公表済みデータの照会であり、完全性・鮮度・到達性・実環境への影響を保証しない。

API 出力と共通 report は `adapters/govulncheck-discovery/testdata/live-20261115/` に保存した。
本実装への採用判断には、PR identity の検証、SHA の扱い、実行の信頼境界、再評価、表示方針の検討が残る。

## 11. 実装・仕様の原文（付録）

### adapters/govulncheck-discovery/go.mod

````text
module github.com/gkzz/renodash/adapters/govulncheck-discovery

go 1.27

require golang.org/x/vuln v1.9.0

require (
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/telemetry v0.0.0-20261010193011-f71dc92da45e // indirect
	golang.org/x/tools v0.52.0 // indirect
)

````

### adapters/govulncheck-discovery/go.sum

````text
github.com/google/go-cmp v0.6.0 h1:ofyhxvXcZhMsU5ulbFiLKl/XBFqE1GSq7atu8tAmTRI=
github.com/google/go-cmp v0.6.0/go.mod h1:17dUlkBOakJ0+DkrSSNjCkIjxS6bF9zb3elmeNGIjoY=
golang.org/x/mod v0.42.0 h1:ICeL3t2L30ljSPHHxDnR4fqVcBt/zR2zJAb9juavmQg=
golang.org/x/mod v0.42.0/go.mod h1:u5fzIrRInkL8MaO1cbRkUHT64v303YIT5KEqpNr7jRs=
golang.org/x/sync v0.24.0 h1:iqiDrGoDgyuvAKrhLQthUEzU/R378N78gXbBjxed7oA=
golang.org/x/sync v0.24.0/go.mod h1:sUUOizhqBxiL6pEWpqNLUiaJn1ShEbZ6BBqskPbjZm0=
golang.org/x/sys v0.49.0 h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA=
golang.org/x/sys v0.49.0/go.mod h1:hNLxWAXmnKAxqDtdwIYC4bM9oQPEecfsnNMuSxOs3og=
golang.org/x/telemetry v0.0.0-20261010193011-f71dc92da45e h1:P6NGsugec4oASJ+yKMNgTndeo0h2CbixZLvQIe9h5R4=
golang.org/x/telemetry v0.0.0-20261010193011-f71dc92da45e/go.mod h1:GNu+nb+E5axfGmP+Uu3ZVQu5lE/Xk42MjZOHWUD94/o=
golang.org/x/tools v0.52.0 h1:ryxg4U63oaUYk7uOLTF37n43f4qgtGvqV4Ioit1BaEs=
golang.org/x/tools v0.52.0/go.mod h1:ItcP5UNaVGG+Yc4eIYmaFLNJ4soEopkx5Y2d4lcEurs=
golang.org/x/tools/go/expect v0.1.1-deprecated h1:jpBZDwmgPhXsKZC6WhL20P4b/wmnpsEAGHaNy0n/rJM=
golang.org/x/tools/go/expect v0.1.1-deprecated/go.mod h1:eihoPOH+FgIqa3FpoTwguz/bVUSGBlGQU67vpBeOrBY=
golang.org/x/tools/go/packages/packagestest v0.1.1-deprecated h1:1h2MnaIAIXISqTFKdENegdpAgUXz6NrPEsbIeWaBRvM=
golang.org/x/tools/go/packages/packagestest v0.1.1-deprecated/go.mod h1:RVAQXBGNv1ib0J382/DPCRS/BPnsGebyM1Gj5VSDpG8=
golang.org/x/vuln v1.9.0 h1:12VaqnI9kJfqwRoPptxoqsUhQtdfrSbdksT7X/KY/sc=
golang.org/x/vuln v1.9.0/go.mod h1:E/Frwm0xp49fTyBW+6CQHdkvobknbwUM49yEpMGUauw=

````

### adapters/govulncheck-discovery/main.go

````go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/vuln/scan"
)

const database = "https://vuln.go.dev"
const sdkVersion = "golang.org/x/vuln v1.9.0"

type Target struct {
	Ecosystem    string `json:"ecosystem"`
	Distribution string `json:"distribution"`
	Module       string `json:"module"`
	Old          string `json:"old_version"`
	Proposed     string `json:"proposed_version"`
}
type Assessment struct {
	Version      string            `json:"version"`
	State        string            `json:"state"`
	Matches      []string          `json:"matches"`
	Limitations  []string          `json:"limitations"`
	Records      []json.RawMessage `json:"records"`
	SDKConfig    json.RawMessage   `json:"sdk_config,omitempty"`
	Query        string            `json:"query"`
	RetrievedAt  string            `json:"retrieved_at"`
	EvidenceFile string            `json:"evidence_file,omitempty"`
	Digest       string            `json:"sha256,omitempty"`
}
type Report struct {
	Experiment      string     `json:"experiment"`
	SDK             string     `json:"sdk"`
	Database        string     `json:"database"`
	Target          Target     `json:"target"`
	SelectionReason string     `json:"selection_reason"`
	CheckedAt       string     `json:"checked_at"`
	Old             Assessment `json:"old"`
	Proposed        Assessment `json:"proposed"`
	Coverage        string     `json:"coverage"`
}

// Runner is a narrow boundary around the public Go API, not an advisory client.
type Runner interface {
	Query(context.Context, string, io.Writer, io.Writer) error
}
type officialSDK struct{ db string }

func (s officialSDK) Query(ctx context.Context, query string, out, diagnostics io.Writer) error {
	db := s.db
	if db == "" {
		db = database
	}
	cmd := scan.Command(ctx, "-mode=query", "-json", "-db", db, query)
	cmd.Stdout, cmd.Stderr = out, diagnostics
	cmd.Stdin = bytes.NewReader(nil)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}
func decodeTarget(data []byte) (Target, error) {
	var t Target
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return t, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return t, fmt.Errorf("expected one target JSON object")
	}
	if t.Module == "" || t.Distribution == "" || t.Ecosystem == "" || t.Old == "" || t.Proposed == "" {
		return t, fmt.Errorf("module, ecosystem, distribution, and both versions are required")
	}
	return t, nil
}
func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("govulncheck-discovery", flag.ContinueOnError)
	input := fs.String("input", "", "target JSON; no advisory IDs")
	evidence := fs.String("evidence-dir", "", "new evidence directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *input == "" || *evidence == "" {
		return fmt.Errorf("usage: govulncheck-discovery -input target.json -evidence-dir NEW_DIR")
	}
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	if err != nil {
		return err
	}
	if len(data) > 64<<10 {
		return fmt.Errorf("target exceeds 64 KiB")
	}
	t, err := decodeTarget(data)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	r, err := inspect(ctx, officialSDK{}, t, *evidence)
	if err != nil {
		return err
	}
	data, err = json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(*evidence, "report.json"), data, 0600); err != nil {
		return err
	}
	n, err := out.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}
func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

````

### adapters/govulncheck-discovery/adapter.go

````go
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Static registration of one official API; no home-grown source client or range matcher.
func eligibility(t Target) string {
	if t.Ecosystem != "Go" || t.Distribution != "upstream-module" {
		return "unsupported ecosystem or distribution; requires a Go upstream module"
	}
	if !regexp.MustCompile(`^[a-z0-9.-]+\.[a-z]{2,}/[A-Za-z0-9._~!/+\-]+$`).MatchString(t.Module) || strings.Contains(t.Module, "..") || strings.Contains(t.Module, "//") || strings.HasSuffix(t.Module, "/") {
		return "unsupported Go module identity; runtime and stdlib are outside this experiment"
	}
	for _, v := range []string{t.Old, t.Proposed} {
		if strings.ContainsAny(v, "@ \t\r\n") || strings.HasPrefix(v, "-") {
			return "invalid compound module query"
		}
	}
	return ""
}
func emptyAssessment(v string) Assessment {
	return Assessment{Version: v, State: "Unknown", Matches: []string{}, Records: []json.RawMessage{}, Limitations: []string{}}
}
func inspect(ctx context.Context, sdk Runner, t Target, dir string) (Report, error) {
	r := Report{Experiment: "Official Go API advisory discovery — NOT MERGE", SDK: sdkVersion, Database: database, Target: t, Old: emptyAssessment(t.Old), Proposed: emptyAssessment(t.Proposed), Coverage: "Only Go Vulnerability Database records returned by the official API for this exact upstream module and supplied versions. No completeness, freshness, exploitability, symbol reachability, environment applicability, transitive coverage, Node.js runtime support, or current PR assessment is established. Saved bytes are SDK output, not raw database HTTP responses."}
	if err := os.Mkdir(dir, 0700); err != nil {
		return r, fmt.Errorf("create new evidence directory: %w", err)
	}
	r.SelectionReason = eligibility(t)
	if r.SelectionReason != "" {
		r.Old.Limitations = []string{r.SelectionReason}
		r.Proposed.Limitations = []string{r.SelectionReason}
	} else {
		r.SelectionReason = "Selected the official golang.org/x/vuln/scan query API for an upstream Go module. Target identity is caller-supplied."
		var err error
		r.Old, err = query(ctx, sdk, t, t.Old, "old", dir)
		if err != nil {
			return r, err
		}
		r.Proposed, err = query(ctx, sdk, t, t.Proposed, "proposed", dir)
		if err != nil {
			return r, err
		}
	}
	r.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	return r, nil
}

type capture struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (c *capture) WriteString(s string) (int, error) { return c.Write([]byte(s)) }

func (c *capture) Write(p []byte) (int, error) {
	if len(p) > c.limit-c.Len() {
		n, _ := c.Buffer.Write(p[:c.limit-c.Len()])
		c.exceeded = true
		return n, fmt.Errorf("SDK output capture limit exceeded")
	}
	return c.Buffer.Write(p)
}
func query(ctx context.Context, sdk Runner, t Target, v, label, dir string) (Assessment, error) {
	a := emptyAssessment(v)
	a.Query = t.Module + "@" + v
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	stdout, stderr := &capture{limit: 8 << 20}, &capture{limit: 1 << 20}
	err := sdk.Query(ctx, a.Query, stdout, stderr)
	a.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	a.EvidenceFile = label + ".sdk.json"
	sum := sha256.Sum256(stdout.Bytes())
	a.Digest = hex.EncodeToString(sum[:])
	if e := os.WriteFile(filepath.Join(dir, a.EvidenceFile), stdout.Bytes(), 0600); e != nil {
		return a, e
	}
	if e := os.WriteFile(filepath.Join(dir, label+".stderr.txt"), stderr.Bytes(), 0600); e != nil {
		return a, e
	}
	if err != nil {
		a.Limitations = append(a.Limitations, "official API query failed: "+err.Error())
	}
	if ctx.Err() != nil {
		a.Limitations = append(a.Limitations, "query context: "+ctx.Err().Error())
	}
	if stdout.exceeded || stderr.exceeded {
		a.Limitations = append(a.Limitations, "SDK output exceeds capture limit; saved output is incomplete")
	}
	if stderr.Len() > 0 {
		a.Limitations = append(a.Limitations, "SDK diagnostics present; inspect "+label+".stderr.txt")
	}
	parseStream(&a, t, stdout.Bytes())
	if len(a.Matches) > 0 {
		a.State = "Affected"
	} else if len(a.Limitations) == 0 {
		a.State = "NoMatchInCheckedSources"
	}
	meta, e := json.MarshalIndent(a, "", "  ")
	if e != nil {
		return a, e
	}
	if e := os.WriteFile(filepath.Join(dir, label+".assessment.json"), append(meta, '\n'), 0600); e != nil {
		return a, e
	}
	return a, nil
}
func parseStream(a *Assessment, t Target, data []byte) {
	dec := json.NewDecoder(bytes.NewReader(data))
	validConfig := false
	known := map[string][]byte{}
	for {
		var message map[string]json.RawMessage
		err := dec.Decode(&message)
		if err == io.EOF {
			break
		}
		if err != nil || len(message) != 1 {
			a.Limitations = append(a.Limitations, "invalid SDK JSON stream")
			break
		}
		for kind, raw := range message {
			switch kind {
			case "config":
				var c struct {
					Protocol string `json:"protocol_version"`
					Mode     string `json:"scan_mode"`
					DB       string `json:"db"`
				}
				if err := json.Unmarshal(raw, &c); err != nil || c.Protocol != "v1.0.0" || c.Mode != "query" || c.DB != database {
					a.Limitations = append(a.Limitations, "unsupported SDK protocol, mode, or database")
					continue
				}
				if validConfig {
					a.Limitations = append(a.Limitations, "duplicate SDK configuration")
				}
				validConfig = true
				a.SDKConfig = bytes.Clone(raw)
			case "osv":
				if !validConfig {
					a.Limitations = append(a.Limitations, "OSV record before valid SDK configuration")
					continue
				}
				var rec struct {
					ID        string `json:"id"`
					Withdrawn string `json:"withdrawn"`
					Affected  []struct {
						Package struct {
							Name      string `json:"name"`
							Ecosystem string `json:"ecosystem"`
						} `json:"package"`
					} `json:"affected"`
				}
				if err := json.Unmarshal(raw, &rec); err != nil || rec.ID == "" {
					a.Limitations = append(a.Limitations, "invalid SDK advisory record")
					continue
				}
				exact := false
				for _, item := range rec.Affected {
					if item.Package.Name == t.Module && item.Package.Ecosystem == "Go" {
						exact = true
					}
				}
				if !exact {
					a.Limitations = append(a.Limitations, rec.ID+": no exact Go module identity match")
					continue
				}
				var canonical bytes.Buffer
				if err := json.Compact(&canonical, raw); err != nil {
					a.Limitations = append(a.Limitations, "invalid native advisory JSON")
					continue
				}
				if before, ok := known[rec.ID]; ok {
					if bytes.Equal(before, canonical.Bytes()) {
						continue
					}
					a.Limitations = append(a.Limitations, "differing records for "+rec.ID+"; manual review required")
				}
				known[rec.ID] = bytes.Clone(canonical.Bytes())
				a.Records = append(a.Records, bytes.Clone(raw))
				if rec.Withdrawn == "" && !slices.Contains(a.Matches, rec.ID) {
					a.Matches = append(a.Matches, rec.ID)
				}
			case "progress": // Informational SDK progress; raw stream preserves the text.
			default:
				a.Limitations = append(a.Limitations, "unexpected SDK query message: "+kind)
			}
		}
	}
	if !validConfig {
		a.Limitations = append(a.Limitations, "missing valid SDK query configuration")
	}
}

````

### adapters/govulncheck-discovery/adapter_test.go

````go
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeSDK func(context.Context, string, io.Writer, io.Writer) error

func (f fakeSDK) Query(c context.Context, q string, o, e io.Writer) error { return f(c, q, o, e) }
func testTarget() Target {
	return Target{Ecosystem: "Go", Distribution: "upstream-module", Module: "example.org/library", Old: "v1.2.2", Proposed: "v1.2.3"}
}

const config = `{"config":{"protocol_version":"v1.0.0","scan_mode":"query","db":"https://vuln.go.dev"}}` + "\n"

// Fabricated SDK output; no real vulnerability is asserted by this record.
const record = `{"osv":{"id":"GO-SYNTHETIC-1","affected":[{"package":{"name":"example.org/library","ecosystem":"Go"}}],"aliases":["CVE-SYNTHETIC"]}}` + "\n"

func TestSDKAdapter(t *testing.T) {
	var calls []string
	sdk := fakeSDK(func(ctx context.Context, q string, out, stderr io.Writer) error {
		calls = append(calls, q)
		io.WriteString(out, config)
		if strings.HasSuffix(q, "@v1.2.2") {
			io.WriteString(out, record)
		}
		return nil
	})
	dir := filepath.Join(t.TempDir(), "evidence")
	got, err := inspect(context.Background(), sdk, testTarget(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "example.org/library@v1.2.2,example.org/library@v1.2.3" {
		t.Fatal(calls)
	}
	if got.Old.State != "Affected" || got.Proposed.State != "NoMatchInCheckedSources" || len(got.Old.Records) != 1 || len(got.Old.Matches) != 1 {
		t.Fatal(got)
	}
	for _, a := range []Assessment{got.Old, got.Proposed} {
		if raw, err := os.ReadFile(filepath.Join(dir, a.EvidenceFile)); err != nil || len(raw) == 0 || len(a.Digest) != 64 || a.RetrievedAt == "" {
			t.Fatal(a, err)
		}
	}
	if _, err := inspect(context.Background(), sdk, testTarget(), dir); err == nil {
		t.Fatal("existing evidence overwritten")
	}
}
func TestSDKFailuresRemainUnknown(t *testing.T) {
	for _, mode := range []string{"failure", "empty", "malformed", "missing-config", "wrong-mode", "unknown-protocol", "wrong-module", "limit", "partial"} {
		t.Run(mode, func(t *testing.T) {
			sdk := fakeSDK(func(_ context.Context, _ string, out, stderr io.Writer) error {
				switch mode {
				case "failure":
					return fmt.Errorf("synthetic database failure")
				case "empty":
					return nil
				case "malformed":
					io.WriteString(out, config+"{")
				case "missing-config":
					io.WriteString(out, record)
				case "wrong-mode":
					io.WriteString(out, strings.Replace(config, "query", "source", 1))
				case "unknown-protocol":
					io.WriteString(out, strings.Replace(config, "v1.0.0", "v2.0.0", 1))
				case "wrong-module":
					io.WriteString(out, config+strings.Replace(record, "example.org/library", "example.org/other", 1))
				case "limit":
					_, err := io.WriteString(out, strings.Repeat("x", (8<<20)+1))
					return err
				case "partial":
					io.WriteString(out, config+record)
					return fmt.Errorf("synthetic partial failure")
				}
				return nil
			})
			dir := filepath.Join(t.TempDir(), "evidence")
			got, err := inspect(context.Background(), sdk, testTarget(), dir)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "limit" {
				data, err := os.ReadFile(filepath.Join(dir, got.Old.EvidenceFile))
				if err != nil || len(data) > 8<<20 || !strings.Contains(strings.Join(got.Old.Limitations, " "), "capture limit") {
					t.Fatal("capture was not bounded", len(data), err)
				}
			}
			want := "Unknown"
			if mode == "partial" {
				want = "Affected"
			}
			if got.Old.State != want || len(got.Old.Limitations) == 0 {
				t.Fatal(got.Old)
			}
		})
	}
}
func TestUnsupportedTargetSkipsSDK(t *testing.T) {
	for _, module := range []string{"stdlib", "example.org/library@v1.0.0", "-help"} {
		in := testTarget()
		in.Module = module
		got, err := inspect(context.Background(), fakeSDK(func(context.Context, string, io.Writer, io.Writer) error {
			t.Fatal("unsupported input queried")
			return nil
		}), in, filepath.Join(t.TempDir(), "evidence"))
		if err != nil || got.Old.State != "Unknown" || got.SelectionReason == "" {
			t.Fatal(got, err)
		}
	}
}
func TestOfficialSDKCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (officialSDK{}).Query(ctx, "example.org/library@v1.2.2", io.Discard, io.Discard); err == nil {
		t.Fatal("canceled SDK query succeeded")
	}
}
func TestSavedOfficialSDKOutput(t *testing.T) {
	var digests map[string]string
	raw, err := os.ReadFile("testdata/live-20261115/sha256.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &digests); err != nil {
		t.Fatal(err)
	}
	for name, want := range digests {
		data, err := os.ReadFile(filepath.Join("testdata/live-20261115", name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			t.Fatal("capture digest mismatch", name)
		}
	}
	in := Target{Ecosystem: "Go", Distribution: "upstream-module", Module: "golang.org/x/text", Old: "v0.3.6", Proposed: "v0.3.7"}
	sdk := fakeSDK(func(_ context.Context, q string, out, stderr io.Writer) error {
		name := "proposed.sdk.json"
		if strings.HasSuffix(q, "@v0.3.6") {
			name = "old.sdk.json"
		}
		data, err := os.ReadFile(filepath.Join("testdata/live-20261115", name))
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	})
	got, err := inspect(context.Background(), sdk, in, filepath.Join(t.TempDir(), "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Old.State != "Affected" || got.Proposed.State != "Affected" || len(got.Old.Matches) != 4 || len(got.Proposed.Matches) != 3 {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(got.Old.Matches, []string{"GO-2021-0113", "GO-2022-1059", "GO-2026-5970", "GO-2026-6629"}) ||
		!reflect.DeepEqual(got.Proposed.Matches, []string{"GO-2022-1059", "GO-2026-5970", "GO-2026-6629"}) {
		t.Fatal(got)
	}
	if len(got.Old.Limitations) != 0 || len(got.Proposed.Limitations) != 0 {
		t.Fatal(got)
	}
	for _, a := range []Assessment{got.Old, got.Proposed} {
		for _, raw := range a.Records {
			var r map[string]any
			if err := json.Unmarshal(raw, &r); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestSDKRecordHandling(t *testing.T) {
	for _, tc := range []struct {
		name, stream, want string
		limitations        bool
		records            int
	}{
		{"duplicate", config + record + record, "Affected", false, 1},
		{"conflict", config + record + strings.Replace(record, "CVE-SYNTHETIC", "CVE-SYNTHETIC-OTHER", 1), "Affected", true, 2},
		{"withdrawn", config + strings.Replace(record, `"id":"GO-SYNTHETIC-1"`, `"id":"GO-SYNTHETIC-1","withdrawn":"2020-01-01T00:00:00Z"`, 1), "NoMatchInCheckedSources", false, 1},
		{"partial-stream", config + record + "{", "Affected", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sdk := fakeSDK(func(_ context.Context, _ string, out, stderr io.Writer) error {
				_, err := io.WriteString(out, tc.stream)
				return err
			})
			got, err := inspect(context.Background(), sdk, testTarget(), filepath.Join(t.TempDir(), "evidence"))
			if err != nil {
				t.Fatal(err)
			}
			if got.Old.State != tc.want || len(got.Old.Records) != tc.records || (len(got.Old.Limitations) > 0) != tc.limitations {
				t.Fatal(got.Old)
			}
		})
	}
}
func TestTargetValidation(t *testing.T) {
	for _, data := range []string{`null`, `{}`, `{"ecosystem":"Go","cve":"not-an-input"}`, `{} {}`} {
		if _, err := decodeTarget([]byte(data)); err == nil {
			t.Fatal("accepted invalid target", data)
		}
	}
	for _, mode := range []string{"ecosystem", "distribution", "version"} {
		in := testTarget()
		switch mode {
		case "ecosystem":
			in.Ecosystem = "npm"
		case "distribution":
			in.Distribution = "debian"
		case "version":
			in.Proposed = "v1.2.3@other"
		}
		got, err := inspect(context.Background(), fakeSDK(func(context.Context, string, io.Writer, io.Writer) error {
			t.Fatal("unsupported target called API")
			return nil
		}), in, filepath.Join(t.TempDir(), "evidence"))
		if err != nil || got.Proposed.State != "Unknown" {
			t.Fatal(got, err)
		}
	}
}

````

### 動作仕様の原文

````markdown
# Official Go API advisory discovery spike

This NOT MERGE experiment wraps the public `golang.org/x/vuln/scan` API.
The pinned module version is `v1.9.0`. Query mode accepts a module and version without advisory IDs.
The official implementation owns database acquisition and affected-version matching.
The adapter delegates HTTP requests, version comparisons, and affected-range evaluation to that API.
It does not change the renodash CLI, production workflows, or merge policy.

## Input and validation

Accept one target JSON object with `ecosystem`, `distribution`, `module`, `old_version`, and `proposed_version`.
Reject unknown fields, trailing JSON, missing identity, and input larger than 64 KiB.
The static registry supports only ecosystem `Go` and distribution `upstream-module`.
Require a module path with a domain and path component. Standard library and runtime targets are unsupported.
Reject option-like or compound query inputs before calling the API.
Delegate version syntax and affected-range matching to the official implementation.
Identity remains caller-supplied, not verified against a PR or dependency graph.

## Behavior

Call `scan.Command` with `-mode=query`, `-json`, the fixed database `https://vuln.go.dev`, and `module@version`.
Use `Start` and `Wait`; do not launch a separate executable or target code.
Query old and proposed versions separately with a 30-second deadline per query and 60-second operation deadline.
Limit captured stdout to 8 MiB and diagnostics to 1 MiB per query.
Do not overwrite an existing evidence directory.
Save API stdout, diagnostics, SHA-256 digest, query, UTC completion time, and reported API configuration.
The saved stream is SDK output, not the original database HTTP response bytes.
Database retrieval limits, caching, and redirects remain upstream implementation behavior.

Accept the `v1.0.0` JSON protocol and query-mode configuration for the expected database.
Interpret returned `osv` records as matches reported by the official query implementation.
Check exact module identity in returned records, preserve native record JSON and aliases, and ignore withdrawn records.
Keep confirmed matches when a later stream or query failure occurs.
Source failures, output limits, invalid streams, conflicting IDs, and missing configuration remain limitations.
Without a confirmed match, unresolved limitations produce `Unknown`.
A completed empty query produces `NoMatchInCheckedSources` for this database snapshot and exact module scope only.
An empty result never establishes safety, freshness, or complete database coverage.
Do not merge independent records solely because they share an alias.

## Output and failures

Write the common JSON report to stdout and `report.json` in the evidence directory.
Include the target, SDK version, database, selection reason, two assessments, assessment time, evidence files, and coverage limits.
Input, evidence-write, and output errors return nonzero status.
Query failures remain visible as limitations, independently of the process exit status.
Do not infer exploitability, call reachability, environment applicability, transitive coverage, or Node.js runtime support.
No result approves, blocks, or merges a PR.

## Implementation and test evidence

See [the isolated module](../../../adapters/govulncheck-discovery/).
Run `go test ./... -count=1` and `go vet ./...` inside that module.
Tests inject SDK output for protocol, identity, partial failure, evidence preservation, and unsupported inputs.
A canceled-context test exercises the real public API without network access.
Saved query outputs support offline reproduction. Live verification and OSS comparisons belong in Issue #80.

````

## 12. 追加したライセンス表示の原文

### adapters/govulncheck-discovery/THIRD_PARTY_NOTICES.md

````markdown
# Third-party notices

## golang.org/x/vuln v1.9.0

This experiment imports the public Go API from `golang.org/x/vuln/scan`.
Source: https://github.com/golang/vuln/tree/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5

The following notice reproduces the pinned module's LICENSE file.
Source distributions must preserve its copyright, conditions, and disclaimer.
Binary distributions must reproduce them in accompanying documentation or materials.
Before distributing binaries, check and include the required notices for all bundled dependencies.
This file records the direct SDK dependency; it is not a complete binary distribution notice inventory.

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

## Go Vulnerability Database entries

The database entries are attributed to The Go Authors / Go Vulnerability Database at https://vuln.go.dev/.
They are available under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
See the [upstream statement](https://github.com/golang/vulndb/blob/master/README.md#license) and [saved-data attribution](testdata/README.md#data-attribution-and-license).
The saved reports select and arrange SDK output; the database license remains applicable to the included entries.
Original record references are preserved. No endorsement is implied.
The material is provided without warranties under the license.

````

### adapters/govulncheck-discovery/testdata/README.md

````markdown
# SDK inputs and saved output

`target.json` provides a historical Go module update pair without advisory IDs or URLs.
It is a synthetic target, not an actual PR diff or a recommended update.

`live-20261115` preserves output acquired through the public Go API at module version `v1.9.0`.
`old.sdk.json` and `proposed.sdk.json` are the original SDK JSON streams.
`report.json` records module identity, queries, API configuration, completion times, digests, matches, and limitations.
`sha256.json` records the captured file digests for offline verification.
The recorded assessment completed on 2026-11-15 at 03:08:54 UTC.
SDK query configuration records the database's last modification time separately.
The SDK output is not a capture of raw upstream HTTP bytes.

The target is `golang.org/x/text`, ecosystem `Go`, distribution `upstream-module`.
Old version `v0.3.6` matched four GO records; proposed version `v0.3.7` matched three.
The official API, not local range evaluation, produced these matches.
`TestSavedOfficialSDKOutput` replays the streams without network access.
Saved advisory IDs are expected outputs, not inputs to discovery.
This does not establish complete coverage or applicability to a deployed application.

Synthetic tests use fabricated `example.org/library` records and explicitly synthetic IDs.
Their minimal records exercise stream conversion, not the official version-matching algorithm.
The canceled-context test exercises the actual API without network access.

## Data attribution and license

The database entries in `live-20261115` originate from The Go Authors / Go Vulnerability Database at [vuln.go.dev](https://vuln.go.dev/).
They are used under [Creative Commons Attribution 4.0 International (CC BY 4.0)](https://creativecommons.org/licenses/by/4.0/).
See the [upstream license statement](https://github.com/golang/vulndb/blob/master/README.md#license).
Existing record references, aliases, publication times, and modification times remain in the saved data.

The SDK streams preserve the captured API output.
`report.json` selects and arranges SDK results with adapter metadata; its database entries remain subject to CC BY 4.0.
The surrounding notes summarize the results; they do not imply endorsement by the original authors.
Database entries are provided without warranties under the license.
The repository's MIT license does not replace the database license.

````
