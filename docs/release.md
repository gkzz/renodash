# リリース

## 公開手順

変更を PR 経由で main にマージし、最新の main にタグを付けて push します。

```sh
git switch main
git pull --ff-only
git tag -a v0.1.0 -m 'Release v0.1.0'
git push origin v0.1.0
```

タグは `vMAJOR.MINOR.PATCH` 形式に対応し、プレリリースタグは未対応です。CI で対象コミットが main に含まれることを確認します。

手動で公開する場合は、Actions → Release → Run workflow で `main` を選び、`Release tag` に `v0.1.0` などを入力します。実行開始時の main にタグを作成して公開します。main 以外の指定や、別のコミットを指す既存タグの再利用はエラーになります。

[Release ワークフロー](../.github/workflows/release.yml)は `make check` と配布用ビルドを実行し、成果物を GitHub Release のドラフトに添付してから公開します。追加のシークレットは不要です。途中で失敗した場合は同じ実行を再実行してください。手動実行でドラフトを再利用するときは、対象コミットの一致も確認します。公開済みリリースは上書きできず、再実行はエラーになります。

## 配布物とビルドの確認

Linux・macOS・Windows の amd64/arm64 向けに、計6種類のバイナリを配布します。README と LICENSE を同梱し、SHA-256 の `checksums.txt` を添付します。形式は Linux・macOS が `.tar.gz`、Windows が `.zip` です。

PR でも検証と6種類のビルドを行い、Linux バイナリの動作と `version` 出力を確認します。バージョンとコミット SHA がビルド対象に一致することも検証します。

ローカルでビルドを確認するには、Go・tar・zip・sha256sum を用意し、未作成の出力先を指定します。

```sh
bash setup/release-build.sh v0.1.0 /tmp/renodash-dist
```

全バイナリに指定バージョンと `git rev-parse HEAD` の SHA を埋め込みます。未コミットの変更も含まれるため、公開用ビルドは作業ツリーに変更がない状態で実行してください。

## PR とリリースノートの分類

PR タイトルは `feat:`、`fix:`、`docs:`、`refactor:`、`test:`、`ci:`、`chore:` のいずれかで始めます。`fix(parser):` のように対象を添えても構いません。

`actions/github-script` が[分類ルール](../.github/labeler.yml)に従って、ブランチ名から `Type: ...` ラベルを付けます。対象は `feat/`、`fix/`、`docs/`、`refactor/`、`test/`、`ci/`、`chore/`、`renovate/` で、`chore/deps` と `build/deps` は依存更新に分類します。

[リリースノートの設定](../.github/release.yml)で変更種別ごとに分類し、該当しない PR は `Other Changes` に掲載します。Breaking Change とセキュリティ修正のラベルは内容を確認して手動で付けます。掲載順は Breaking Changes、Security Fixes を優先します。

## 自動ラベル付けの構成と検証

ラベル付けは次の2段階で実行します。

- [Label PR Trigger](../.github/workflows/label-pr-trigger.yml)：`pull_request` の `opened`・`reopened`・`synchronize` で起動します。`permissions: {}` とし、PR コードは取得・実行しません。
- [Label PR](../.github/workflows/labeler.yml)：前段の成功を `workflow_run` で受け、デフォルトブランチ上の定義で実行します。ラベル付けジョブだけに `pull-requests: write` と設定取得用の `contents: read` を付与します。

後段では実行元の名前・パス・イベント・成功状態を検証し、関連 PR が1件の場合に API で最新状態を取得します。同一リポジトリの open な PR で、head SHA とブランチが一致する場合だけ処理します。fork、関連付けが空・複数・不一致の実行はスキップし、API 障害時も処理を進めません。PR 番号をブランチ名から推測することもありません。

検証済みの PR 番号と head ブランチ名を使い、GitHub API でラベルを追加します。分類ルールは `.github/labeler.yml` だけで管理し、JSON としても読める YAML 形式で保存します。設定は `workflow_run` の `GITHUB_SHA` から取得します。対応するルールは `head-branch` の正規表現文字列のみで、未対応の形式はエラーにします。既存ラベルは削除しません。PR のコード・artifact・キャッシュは取得しません。

この2段階構成は GitHub 上では未検証です。[以前の成功実行](https://github.com/gkzz/renodash/actions/runs/42447391752)は `pull_request` に直接書き込み権限を付けた構成の結果です。

ローカルでは次のコマンドでワークフロー定義を検査できます。

```sh
actionlint .github/workflows/label-pr-trigger.yml .github/workflows/labeler.yml
```

両ワークフローをデフォルトブランチへマージした後、次を確認します。

1. 同一リポジトリの `ci/` PR を作成・更新し、前段の成功後に `Type: CI` が付く。`renovate/` などの分類も確認する。
2. fork PR、前段が失敗した実行、関連 PR が空の実行ではラベルが変更されない。
3. 前段完了後に PR を閉じるかコミットを追加すると、旧実行の再実行がスキップされる。
4. ログ上で前段に書き込み権限がなく、後段にも checkout・artifact 取得・PR コード実行がない。

状態確認からラベル更新までの間に PR が変わる可能性は残るため、ラベルを認可やデプロイの判定には使いません。また、同一リポジトリへ push できる利用者が、別の PR ワークフローで権限を要求することまでは防げません。
