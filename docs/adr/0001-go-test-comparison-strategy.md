# ADR 0001: Go テストの比較方針

| 項目 | 内容 |
| --- | --- |
| 状態 | Accepted（承認済み） |
| 日付 | 2026-11-22 |
| 承認日 | 2026-11-23 |
| 承認記録 | [PR #100 の Notes](https://github.com/gkzz/renodash/pull/100) |

## 決定: testing と go-cmp を併用する

2026-11-23、メンテナーが標準 `testing` と `go-cmp` の併用を承認した。
このADRの意思決定は完了。PRのレビューとマージは別途行う。

テストの実行には、標準の `testing` を使う。
構造体の比較には、`go-cmp` を使う。
数値や文字列などは、`==` や `!=` で比較する。

目的は、失敗した箇所を見つけやすくすること。
既存テストの期待値と、比較の厳密さは維持する。

## 背景: ネストした値の違いを追いにくい

基準コミット `848ea5c` では、`reflect.DeepEqual` で構造体を比較していた。
期待値と実際の値は、`%#v` で表示していた。
この表示では、Dashboard 内の Research がポインタのアドレスになる。
どのフィールドが違うのか、失敗メッセージから追いにくい。

[PR #99](https://github.com/gkzz/renodash/pull/99) では、比較方法を調査した。
標準パッケージだけでも、Research の中身は表示できる。
ただし、型ごとに表示処理を書く必要がある。
型が増えれば、その処理も保守することになる。

`go-cmp` なら、ネストした値の差分を表示できる。
依存の更新は、既存の Renovate Dependency Dashboard で追う。

## 判断で重視したこと

| 重視すること | 方針 |
| --- | --- |
| 失敗箇所の把握 | フィールドごとの差分を表示する |
| 比較の厳密さ | nil と空、スライスの順序を区別する |
| 保守の負担 | 型ごとの表示処理を減らす |
| 依存の管理 | Renovate の Go モジュール更新ルールを有効にする |
| 既存テストとの整合 | 標準のテスト構成とオフライン実行を維持する |

## 比較方法の使い分け

| 比較するもの | 使う方法 |
| --- | --- |
| Dashboard、Research、依存情報 | `cmp.Diff(want, got)` |
| 保存済み JSON のメタデータ | `cmp.Diff(want, got)` |
| コマンド呼び出しの内容と順序 | `cmp.Diff(want, got)` |
| 数値、文字列、真偽値 | `==`、`!=` |
| fake gh の引数判定 | `slices.Equal` |
| nil と空を同じとみなす単純な比較 | `slices.Equal`、`maps.Equal` |

`want` は期待値、`got` は実際の値を表す。
差分が空でなければ、テストを失敗にする。

```go
if diff := cmp.Diff(want, got); diff != "" {
    t.Errorf("dashboard mismatch (-want +got):\n%s", diff)
}
```

`-` は期待値、`+` は実際の値を表す。
構造体全体を比較対象にして、フィールドの検証漏れを減らす。
失敗後も続ける場合は、`t.Errorf` を使う。
そこで止める場合は、`t.Fatalf` を使う。

ただし、`cmp.Diff` は型の `Equal` メソッドを利用する場合がある。
そのメソッドで一部のフィールドを無視すると、`reflect.DeepEqual` では不一致だった値も一致する可能性がある。
既存テストを移行する際は、ネストした型も含めて `Equal` メソッドの有無と、合否判定への影響を確認する。
詳しくは、[go-cmp の比較規則](https://pkg.go.dev/github.com/google/go-cmp/cmp#Equal)を参照する。

## メモ: 標準版と go-cmp で失敗表示が異なる理由

既存の `dashboard.Parse` を対象に、同じ入力・期待値・比較範囲で `reflect.DeepEqual` と `cmp.Diff` を比較した。
どちらも標準の `testing` を使い、Dashboard 全体が一致するかを自動判定する。
合否を目視で判断する必要はない。

標準版でも期待値（`want`）と実際の値（`got`）を両方出力している。
ただし、次の `%#v` 表示では、Dashboard 内の `Research` が参照先の内容ではなくポインタのアドレスになる。
これは `testing` の制限ではなく、この表示方法によるもの。

```go
if !reflect.DeepEqual(want, got) {
    t.Fatalf("dashboard mismatch\nwant: %#v\ngot:  %#v", want, got)
}
```

次は出力の抜粋。アドレスは実行ごとに変わる。

```text
want: ... Research:(*dashboard.Research)(0x21f24484a0d0) ...
got:  ... Research:(*dashboard.Research)(0x21f24484a000) ...
```

`reflect.DeepEqual` 自体は参照先まで比較して不一致を検出する。
ただし、この失敗表示だけでは、どのフィールドが違うかはわからない。
なお、ポインタのアドレスが異なるだけでは不一致にならず、参照先の値が一致していれば比較は成功する。

`cmp.Diff` は参照先の構造までたどり、異なるフィールドと値を差分として表示する。
同じテストで `CurrentVersion` を意図的に変えると、次の差分が出る（抜粋）。

```go
if diff := cmp.Diff(want, got); diff != "" {
    t.Fatalf("dashboard mismatch (-want +got):\n%s", diff)
}
```

```diff
- CurrentVersion: "12.13.0",
+ CurrentVersion: "12.12.0",
```

標準パッケージでも、表示対象を直接指定すれば値を出せる。

```go
t.Logf("want: %s, got: %s",
    want.Items[0].Research.DetectedDependencies[0].CurrentVersion,
    got.Items[0].Research.DetectedDependencies[0].CurrentVersion,
)
```

この方法では、表示するフィールドを自分で選ぶ必要がある。
`go-cmp` を使うと、構造全体の比較と、不一致だったフィールドの差分表示をまとめて行える。
今回の型とオプションなしの比較では、nil と空、スライス順序の区別も維持できる。

サンプルはローカルの `spike/go-cmp-usecase` のコミット `767026c`、`internal/dashboard/comparison_example_test.go` にある。
通常は両方成功し、共通の準備処理にあるバージョン変更のコメントを外すと両方失敗する。
サンプルを実行し、通常の成功と、意図的な不一致時の差分を確認した。

```sh
go test ./internal/dashboard -run '^TestParseComparison' -v
```

## 比較オプションはテストごとに選ぶ

まず、オプションなしで比較する。
Dashboard と Research の既存テストでは、追加のオプションは不要。

| 判断したいこと | 方針 |
| --- | --- |
| nil と空を区別するか | 原則として区別する |
| 要素の順序を区別するか | 原則として区別する |
| nil と空を同じとみなす仕様か | `cmpopts.EquateEmpty` を検討する |
| 順序を問わない仕様か | ソートするオプションを検討する |
| 検証対象外のフィールドがあるか | 除外する理由を明記する |
| 非公開フィールドを比較したいか | まず、返り値や外から見える振る舞いで検証する |

オプションで比較条件を変えるときは、その理由をテスト内に書く。
条件の異なるテストへ、一括で同じオプションを適用しない。

非公開フィールドは、既定の設定ではパニックになる場合がある。
比較が必要なら、型を限定した比較関数や `cmp.AllowUnexported` を検討する。
比較のためだけに、フィールドを公開しない。
安易な除外は、不具合の見落としにつながる。

## 検討した代替案

| 案 | 利点 | 今回の判断 |
| --- | --- | --- |
| 標準パッケージだけを使う | 依存が増えない | ネストした値の表示処理が必要 |
| testing と go-cmp | フィールドの差分が読める | 採用する。依存とオプションの管理は必要 |
| testing と Testify / Gomega | アサーションや待機処理を追加できる | 現在のテストでは必要性がない |
| Ginkgo と Gomega | シナリオを階層化できる | 現在のテーブル駆動テストで足りる |

`testing/synctest` は、ゴルーチンや仮想時間を検証するための機能。
外部プロセスやループバック通信を使う既存テストの代わりにはならない。

## 変更する範囲

`github.com/google/go-cmp/cmp` をテスト用の依存として追加する。
既存の構造比較を置き換える。
CLI の動作と、出力の期待値ファイルは維持する。
本番コードには、外部モジュールの依存を追加しない。

共有のアサーション基盤や、独自の差分エンジンは作らない。
調査専用パッケージと、内部状態の検証実験は取り込まない。
テストデータの構造体リテラルにフィールド名を付ける作業は、[Issue #101](https://github.com/gkzz/renodash/issues/101) で扱う。

## 採用後の効果と見直し条件

| 項目 | 効果と負担 |
| --- | --- |
| 失敗メッセージ | ネストしたフィールドの差分が見える |
| 保守 | 型ごとの表示処理を減らせる |
| テストの流れ | 標準 testing のまま実行できる |
| 依存 | 開発環境と CI で実行する依存が増える |
| 比較条件 | オプションが仕様に合うか、レビューが必要になる |

オプションの保守が難しくなったら、この方針を見直す。
依存の管理負担が、差分表示の利点を上回る場合も見直す。
別のライブラリを追加する前に、具体的な問題と代替案を記録する。
スター数やリリース間隔だけで、安全性や保守品質は判断しない。

## 根拠と関連資料

調査では、ネストした値、nil と空、要素順、実際のパーサー出力を確認した。
非公開フィールドも、実在する内部型で比較した。
これらは Linux 上での限定的な確認。
性能や、他の OS での動作は検証していない。

調査記録は、`spike/testing-library-comparison` のコミット `540d545` にある。

- [比較調査](https://github.com/gkzz/renodash/blob/540d545/docs/test-libraries.ja.md)
- [標準パッケージだけの検証](https://github.com/gkzz/renodash/blob/540d545/internal/testingcomparison/standard_test.go)
- [非公開フィールドの検証](https://github.com/gkzz/renodash/blob/540d545/internal/dashboard/comparison_spike_test.go)
- [標準 testing](https://pkg.go.dev/testing)
- [go-cmp の比較規則](https://pkg.go.dev/github.com/google/go-cmp/cmp)
- [reflect.DeepEqual](https://pkg.go.dev/reflect#DeepEqual)
- [slices.Equal](https://pkg.go.dev/slices#Equal) / [maps.Equal](https://pkg.go.dev/maps#Equal)
- [テストの書き方](../testing.md)
