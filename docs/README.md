# ドキュメント

renodash の使い方、仕様、開発計画、調査記録をまとめています。初めて使う場合は[使い方](usage.md)から確認してください。

## 使う・開発する

- [使い方](usage.md)：インストール、基本操作、エラーの確認、バージョン表示
- [開発ガイド](development.md)：開発環境、テスト、バージョン情報
- [JSON と解析の仕様](output-format.md)：出力形式、解析対象、エージェント向けの調査情報
- [リリースガイド](release.md)：公開手順、配布物、PR とリリースノートの分類
- [ロードマップ](ROADMAP.md)：実装済みの機能と今後の検討事項
- [テストデータの説明](../testdata/README.md)：スナップショットの出典と派生データの作成方法

## エージェントで調査する

- [エージェントによるセキュリティ調査](agent-security-triage.md)：調査プロンプトと結果の期待形式
- [Dashboard 読み取りスキルのサンプル](../examples/skills/renodash-dashboard/SKILL.md)：`get` の結果と出所を報告するスキル

サンプルを使う際は、対応するエージェントのスキルディレクトリへコピーしてください。例えば `.agents/skills/renodash-dashboard/SKILL.md` に配置します。

## PoC と調査の引き継ぎ

以下は調査時点の記録です。保存済みの Dashboard 本文と JSON は当時の状態を保持しているため、現在の CLI 出力や上流の最新情報とは異なる場合があります。

- [更新の保留理由と公開日時を調べる PoC](minimum-release-age-triage/README.md)
- [dotfiles の調査引き継ぎ](handoffs/renodash-handoff.md)
- [maestro-playground の調査引き継ぎ](handoffs/maestro-playground-handoff.md)
