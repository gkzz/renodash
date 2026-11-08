# リリース

PR の自動ラベル付けは `pull_request` イベントで実行し、同一リポジトリのブランチから作成した PR だけを対象にします。fork からの PR はラベル付けジョブをスキップします。ジョブの権限は `contents: read` と `pull-requests: write` に限定しています。同一リポジトリの PR では、[Label PR の成功実行](https://github.com/gkzz/renodash/actions/runs/42447391752)を確認済みです。Renovate の PR での動作は未確認です。

`.github/workflows/release.yml` は `v0.1.0` のようなタグの push または Actions の GUI からの手動実行を契機に、`make check`、配布用ビルド、GitHub Release の公開を実行します。`.github/release.yml` は、自動生成するリリースノートを変更種別ごとに分類します。分類に当てはまらない PR は `Other Changes` に掲載します。

PR のタイトルは `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `ci:`, `chore:` のいずれかで始めます。必要なら `fix(parser):` のように対象を括弧で示せます。`.github/workflows/labeler.yml` は GitHub 公式の `actions/labeler` を使い、`.github/labeler.yml` のルールに従って作業ブランチ名から `Type: ...` ラベルを付けます。`feat/`, `fix/`, `docs/`, `refactor/`, `test/`, `ci/`, `chore/` と `renovate/` が対象です。`chore/deps` と `build/deps` は依存更新として扱います。ブランチ名が対応しない PR は `Other Changes` に掲載します。Breaking Change とセキュリティ修正のラベルは内容を確認して手動で付けます。リリースノートでは Breaking Changes、Security Fixes の順に優先します。

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

