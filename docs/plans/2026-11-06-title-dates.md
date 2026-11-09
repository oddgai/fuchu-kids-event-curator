# タイトルの日付抽出 Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan.

**Goal:** タイトルの日付を利用し、開催年の根拠と推定を明示してカレンダーへ表示する。
**Architecture:** `collect/parse.go` でタイトルの明示日程と開催日欄を照合する。`collect/date_estimate.go` は掲載日を根拠にした年の仮置きのみを担当する。`event.Event.DateNote` で根拠を保存し、HTML・登録リンクに伝える。
**Spec:** docs/specs/title-dates.md

- [x] `collect/parse_test.go` で日程欄とタイトルの年、範囲の維持、年跨ぎ、曖昧な年を再現する。修正前に失敗することを確認する。
- [x] タイトルに明記された開催年を、包含関係と一意性を確認して利用する。
- [x] 掲載日からの年推定を追加する。現在年・掲載月日の流用・曜日不一致を防ぐテストを行う。
- [x] 年の推定根拠をJSONへ保存し、カレンダー・詳細・Google下書きに表示する。
- [x] 対象期間内の仮表示と期間外の除外を統合テストで確認する。
- [x] 全チェック・ブラウザテスト・実サイトの再収集・レビューを行う。レビューで見つかった申込日誤認と期間情報の配布用注記も修正。
- [x] #20をbaseに提出する準備を完了する。CI結果はPRで確認する。
