# イベント収集・配布 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Issue #2〜#8を、検証できる収集CLIと配布成果物として実装する。

**Architecture:** ソース別HTTP取得を共通Eventに変換し、期間・参加条件・重複を整理してJSONとHTMLへ保存する。ChromiumでPDFと分割PNGを出力する。失敗した情報源は明示し、全件失敗は非0終了にする。

**Tech Stack:** Go + mise、goquery、chromedp、GitHub Actions。

**Spec:** docs/specs/2026-11-04-event-curator-design.md

## Global Constraints

- 入力PDF・画像のOCRは対象外。
- 未記載の日時や年齢条件を確定情報として補わない。
- Goはmise管理。Issue本文をリポジトリに複製しない。
- ユーザー不在のため可逆的な判断は仮置きし、確認事項を記録する。

## Review Focus

- 0件とHTML構造変化・通信失敗を区別する。
- 開催日と掲載日・申込締切を混同しない。
- 年齢不明や画像のみの詳細は要確認として残す。
- 連続開催・別日開催・複数会場の重複除去で誤って候補を消さない。
- 空の一覧、大量の候補、長い日本語、出典リンク、部分失敗を出力できる。

## 作業とインターフェース

- [x] #2: 7サイトを調査し docs/specs/source-research.md に記録する。
- [x] #3: internal/event のEvent/DateRange/SourceResult/Reportを共通形式とする。internal/collectで取得・解析、保存fixtureによるテストを先に書く。
- [x] #4: event.Select(events []Event, from, until string, ageMonths int) []Event を作る。期間重複、年齢判定、出典統合をテストする。
- [x] #5: internal/report.WriteHTML(io.Writer, event.Report) error を作る。日時・不明値・出典・ソース別失敗・HTMLエスケープを検証する。
- [x] #6: report.Export(ctx context.Context, htmlPath, outDir, browserPath string, perPage int) error でPDF・分割PNGを作る。実ブラウザで0件・少数・多数を検証する。
- [ ] #7: 子Issue #10〜#15を追加。museum/park/theater/kankouを実装・実サイト確認済み。ルミエールは公式公開APIで取得経路を追加。#15 まちづくり府中の出典リンク条件は未解決。
- [x] #8: CLI、設定、手動/定期Actions、artifact保存、失敗判定を整える。
- [x] 全体チェック・実サイト収集・成果物確認・レビュー後、PR #16で提出。ユーザーからのマージ依頼はないためマージしない。

## 仮置きの判断

- 年齢は設定で32ヶ月。生年月日と住所は保存しない。実行時に上書き可能にする。
- 掲載期間は実行日から30日（開始日を含む）。年齢不明と日付未確定は別途要確認として表示する。
- 定期実行は月2回（1日・15日、日本時間朝）。14日間隔とは異なることをREADMEに明記する。
- 交通所要時間は推測せず会場・原文のアクセス案内を確認する。
- AI APIや有料サービスは導入しない。興味の判定は根拠のある簡単なキーワードで補助し、年齢条件とは別に表示する。
- 保存先はActions artifact、保持期間30日。公開サイトへのデプロイは行わない。

## 実施記録

- 2026-11-05: mainへマージされた初期セットアップからfeat/event-curatorを作成。
- 初回調査では7サイトすべてHTTP 200でHTMLを取得。詳細や一覧構造の検証は継続中。

- 2026-11-05: 7サイトのHTTP・robots・詳細構造を調査し、観光協会の月切替/詳細をChromeで検証。調査結果はdocs/specs/source-research.md。
- 選別・パーサー・HTML・CLIのテストを先に書き、失敗から成功への変化を確認。時刻を日付と誤認するケース、年跨ぎ、別企画の年齢条件、休館日注記、中止タイトルの回帰を追加。
- 実サイト4情報源で収集からPDF/PNGまで実行後、観光協会を含めた全設定で35件取得・26候補・PNG6枚を生成。上限による部分取得、ルミエールの失敗もレポートへ表示。
- 実Chromeで0件/1件/13件のPDF/PNGテスト成功。PDFの日本語と出典URI注釈を確認。
- ワークフローの引数引用不具合をレビューで発見して修正。YAMLのrun文字列を取り出し、架空サンプルで実行して検証。
- #8のチェックは実装とローカル検証の完了を示す。GitHubの手動/定期実行そのものはデフォルトブランチへのマージ後に確認が必要。

## 後日ユーザーに確認すること

1. 月齢32ヶ月・掲載30日・月2回（1日/15日）・artifact30日保存でよいか。
2. お子さんの興味と、日程/年齢不明の候補をどの程度表示したいか。現状は取りこぼしを避けて要確認に残す。
3. 博物館のカテゴリ内をさらに巡回する必要があるか。現在は静的トップから直接辿れる個別記事のみ。
4. まちづくり府中の出典リンク条件の扱い（#15）。ルミエールは公開APIで取得するが、画像のみの記事は要確認として残す。
5. 所要時間や移動手段の優先順位。現状は原文の会場だけ表示し、経路や移動時間は推測しない。

- GitHub Ubuntu最新版ではAppArmorの制約でChrome sandboxが起動できずCI失敗を確認。ブラウザsandboxを無効化せず、user namespace制約の異なるUbuntu 22.04 runnerに固定して再検証する。

- 最終版ではルミエールを含む6情報源から39件取得・29候補・PNG6枚を生成。既定設定にまちづくり府中のIDも含め、自動出力の保留理由をレポートへ明示する。
