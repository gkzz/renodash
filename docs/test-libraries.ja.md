# Goのテストライブラリ比較

renodashでテストを書く人に向けて、テストの実行、値の比較、非同期処理の検証に使う候補を比較します。
再調査では、まず標準ライブラリだけで失敗メッセージを改善する案を推奨します。
この資料は、既存のテスト構成に基づく比較と判断材料を示します。
テストの基本方針は [Testing strategy](testing.md) に集約します。
このブランチの `go-cmp` 利用は比較実験であり、採用決定ではありません。

## 現在の構成

### 解決したい課題

課題は、期待値と実際の値が違うとき、どの入れ子の値が違うかを読み取りにくいことです。
元の標準ライブラリ案も、構造体全体の不一致は検出できていました。
内部データの検証を増やす必要性は、この課題からは導けません。

### 比較ブランチで試している構成

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
| 標準ライブラリのみ | 全体比較を残し、読みにくい失敗メッセージを補う | 単純な条件から構造体全体の比較まで | 最初に試す案。今回の入れ子の値も表示できた |
| `testing` + `go-cmp` | 実験中の差分表示を利用できる | DashboardやResearchの構造比較 | 標準の診断では調査が難しい場合に再検討する |
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
非公開フィールドの実測結果は、後の[比較実験](#非公開フィールドは比較対象に応じて扱う)にまとめます。
`Equal` メソッドの影響は今回の追加実験では検証していません。
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
既存の[Go依存関係の脆弱性検査](../.github/workflows/lint.yml)は、`govulncheck` v1.9.0相当のコミットに固定しています。
実行コマンドの `govulncheck ./...` は、既定では `_test.go` を解析対象に含めません。
そのため、テストからのみ使う `go-cmp` などの依存は、現在のCIではテストからの呼び出しを解析できません。
テストコードとその依存も解析するには、`govulncheck -test ./...` が必要です。
`-test` の既定値がfalseであることは、[固定コミットのフラグ定義](https://github.com/golang/vuln/blob/e672cbe8fbc8c76e28fcb6db0efd9ce354e0dac5/internal/scan/flags.go)で確認できます。
この比較では説明のみを訂正し、CIの解析範囲は変更しません。

実行方法を変更する場合は、[CIの検証範囲](testing.md#command-subprocess-tests)への影響も確認します。
Ubuntuでは全Goテスト、macOSではコマンドと統合テストを実行しており、Windowsでの実行は未検証です。
独自ランナーの必要性や、既存の `go test` による実行を維持できるかも確認します。
Testifyの `suite` を検討する場合は、並列テストをサポートしない制約を確認します。

## testingのみとgo-cmp併用の実装比較

### 全体比較を残して、ポインター先を表示する

[比較用テスト](../internal/testingcomparison/comparison_test.go)で、同じ入力を次の3案に渡します。
どの案も標準の `testing` で実行します。

| 案 | 一致の判定 | 不一致の診断 |
| --- | --- | --- |
| `standard` | `reflect.DeepEqual` | Dashboard全体を `%#v` で表示 |
| `standard-detailed` | 同じ `reflect.DeepEqual` | 全体表示に加え、不一致のResearchを直接表示 |
| `go-cmp` | `cmp.Diff` が空か確認 | ライブラリの差分表示 |

[標準のみの実装](../internal/testingcomparison/standard_test.go)は、全体比較で不一致になった場合だけ、診断を補います。
同じ位置のItemにあるResearchを比較し、違っていれば `Items[番号].Research` と両方の値を表示します。
Researchのポインターを直接 `%#v` に渡すと、今回の型ではアドレスだけでなく内容も表示できます。
全体比較を残すため、表示を補っていないフィールドの不一致も見逃しません。

- nilと空、スライスの順序を区別する判定は維持します。
- JSON変換、フィールドの無視、独自の再帰的な差分エンジンは使いません。
- 解析エラーは `t.Fatal`、値の不一致は `t.Error` で報告します。

### 診断の違いと限界

| 確認したケース | `standard` | `standard-detailed` | `go-cmp` |
| --- | --- | --- | --- |
| 入れ子のPackage変更 | Researchはアドレスのみ | Itemの位置、Packageと両方の値を表示 | Packageの差分を表示 |
| グループ内のCurrentVersion変更 | Researchはアドレスのみ | Research内のバージョンを表示 | この追加ケースでは未実行 |
| Research内のnilと空 | 内容を読み取れない | `[]string(nil)` と `[]string{}` を表示 | この追加ケースでは未実行 |
| Itemsのnilと空 | 全体表示から判別 | 同左 | 差分を表示 |
| Itemsの順序変更 | 全体表示から照合 | 同左 | Branchの差分を表示 |

標準の改善案は、Research全体を表示します。変更箇所だけを自動で抜き出す機能はありません。
大きなResearchでは表示が長く、値を目で照合する必要があります。
Itemの追加・削除や並べ替えも、全体表示から読み取ります。
将来さらにポインターが入れ子になる場合は、表示を補うか、別の方法を検討します。

今回の例では、標準ライブラリでも入れ子の値を表示できました。
`go-cmp` の差分表示は便利ですが、採用が必要だとする根拠にはなりません。

### 非公開フィールドは比較対象に応じて扱う

Goでは、小文字で始まるフィールドは同じパッケージ内からアクセスできます。
今回のDashboard・Researchと、その配下で比較する型のフィールドは公開されています。
一方、解析途中に使う [dependencyRecord](../internal/dashboard/research.go) には、非公開の `name` と `updates` があります。
テスト用の [fakeGitHubClient](../internal/command/github_client_test.go) にも、非公開の関数フィールドがあります。
「アプリケーションに非公開フィールドがない」という意味ではありません。

公開範囲は用途で決めます。
解析結果として他のパッケージに渡すデータと、解析途中のデータを分ける現在の構成は自然です。
比較ライブラリの都合で、内部フィールドを公開する必要はありません。

[非公開フィールドの実験](../internal/dashboard/comparison_spike_test.go)では、実際の `dependencyRecord` を比較しました。

| 入力 | `reflect.DeepEqual` | 既定の `cmp.Diff` | `AllowUnexported` | `IgnoreUnexported` |
| --- | --- | --- | --- | --- |
| 値が同じ別のレコード | 一致 | panic | 一致 | 一致 |
| 非公開のnameだけ変更 | 不一致 | panic | 不一致 | 一致してしまう |
| 非公開のupdatesだけ変更 | 不一致 | panic | 不一致 | 一致してしまう |
| updatesのnilと空 | 不一致 | panic | 不一致 | 一致してしまう |
| 公開のCurrentVersionを変更 | 不一致 | panic | 不一致 | 不一致 |

`AllowUnexported(dependencyRecord{})` は、指定した型の非公開フィールドも比較対象にします。
`IgnoreUnexported(dependencyRecord{})` は、それらを比較から外します。
同じパッケージのテストでも、`go-cmp` の既定設定ではpanicしました。
この実験ではpanicを捕捉し、想定どおりの挙動かを検証しています。
同じポインター同士など、すべての入力でpanicすることを主張するものではありません。

内部レコードを直接比較するかは、比較APIとは別に判断します。

| 確認したいこと | 優先するテスト |
| --- | --- |
| 解析結果のパッケージ名やバージョン | Parseが返すResearchを確認する |
| CLIの出力やエラー | 既存のコマンドテスト・統合テストで確認する |
| 複雑な内部処理を単独で検証したい | 必要な場合だけ内部関数の結果を直接確認する |

今回の内部レコードの直接比較は、比較規則を調べる実験です。
アプリケーションの全内部状態を固定する方針ではありません。
非公開の関数フィールドは今回の比較実験に含めていません。
`reflect.DeepEqual` でも、nil以外の関数同士は等しいと判断しないため、万能な比較方法ではありません。
詳細は [reflect.DeepEqual](https://pkg.go.dev/reflect#DeepEqual) を参照してください。

### 再現手順と確認範囲

リポジトリルートから、[開発環境](development.md#setup-and-tests)のGoで実行します。

```sh
go test ./internal/testingcomparison -count=1 -v
go test ./internal/dashboard -run '^TestComparisonSpikeUnexportedFields$' -count=1 -v
```

意図的に異なる値を渡す実験は、不一致を検出できれば成功します。
診断文は `t.Log` で表示します。通常のテストでは同じ文を `t.Error` に渡します。
追加した標準案では、Package、グループ内のバージョン、Researchのnil、内部スライスのnilと空を確認します。
Itemの追加、タイトル、BulkActions、UnknownControlsの不一致と、値が一致する場合も検証します。
既存の共通ケースでは、空入力、順序付き入力、入れ子の不一致、nilと空、順序変更を3案で確認します。

標準案は、外部ライブラリをimportしないファイルだけでも実行できます。
実際のParseによるResearchの検証も含みます。

```sh
go test ./internal/testingcomparison/standard_test.go -count=1 -v
```

このコマンドは `go-cmp` を使う比較用ファイルや既存のパッケージテストを実行しません。
モジュール全体から `go-cmp` を削除したことを示すものではありません。

改善前のメッセージで追加テストを実行し、入れ子の値と位置が表示されず失敗することを確認しました。
表示を補った後、同じテストは成功しました。
ローカルLinuxでの実測であり、性能や保守工数、macOS・Windowsでの実行は検証していません。

## 推奨する使い分け

### まず標準ライブラリで改善する

再調査では、標準の `testing` と `reflect.DeepEqual` を基本にする案を推奨します。
Goの基本的な条件分岐と失敗メッセージで対応でき、外部APIの学習や依存管理を増やさずに済みます。
今回の改善案は、失敗時の表示だけを変え、全体比較の厳密さを維持します。

| 状況 | 判断 |
| --- | --- |
| 入れ子の値がアドレスだけになり読みにくい | その値を直接表示する小さな改善を試す |
| 標準の表示が長く、失敗原因の調査が難しい | `go-cmp` の差分表示と依存管理の負担を再比較する |
| 単純なスライス・マップを比較したい | nilと空の意味を確認し、標準の比較関数を選ぶ |
| 条件のポーリングが必要になる | 標準での同期方法を確認し、必要なら待機ライブラリを比較する |
| プロセス内のgoroutineやタイマーを検証したい | `testing/synctest` を検討する |

標準で改善するという判断は、`go-cmp` の危険性や保守停止を示すものではありません。
Star数やリリース間隔だけで安全性は判断できません。
テスト専用でもコードは開発端末やCIで動くため、依存の追加には確認と更新の負担があります。
今回得たい診断の改善に、その負担を加える必要があるかで判断します。

### 比較ブランチと本実装を分ける

このブランチは比較用のスパイク（仮実装）です。
既存テストの `cmp.Diff` と `go.mod` の依存は、比較対象として残しています。
この再調査では、アプリケーションの動作、CI、既存テスト全体の比較方法は変更しません。
本実装では標準案を必要なテストへ適用し、`go-cmp` の利用がなくなった段階で依存を削除する案になります。
実験専用の `internal/testingcomparison/` と `comparison_spike_test.go` は本実装に取り込みません。

参考にした入門記事: [Goのテストをはじめてみよう（2025年版）](https://future-architect.github.io/articles/20250509a/)。
この記事は標準の比較例と `go-cmp` の両方を紹介しています。
今回の推奨は、記事だけで決めず、上記の比較実験に基づいています。
