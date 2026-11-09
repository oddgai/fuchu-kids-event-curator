# 開発方針

- ドキュメントとIssueは日本語で記述する。
- 計画書は `docs/plans/`、設計書は `docs/specs/` に置く。Issue本文はGitHubで管理し、リポジトリ内に複製しない。
- Issue単位で作業し、現在の範囲と完了条件を確認する。
- Issueを起票・編集するときは [.agents/skills/write-issue/SKILL.md](.agents/skills/write-issue/SKILL.md) を読む。タイトルに先頭の連番を付けず、本文は「背景」「実現したいこと」「完了条件」に統一する。
- PRを準備・作成・編集するときは [.agents/skills/create-pr/SKILL.md](.agents/skills/create-pr/SKILL.md) を読む。タイトル・本文は日本語で、背景・変更内容・確認方法を記載する。
- Go本体はmiseでプロジェクト単位に管理する。システムへの直接インストールやグローバル設定の変更を前提にしない。
- Goの変更後は `mise run check` で整形・静的解析・テスト・ビルドを確認する。整形は `mise run fmt` を使う。
- 秘密情報の検査には `mise run secrets`（全Git履歴）と `mise run secrets:staged`（ステージ済み変更）を使う。コミット前にはステージ済み変更を検査し、検出値をマスクする設定を維持する。
- 入力PDF・画像のOCRは現在の対象外。HTMLから配布用PDF・PNGを生成することが要件。
- 外部サイトからの取得では出典を保持し、未記載の日時・参加条件を確定情報として補わない。
