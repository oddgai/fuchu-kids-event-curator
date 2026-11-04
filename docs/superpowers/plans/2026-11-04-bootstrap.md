# 初期セットアップ Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Issue #1 のGo開発基盤と、統一されたIssueの書き方を用意する。

**Architecture:** miseでGoを固定し、Go標準ツールで開発する。最小CLIはヘルプだけを提供し、収集・出力は後続Issueとする。

**Tech Stack:** Go、mise、Go Modules、GitHub Actions。

**Spec:** docs/superpowers/specs/2026-11-04-event-curator-design.md

## Global Constraints

- Goをシステムに直接インストールせず、miseのグローバル設定を変更しない。
- 入力PDF・画像のOCRは対象外。
- Issueのタイトルは連番なし。本文は背景・実現したいこと・完了条件。

## Review Focus

- シェル有効化なしでもmise execで開発できる。
- CIとローカルで同じGoの指定を使用する。
- ヘルプが収集機能の完成を示唆しない。
- skillの利用がGitHubへの無断書き込みを意味しない。
- 外部依存なしの段階で不要なgo.sumを生成しない。

## Task 1: Issue形式を整える

Files: .agents/skills/write-issue/SKILL.md、AGENTS.md、.github/ISSUE_TEMPLATE/task.md、docs/issues/README.md。

- [x] 既存原稿から起票させ、指定の背景・実現したいことが欠落する基準結果を確認する。
- [x] skill、テンプレート、参照指示を作成する。
- [x] skill適用後の原稿が指定の構成で要件を保持することを確認する。
- [x] GitHubの既存8件を更新する。
- [x] 構造検証とGitHubからの再取得を行う。

## Task 2: Go基盤を用意する

Files: mise.toml、go.mod、cmd/fuchu-kids-event-curator/main.go、README.md、.gitignore、.github/workflows/ci.yml。

- [x] miseで確認したGo 1.27.3を固定し、mise execで導入する。
- [x] CLIは標準flagパッケージのヘルプのみ提供する。設定・骨格に実装を写すだけのテストは追加しない。
- [x] miseタスクでfmt、checkを用意し、CIでは同じcheckを実行する。
- [x] mise exec -- go version、mise run check、CLIの-helpを確認する。
- [x] READMEの実行手順と実物を照合する。

## 実施記録

- 初期コミットがないためworktreeを作る土台がなく、現在の作業ディレクトリで初期ファイルを整備する。
- Issue形式は既存の「目的・範囲」から3見出しへの変更を確認。skill適用後の原稿は3見出しに統一された。
- skill-creatorのquick_validateで成功。検証用PyYAMLは/tmpの一時環境にのみ導入した。
- mise run checkは成功（機能テストなし、コンパイル・静的解析・ビルドの確認）。GitHubのCIは未実行。
