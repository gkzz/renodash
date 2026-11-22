# Goのテストライブラリ比較

renodashでテストを書く人に向けて、テストの実行、値の比較、非同期処理の検証に使う候補を比較します。
現状には、標準の `testing` と `go-cmp` の組み合わせが適しています。
この資料は、既存のテスト構成に基づく比較と判断材料を示します。
採用方針と必須条件は [Testing strategy](testing.md) に集約します。

## 現在の構成

- [go.mod](../go.mod) で宣言している外部モジュールは、テスト用の `github.com/google/go-cmp v0.7.0` です。
- [パーサーのテスト](../internal/dashboard/parser_test.go) は、`testing` と `cmp.Diff` で構造体を検証しています。
- [コマンドのテスト補助](../internal/command/fakegh_test.go) は、Goのテスト実行ファイルを偽の `gh` として使います。
- [統合テスト](../internal/integration/offline_test.go) は、保存した入力と実際のプロセスを使ってCLIを検証します。

比較では、失敗時の差分、比較の厳密さ、既存テストとの書き方の統一、依存関係と移行作業を重視します。
フロントエンド用のテストツールは対象に含めません。

## 役割と機能の比較

以下の機能は各ライブラリの公式資料に基づきます。後の適合性評価とは分けて読んでください。
テスト実行の仕組みと、結果を検証するライブラリは役割が異なり、組み合わせて使えます。

