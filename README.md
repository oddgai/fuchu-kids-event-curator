# fuchu-kids-event-curator

府中市と近郊のイベントを集め、ご家庭向けの候補一覧をJSON・HTML・PDF・PNGで保存するGo製CLIです。
年齢が明示された参加条件と、原文キーワードによる興味の補助情報を分けて表示します。条件不明の候補は要確認として残します。

## ルールによるピックアップ

AI APIは使いません。参加条件と内容の好みを確認し、HTML・PDF・PNGを次の3枠に分けます。

- おすすめイベント：内容が好みに合い、開催日が対象期間内と確認できる候補。
- 期間限定のお出かけ情報：イルミネーション等の開催期間を簡潔に表示。
- 要確認の候補：日程不明、集合案内、内容や参加条件の根拠が不足するもの。

除外した情報は `report.json` の `excluded` に理由とともに残します。過去の開催、明示的な中止・満員・対象外年齢を、情報源や「親子向け」の名称より優先して判定します。
プレーパーク、幼児歓迎、ファミリーコンサート、子供用遊具・乗車体験等を肯定材料とし、飲食中心・落語・清掃活動等は現在の好みでは優先しません。
昼間を優先し、18時以降開始は原則除外する仮のルールです。ユーザーがOKとした夜のプレーパークは例外として時間の確認を促します。

`feedback/reviews.json` に29件の評価と人手の根拠文を保存しています。実行時はOK/NGラベルで上書きせず、同じURL・同じタイトルの確認済み根拠文だけを補足します。
チラシ画像の年齢条件もこの方法で補えますが、OCRは行いません。取得日が確認日から90日を超えた根拠は再適用しない仮の期限を設けています。
本文中の別企画の年齢条件を親イベントへ混ぜないため、内容判定用本文と全体参加条件の文章を分けて保存します。

```sh
# 人手の根拠を補足して再判定し、内容評価との比較も保存
mise exec -- go run ./cmd/fuchu-kids-event-curator \
  -input output/collected.json -from 2026-11-05 \
  -feedback feedback/reviews.json -evaluate-feedback -output output/review

# 人手の補足だけを使わない（一般ルールは有効）
mise exec -- go run ./cmd/fuchu-kids-event-curator -feedback ''
```

`evaluation.json` は評価時の月齢で内容の判断を比較します。満員や過去の日程へのOK評価は、掲載可能という意味にはしません。
同じ29件に対する回帰比較であり、新しいイベントに対する精度の測定ではありません。別タイトル・未知URL・制限付きの肯定例もテストしています。
以前の収集JSONには本文が保存されていないため、十分な根拠がなければ要確認に回します。本文を使う判定には再収集してください。

## 準備

