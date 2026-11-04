# fuchu-kids-event-curator

府中市と近郊の子供向けイベントを集め、ご家庭向けの候補一覧をまとめるプロジェクトです。
HTMLを共通の原稿として、配布用PDF・PNGを出力する予定です。

現在は初期セットアップ段階です。イベント収集・レポート生成はまだ実装していません。

## 開発環境

[mise](https://mise.jdx.dev/)を導入してから、リポジトリで次を実行します。

```sh
mise trust
mise install
mise exec -- go version
```

Goのバージョンはmise.tomlで固定しています。Go本体はmiseのユーザー用管理ディレクトリに保存され、システムへの直接インストールやグローバル設定の変更は不要です。
シェルへのmiseの有効化も必須ではありません。ライブラリの依存関係はGo Modulesで管理します。
現時点では標準ライブラリのみを使用するため、go.sumはありません。

```sh
# CLIのヘルプ
mise exec -- go run ./cmd/fuchu-kids-event-curator -help

# 整形
mise run fmt

# 整形チェック・静的解析・テスト・ビルド（CIと共通）
mise run check
```

現在は機能テストがまだないため、go testはパッケージのコンパイル確認になります。
機能を追加する段階で、その振る舞いを確認するテストも追加します。
CIのGoバージョンもmise.tomlを参照します。実際のCI実行はGitHubへpushした後に確認します。

## 開発の進め方

[GitHub Issues](https://github.com/oddgai/fuchu-kids-event-curator/issues)を1件ずつ進めます。
取得方法の調査、1サイトでの収集、候補の整理、HTML、PDF・PNG、情報源の追加、定期実行の順に整備します。

- [設計書](docs/specs/2026-11-04-event-curator-design.md)
- [開発方針](AGENTS.md)

## Issueの書き方

タイトルは先頭の連番なし、本文は「背景」「実現したいこと」「完了条件」に統一します。
完了条件は確認可能な結果をチェックボックスで記載します。

リポジトリ専用の [write-issue skill](.agents/skills/write-issue/SKILL.md) とGitHubのIssueテンプレートを用意しています。
Codexでは `$write-issue` で指定できます。既存の汎用skillsは複製せず、必要なプロジェクト固有の規約を追加します。

PRは [create-pr skill](.agents/skills/create-pr/SKILL.md) に沿って、背景・変更内容・確認方法を日本語で記載します。`$create-pr` でも指定できます。