| 候補 | 主な役割 | 得意なこと | 注意点・公式資料 |
| --- | --- | --- | --- |
| Go標準の `testing` | テスト実行と失敗の報告 | `t.Run` によるサブテスト、一時ディレクトリ、後処理 | 複雑な値の差分は別途用意する。[testing](https://pkg.go.dev/testing) |
| `testing/synctest` | プロセス内の並行処理のテスト | goroutineの同期と仮想時刻によるタイマーの検証 | ネットワークや外部プロセスとの連携は避ける。[testing/synctest](https://pkg.go.dev/testing/synctest) |
| `go-cmp` | Goの値の比較と差分表示 | 入れ子の構造体の差分、比較方法の指定 | 非公開フィールドは既定でpanicになる。`Equal` メソッドが比較に使われる場合がある。[go-cmp](https://github.com/google/go-cmp) |
| Testify | アサーション、モック、テストスイート | `assert` で失敗を報告して続行し、`require` でそのテストを中断する。条件成立の待機も可能 | `require` はテストを実行するgoroutineから呼ぶ。`suite` は並列テストをサポートしない。[Testify](https://github.com/stretchr/testify) |
| Gomega | 条件を検証するマッチャー | `Eventually` による条件成立の待機、`Consistently` による継続的な確認 | `NewWithT(t)` で標準テストにも組み込める。Ginkgoは必須ではない。[Gomega](https://onsi.github.io/gomega/) |
| Ginkgo + Gomega | テスト構造の記述と条件の検証 | `Describe`、`Context`、`It` による階層的なシナリオ記述 | 独自の記述方法とスイート初期化を学ぶ必要がある。[Ginkgo](https://onsi.github.io/ginkgo/) |

Testifyの `assert`、`require`、`mock`、`suite` は個別に選べます。
アサーションの導入だけで、モックやスイートへの移行が必要になるわけではありません。
待機には [`assert.Eventually`](https://pkg.go.dev/github.com/stretchr/testify/assert#Eventually)、`assert.EventuallyWithT`、`assert.Never` があります。
`EventuallyWithT` は、各回の評価で複数のアサーションを実行できます。

## renodashでの適合性

次の表は、現行コードとテスト方針を踏まえた評価です。
実行速度や保守工数の測定結果ではありません。

| 選択肢 | 既存テストへの影響 | 向いている用途 | このリポジトリでの判断 |
| --- | --- | --- | --- |
| `testing` のみ | `cmp.Diff` を使う箇所では差分表示の代替が必要 | 単純な条件、文字列、終了コード | 基本として使う。構造体の検証には `go-cmp` を併用する |
| `testing` + `go-cmp` | 現行の構成を継続できる | DashboardやResearchの構造比較 | 現状の推奨。テストの制御と比較を明確に分けられる |
| `testing` + `testing/synctest` | プロセス内の同期と時間の扱いを見直す | goroutineやタイマーを使う並行処理 | 該当するテストが増えた場合に優先して検討する |
| `testing` + Testify | 検証の書き換えと、使うAPIの選定が必要 | エラー確認などの定型処理、条件成立の待機 | 定型処理が読みにくくなった場合に検討する |
| `testing` + Gomega | 待機条件とマッチャーの書き方を追加する | 時間内の状態変化を繰り返し確認する | 非同期条件の検証が増えた場合に検討する |
| Ginkgo + Gomega | スイート構成とテスト本体の書き換えが必要 | 共通の前処理を持つ階層的なシナリオ | 現在のテーブル駆動テストを置き換える必要性は低い |

`go-cmp` はモックや待機の機能を提供するライブラリではありません。
一方、TestifyやGomegaを追加しても、外部プロセスを検証する偽の `gh` は別途必要です。
関数呼び出しのモックだけでは、実行ファイルの検索やプロセスのキャンセルを検証できません。

## 選ぶときに確認すること

### 比較の意味を保てるか

renodashでは、構造体全体の比較でnilと空のコレクション、スライスの順序を区別します。
単純なスライスやマップの比較には、[既存の比較方針](testing.md#equality-comparisons) に従って標準の関数を使います。
ライブラリを変える場合は、同じ「等しい」という名前でも比較結果が変わらないか確認します。

比較APIによって、値が等しいと判断する規則は異なります。

| 比較API | 基本となる比較規則 | 公式資料 |
| --- | --- | --- |
| `cmp.Diff` | `go-cmp` の規則。型の `Equal` メソッドが使われる場合がある | [cmp](https://pkg.go.dev/github.com/google/go-cmp/cmp) |
| Testify `assert.Equal` | 基本は `reflect.DeepEqual`。`[]byte` は専用の比較を使う | [実装](https://github.com/stretchr/testify/blob/v1.12.1/assert/assertions.go) |
| Gomega `Equal` | `reflect.DeepEqual`。両方がnilの場合はエラー | [Equal](https://onsi.github.io/gomega/#equalexpected-any) |
| Gomega `BeComparableTo` | `go-cmp`。比較オプションも指定できる | [BeComparableTo](https://onsi.github.io/gomega/#becomparabletoexpected-any-options-cmpoption) |

`reflect.DeepEqual` は非公開フィールドも比較しますが、`go-cmp` は既定でpanicになります。
非公開フィールドや `Equal` メソッドを追加した場合も、比較の意味を見直します。
移行時はテストの成否に加えて、失敗時にどのフィールドの差分が分かるかも確認します。
テストを通すためだけにフィールドを無視したり、nilと空を同一視したりすると、不具合を見逃す可能性があります。

### 失敗後に続けるか

前提条件が壊れた場合は、後続の検証を止めます。
独立した項目の不一致は、続行して複数の差分を報告できます。
現行の `t.Fatalf` と `t.Errorf` でも、この使い分けができます。
Testifyを選ぶ場合も、`require` と `assert` の役割を揃えます。

### 待機の理由が明確か

GomegaやTestifyの待機機能は、一定時間内に条件が成立することを検証する用途に適しています。
現在のキャンセルテストは、ローカル接続で子プロセスの起動を通知してからキャンセルします。
このように直接同期できる場面では、その仕組みを維持できます。
待機機能を導入する場合も、タイムアウトと失敗時の後処理が必要です。

### 標準ライブラリで並行処理を検証できるか

[go.mod](../go.mod) が指定するGo 1.27.0では、標準の `testing/synctest` を利用できます。
同一プロセス内のgoroutineの同期や、仮想時刻を使ったタイマーのテストで検討します。
[公式資料](https://pkg.go.dev/testing/synctest)では、ネットワークや外部プロセスとの連携を避けるよう案内しています。
実プロセスとループバック接続を使う現在のキャンセルテストには、既存の明示的な同期を維持します。

### テスト方針に適合するか

候補の評価では、[テスト方針](testing.md)への適合を確認します。
特に、ライブラリの機能だけでは代替できない次の検証に注意します。

- 保存した入力と偽の `gh` を使い、認証情報や外部ネットワークに依存せず統合テストを実行する。
- テスト対象のGo実装から期待値を生成せず、独立したgoldenファイルを使う。由来は[fixtureの説明](../internal/summary/testdata/README.md)を参照する。
- 実プロセスでCLIの入出力、終了コード、キャンセルを検証する。
- 実行ファイル名と `PATH` 検索で、プラットフォームごとの要件を守る。

アサーションや関数呼び出しのモックを導入する場合も、これらの検証を維持します。

### 追加依存に見合う改善があるか

候補を導入すると、モジュールの更新確認や、レビュー時に理解するAPIが増えます。
導入前には代表的なテストで、失敗メッセージの読みやすさと比較の意味を確認します。
依存数、対応するGoのバージョン、ライセンスは、採用するバージョンを決めた時点で確認します。
新しいテスト依存も、既存の[Go依存関係の脆弱性検査](../.github/workflows/lint.yml)で使う `govulncheck` の確認対象に含めます。

実行方法を変更する場合は、[CIの検証範囲](testing.md#command-subprocess-tests)への影響も確認します。
Ubuntuでは全Goテスト、macOSではコマンドと統合テストを実行しており、Windowsでの実行は未検証です。
独自ランナーの必要性や、既存の `go test` による実行を維持できるかも確認します。
Testifyの `suite` を検討する場合は、並列テストをサポートしない制約を確認します。

## testingのみとgo-cmp併用の実装比較

[比較用テスト](../internal/testingcomparison/comparison_test.go)に、次の2案を実装しています。
どちらも標準の `testing` で実行し、パーサーの入力と期待値を共通にしています。
既存テストの置き換えではなく、比較用の独立したパッケージです。

| 案 | 一致の判定 | 不一致の診断 |
| --- | --- | --- |
| 標準のみ | `reflect.DeepEqual(want, got)` | `fmt.Sprintf` の `%#v` で期待値と実際の値を表示する |
| `go-cmp` 併用 | `cmp.Diff(want, got)` が空か確認する | `cmp.Diff` の差分を表示する |

両案とも、解析エラーでは `t.Fatal` で中断し、値の不一致では `t.Error` で報告します。
標準のみの案も構造体全体を比較し、JSON変換やnilと空の同一視は行いません。
同じモジュール内で比較するため、`go.mod` の `go-cmp` 依存は残しています。

リポジトリルートから、開発環境のGoで実行します。
ツールの準備は[開発手順](development.md#setup-and-tests)を参照してください。

```sh
go test ./internal/testingcomparison -run '^TestParserComparison/standard$' -count=1 -v
go test ./internal/testingcomparison -run '^TestParserComparison/go-cmp$' -count=1 -v
go test ./internal/testingcomparison -run '^TestMismatchDiagnostics$' -count=1 -v
```

`TestParserComparison` は、空入力と順序付きのチェックボックスを両案で検証します。
`TestMismatchDiagnostics` は意図的に異なる値を渡し、不一致を検出したら成功します。
診断文は `t.Log` で出力するため、`-v` を付けて確認します。
通常のパーサーテストでは、同じ診断文を `t.Error` に渡します。

ローカルのLinux環境で、両案のパーサーテストと次の不一致検出が成功しました。

| 不一致の種類 | 標準のみの診断 | `go-cmp` の診断 |
| --- | --- | --- |
| `Items[0].Research.Package` の変更 | 不一致を検出するが、`Research` はポインターアドレスとして表示される | 入れ子の `Package` と変更前後の値を表示する |
| `Items` のnilと空スライス | `nil` と空の値を全体表示から読み取れる | `Items` のnilと空の差分を表示する |
| `Items` の順序変更 | 全体表示から各要素の順序を照合する | 変更された各要素の `Branch` を差分で表示する |

今回の値では、判定の厳密さに差はなく、入れ子の不一致を調べる手間に差がありました。
標準のみでもフィールド別の検証や表示を追加すれば、診断を改善できます。
その場合は、型や検証項目の追加に合わせて独自の診断処理も保守する必要があります。
この比較は標準ライブラリで実装できるすべての診断方法を評価したものではありません。
非公開フィールドや `Equal` メソッドを持つ型の挙動、性能、保守工数は測定していません。

## 推奨する使い分け

現時点では、標準の `testing` と `go-cmp` を継続する案を推奨します。
構造化データの差分を確認でき、既存テストの書き方も維持できます。
次の表は、具体的な問題と必要な機能に応じて再評価するための目安です。
テストの行数や表の条件だけで採用を決めず、現在の構成で解決できるかも確認します。

| 状況 | 判断 |
| --- | --- |
| 構造体全体の差分比較が中心 | `go-cmp` を維持する |
| `t.Errorf` や `t.Fatalf` の定型処理が反復し、読みにくい | Testifyの `assert` と `require` を検討する |
| 条件のポーリングが必要なテストが増える | GomegaとTestifyの待機機能を比較する |
| プロセス内のgoroutineやタイマーのテストが増える | `testing/synctest` を優先して検討する |
| 階層的なシナリオや共通の前処理が複雑になる | Ginkgoの導入効果を検証する |

採用前に代表的なテストで、比較の意味、失敗時の診断、依存関係、CIへの影響を確認します。

比較用テストでは既存の `go-cmp` だけを利用し、新しいライブラリの追加や既存テストの移行は行っていません。
候補間の性能比較と、未導入ライブラリを使った動作検証は未実施です。