[mise](https://mise.jdx.dev/)を導入してから実行します。GoとGitleaksはmise.tomlに固定され、グローバル設定の変更は不要です。

```sh
mise trust
mise install
mise exec -- go version
```

PDF・PNGの出力と観光協会の動的ページ取得にはChromeまたはChromiumが必要です。日本語フォント（LinuxではNoto CJK等）も用意してください。
Macではインストール済みGoogle Chromeを利用できます。Chromeを見つけられない場合は `-browser` または `CHROME_PATH` で実行ファイルを指定します。

## 収集して一覧を生成する

```sh
# JSON・HTMLを生成（既定値：今日から30日、月齢32ヶ月）
mise exec -- go run ./cmd/fuchu-kids-event-curator -config config.example.json

# PDF・分割PNGも生成
mise exec -- go run ./cmd/fuchu-kids-event-curator -config config.example.json -export

# 期間・月齢・情報源を指定
mise exec -- go run ./cmd/fuchu-kids-event-curator \
  -from 2026-11-05 -days 30 -age-months 32 \
  -sources seiseki,museum,park,theater -output output/november -export
```

`config.example.json`をコピーして編集できます。住所・生年月日は不要です。月齢は自動加算しないため、成長に合わせて変更してください。
`-config`を省略すると組み込みの既定値を使い、引数は設定ファイルより優先されます。
全オプションは `-help` で表示します。対象期間の終了日は「開始日＋日数−1日」です。

出力ディレクトリに次を保存します（同じディレクトリへの再実行は上書き）。

| ファイル | 内容 |
| --- | --- |
| collected.json | 絞り込み前の取得結果・情報源ごとの状態 |
| report.json | おすすめ・期間情報・要確認・除外と判定理由 |
| report.html | 外部CSSや画像なしで読める一覧 |
| report.pdf | 印刷・共有用PDF（`-export`時） |
| report-001.png 以降 | 既定5件ずつに分割した画像（`-export`時） |

画像の件数は `-per-page` で変更できます。PDFには全候補を掲載し、出典リンクを保持します。
別の実行でPDF・画像を再生成しない場合、以前のPDF・画像は更新されません。混同を避けるには出力先を分けてください。

## 通信せずに再生成する

```sh
# 保存済みの収集結果に別の条件を適用
mise exec -- go run ./cmd/fuchu-kids-event-curator \
  -input output/collected.json -from 2026-11-05 -days 30 -output output/review -export

# 実在しないサンプルで出力確認
mise exec -- go run ./cmd/fuchu-kids-event-curator \
  -input examples/events.json -from 2026-11-01 -output output/sample -export
```

## 情報源と制約

| ID | 情報源 | 現在の扱い |
| --- | --- | --- |
| seiseki | 聖蹟桜ヶ丘 | HTTP一覧・詳細、ページ送り |
| museum | 郷土の森博物館 | 静的なイベント案内から辿れる個別ページ。禁止されたカレンダーCGIは使わない |
| park | 府中市の公園 | お知らせ一覧・詳細、ページ送り。年が明示されない記事は日程未確定 |
| theater | 府中の森芸術劇場 | 静的ページで紹介される公演詳細。カレンダーCGIは使わない |
| kankou | 府中観光協会 | Chromeで対象月の一覧と詳細を読み込む |
| lumiere | ルミエール府中 | 公式公開APIの公開イベント記事。画像のみの条件・開催年は補完しない |
| machidukuri | まちづくり府中 | 規約のリンク事前承諾条件を確認するまで自動出力を保留 |

既定では1情報源につき一覧2ページ・詳細30件まで、同じホストのHTTP取得間隔は1秒です。
上限に達した場合は一部取得と表示します。網羅的なイベント一覧ではありません。博物館はトップの案内から直接辿れる範囲で、カテゴリ内のすべての記事を巡回しません。
詳細条件が画像・PDFだけにある場合は抽出しません。まちづくり府中の配布可否は後日確認事項です。

開催年がない公園・ルミエールの告知は、ページ日付を基準に推定し、その根拠を注記します。基準日も分からない告知は要確認に残し、現在年やURLからは補いません。
複数企画を含むページの各企画の年齢・料金をイベント全体の条件にはしません。例外を含む年齢条件や学校段階は原文確認に回します。
単純な年齢範囲のみ自動判定します。中止記事と、雨天時のみ中止という条件は区別します。

情報源の取得失敗や保留はHTML・JSONに表示します。一部が取得できれば成果物を残して終了コード0、全件失敗・全件無効なら状況付き成果物を保存して終了コード1です。
キャッシュによる過去データの代用は行いません。収集後の変更・中止・予約状況は出発前に公式情報を確認してください。

## 定期実行・ダウンロード

GitHub Actionsの「イベント一覧を生成する」を使います。

- 手動：Actions → 対象ワークフロー → Run workflow。掲載日数と月齢を指定可能。
- 定期：毎月1日・15日の09:17 JST。月2回であり、14日間隔ではありません。Actionsの混雑などで遅れる場合があります。
- 保存：実行ページのArtifactsから `fuchu-kids-events-実行番号` をダウンロード。保持期間30日。
- 全件失敗時も、生成できた状況レポートをartifactへ保存します。

定期実行はデフォルトブランチにマージ後に有効になります。公開Webサイトへの配信やメール送信は行いません。
月齢32ヶ月・掲載30日・月2回・保存30日は仮置きで、設定とワークフローで変更できます。

## 開発・検証

```sh
mise run fmt
mise run check

# git addした変更をコミット前に検査
mise run secrets:staged
# 取得済みの全ブランチ・タグのGit履歴を検査
mise run secrets

# Chromeを使う0件・1件・13件のPDF/PNG統合テスト
REPORT_TEST_BROWSER='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' \
  mise exec -- go test ./internal/report -run TestExportBrowser -v
```

通常のテストは保存HTML・疑似HTTPサーバー・オフラインJSONを使い、外部サイトへ接続しません。
ブラウザ統合テストは `REPORT_TEST_BROWSER` 指定時に実行し、CIでも日本語フォントを用意して実行します。
実サイトの構造が変わった場合は、情報源別のエラーと [調査書](docs/specs/source-research.md) を照合します。

秘密情報の検査には[Gitleaks](https://github.com/gitleaks/gitleaks)の標準ルールを使います。CIの独立したGitleaksジョブは全Git履歴を取得して検査し、検出時に失敗します。検出値はログでマスクします。`secrets:staged` はステージ済み変更だけ、`secrets` はコミット済み履歴だけを対象とするため、変更は `git add` してからコミット前の検査を実行してください。Gitフックの自動設定は行いません。

[GitHub Issues](https://github.com/oddgai/fuchu-kids-event-curator/issues)を1件ずつ進めます。
[設計書](docs/specs/2026-11-04-event-curator-design.md)、[実装計画・仮置き判断](docs/plans/2026-11-05-event-curator.md)、[開発方針](AGENTS.md)も参照してください。
Issue本文は背景・実現したいこと・完了条件、PR本文は背景・変更内容・確認方法に統一します。
[write-issue](.agents/skills/write-issue/SKILL.md)・[create-pr](.agents/skills/create-pr/SKILL.md)を用意しています。

## 月表示カレンダー

HTMLは月曜始まりの月表示で、前後月のボタンとイベントのタップで詳細を確認できます。要確認候補は下の別枠に置き、未確定の日付でカレンダーを埋めません。不明な会場・料金・時間・対象年齢等は欄ごと省略します。
複数日開催は横長のバーでつなぎ、週・月の境界で分割して表示します。日程が飛び飛びの場合は隙間をつながず、重なるイベントは上下に並べます。どの区間からも同じ詳細を開けます。
タイトルに年が明記されていれば開催日欄の月日と照合します。年のない公園・ルミエールの単日告知は、ページ更新日以降で最初に来る開催月日の年を仮置きし、「仮：」を付けて表示します。例えば2026/10/30更新の11/14は2026/11/14、2026/12/20更新の2/11は2027/2/11です。更新日当日も対象に含みます。ルミエールはAPIの更新日時、公園は更新日時のメタ情報またはページ表示日付を使います。曜日が記載されていて一致しない場合や根拠が足りないものは要確認に残します。推定の根拠は詳細とGoogle登録下書きにも残します。
詳細のGoogleカレンダー登録リンクは予定の下書きを開くもので、自動保存しません。明確な開始〜終了がある単日開催だけ時刻付き、その他は終日で開きます。複数日・休館日・複数回公演の時間は保存前に確認してください。日付不明の候補にリンクは出しません。
PDF・PNGはクリックなしでも読める詳細一覧として保存します。JavaScriptが無効でも詳細一覧を閲覧できます。
