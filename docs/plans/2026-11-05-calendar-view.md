# 月表示 Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan.

**Goal:** 月表示、タップ詳細、Googleカレンダー下書きリンク、不明項目省略。
**Architecture:** Goで月グリッドと詳細を生成し、ハッシュ付きCSPで許可した最小JSで切替とdialogを操作する。印刷時は詳細一覧を表示。
**Spec:** docs/specs/calendar-view.md

- [x] 月曜始まりと月境界、日程ごとのリンクをテストする。
- [x] 日付不明を推測せず、時刻範囲と複数回公演を区別する。
- [x] 不明項目を省略し、詳細ダイアログと月切替を実装する。
- [x] JSハッシュと実ブラウザのクリック、PDF/PNGを確認する。
- [x] 要確認を別枠にし、収集状況は折りたたんでカレンダーを先に読めるようにする。
- [x] PRに提出する（#18に依存）。CI結果はPRで確認する。

レビューで午後表記と複数回公演の一部だけを時刻確定する問題を発見。曖昧な時刻を終日に戻す回帰テストで修正。
