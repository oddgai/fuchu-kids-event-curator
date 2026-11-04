# 情報源調査（Issue #2）

調査日時：2026年11月5日 00:17〜00:30 JST。公式サイトの通常HTTP取得（Python標準HTTPクライアント、リダイレクト追従）と取得HTMLを確認した。観光協会・ルミエールはヘッドレスChromeでJavaScript実行後のDOMも確認した。取得物は一時ディレクトリに保存し、本文・画像の大きな複製をリポジトリへ入れていない。ログイン・アクセス制限回避・フォーム送信は実施していない。

## 判断と共通方針

最初は聖蹟桜ヶ丘を通常HTTPで実装できる。次に公園、博物館・劇場の静的HTMLを検討する。まちづくりは技術面では通常HTTPで収集できるが、利用規約のリンク条件を配布前に整理する。観光協会はJS後の一覧・詳細を確認できたが月操作を未検証、ルミエールは取得経路を確定できず、取得不能を0件として扱わない。

robots.txtで許可対象に見えることと、本文・画像の転載許可は別である。事実の必要最小限と短い参加条件、出典URL、取得日時を保存し、チラシや紹介文を丸ごと配布しない。利用条件は調査時点の確認範囲であり、再利用の包括許諾を確認したという意味ではない。定期実行前にrobotsと条件を再確認する。

日付のない情報や年齢条件は不明のまま保持する。投稿日・更新日と開催日を分け、複数日・複数回開催、イベント内の個別企画の対象年齢、中止・雨天条件を残す。画像・PDFだけの条件はOCR対象外として元ページへの案内に留める。

## 比較

| 情報源 | 通常HTTP | 継続一覧・ページ送り | 開催日時・対象条件 | 当初の扱い |
| --- | --- | --- | --- | --- |
| まちづくり府中 | 200、カレンダー・詳細に内容あり | 年月クエリ指定可 | HTML、日時属性に不正なTZ表記あり | 規約整理後に対応 |
| 府中観光協会 | 200だが空の一覧枠 | JSのカレンダーUIあり、動作未確定 | 初期HTMLにはイベントなし | 未対応・明示的失敗 |
| 府中市市立公園 | 200、一覧・詳細に内容あり | `/news/`、`/news/page/2/` | HTML本文とチラシ混在 | 通常HTTP候補 |
| 聖蹟桜ヶ丘 | 200、一覧・詳細に内容あり | `/event/`、`/event/page/2/` | 日付範囲・自由記述HTML | 初回実装対象 |
| 郷土の森博物館 | 200、静的ページに内容あり | カテゴリ一覧を辿る | `dl` の開催日・時間等 | CGI除外で対応候補 |
| ルミエール府中 | `/welcome/`へ転送 | イベント一覧へ到達できず | 未確認 | 未対応・明示的失敗 |
| 府中の森芸術劇場 | 200、静的ページに内容あり | お勧めラインナップ等 | `dl` と本文の入場条件 | CGI除外で対応候補 |

## 1. まちづくり府中

- 起点：[カレンダー](https://machidukuri-fuchu.jp/calendar/)。取得時は2026年11月のイベントを含む。過去月が返る前回の閲覧結果は今回のHTTP結果では再現しなかった。
- 月指定：[2026年12月](https://machidukuri-fuchu.jp/calendar/?yr=2026&month=12&dy=1) は200、`td#calendar-2026-12-*` と12月表示に変化した。対象期間をまたぐ月を明示取得し、ページの表示月を検証する。
- 一覧の `a[href*="/mc-events/"]` から詳細へ。同一イベントが複数日に出るのでURLだけで開催回を消さない。
- 詳細例：[ストリートテラス](https://machidukuri-fuchu.jp/mc-events/ストリートテラス-59/)。`.dtstart time[datetime]`、`.dtend time[datetime]`、`.mc-start-date` に開催情報。2026年11月29日12:00〜16:30の表示を確認。
- 日時属性に `2026-11-29T12:00:00+32400:00` のようなRFC3339でないオフセットがある。属性を無条件に標準日時へ変換せず、表示の年月日・時刻と照合する。全日イベントに任意の時刻を加えない。
- [robots.txt](https://machidukuri-fuchu.jp/robots.txt)：`/wp-admin/`禁止、`admin-ajax.php`許可。公開カレンダー・詳細の禁止記載なし。
- [利用規約](https://machidukuri-fuchu.jp/agreement/)：第2項に無断複製等の制限、第4項(2)にリンクを張る場合の事前承諾条件がある（確認事実）。出典リンクを付けた配布用HTML/PDFという要件への適用・承諾状況は未確認であり、今回の読み取り調査だけで自動配布を許可済みとは扱わない（運用判断）。取得技術の確認と配布可否を分ける。

## 2. 府中観光協会

- 起点：[イベント一覧](https://www.kankou-fuchu.com/?p=we-page-eventlist&eventlist=17414&mode=list&type=event&nav=none)。通常HTTPは200、`#mitte-event-list ul.content-container` は空。
- 初期HTMLに `#we-page-eventlist`、`.calendar-container ul`、`.move-prev`、`.move-next` と詳細枠 `#mitte-event-entry ul.content-container` が存在するが、イベントデータはない。`js/mitte3.js`、`js/mitte-app3.js` 等が読み込まれる。少なくとも通常のHTMLパーサだけでは一覧取得できない。JS実行または別のデータ取得経路の追加調査が必要であり、chromedp導入を確定したわけではない。
- JS実行後の一覧DOMは末尾の追加確認を参照。月切替操作は未検証、詳細本文の構造は追加確認を参照。空枠を正常0件にしない。
- [robots.txt](https://www.kankou-fuchu.com/robots.txt) は200だがHTMLの共通アプリ画面を返し、有効なrobots規則として確認できなかった。利用条件ページも今回の初期HTMLから特定できず、許可とは判断しない。

## 3. 府中市市立公園

- 指定された `/news/261114/` は個別記事。継続起点は[お知らせ一覧](https://fuchupark-area.com/news/)。[2ページ目](https://fuchupark-area.com/news/page/2/) も200、別の記事を含む。
- 一覧：`a.news-list__item`、`.news-list__title`、`.news-list__date`。次ページ `a.nextpostslink[rel=next]`。日付は**投稿日**である。年月指定UIは確認できず、ページ送りで必要範囲を取得する。
- 詳細：[プレーパーク](https://fuchupark-area.com/news/261114/)。`main.news-detail`、`.news-detail__container`、`.news-detail__article-title`、`.news-detail__article-body` が候補。開催日の年がタイトルにない場合、記事URLの数字や投稿日だけで確定しない。チラシ画像だけの条件は抽出対象外。
- canonicalが別記事 `/news/260711/` を指す一方、パンくずJSON-LDは要求URL `/news/261114/` を示す不一致を確認。出典URLは実際に取得した記事URLを保持し、canonicalだけで統合しない。
- [robots.txt](https://fuchupark-area.com/robots.txt)：`/wp/wp-admin/`禁止、同配下の `admin-ajax.php`許可。公開一覧・記事の禁止記載なし。[プライバシーポリシー](https://fuchupark-area.com/privacy-policy/) は個人情報中心で、自動取得・再利用の包括許諾は確認できない。

## 4. 聖蹟桜ヶ丘エリアマネジメント

- 起点：[イベント](https://seiseki.org/event/)。一覧 `div.eventCell` ごとに `.eventCellText > a` が詳細リンク、その直下の `p` がタイトル、`p.times` が日付または日付範囲。`.eventTimes small` が年月、`strong` が日。タグリンクを詳細URLとして誤取得しない。
- 次ページ `a.next.page-numbers`、[2ページ目](https://seiseki.org/event/page/2/) 200。調査時は最終13ページへのリンクがあった。月指定UIは確認できず、掲載順が開催日順と保証されないため投稿日や最初の過去日だけで巡回を終了しない。
- 詳細例：[第6回せいせきカワマチオープンデイ](https://seiseki.org/%e3%80%902026%e5%b9%b411%e6%9c%887%e6%97%a5%ef%bc%88%e5%9c%9f%ef%bc%89%e3%83%bb8%e6%97%a5%ef%bc%88%e6%97%a5%ef%bc%89%e9%96%8b%e5%82%ac%e3%80%91%e7%ac%ac6%e5%9b%9e%e3%81%9b%e3%81%84%e3%81%9b%e3%81%8d/)（実装では一覧にあるpercent-encoded hrefをそのまま利用）。`.newsSingleBlockTitle h3` がタイトル、`.newsSingleBlockTitle > p.times` が `2026.11.07〜2026.11.08`。本文は `.newsSingleBlock.bottomSpace60` 内。`.singleMain` は日付と主画像だけで、本文段落はその**兄弟**にある。
- 例では本文に「日時」「場所」、10:00〜15:30、雨天中止がある。年齢条件はないので不明。日付範囲の各日同時刻かは本文と照合する。`dl` の定型構造ではなく自由記述のため、ラベルが見つからない記事も正常に保持する。
- [中止記事の例](https://seiseki.org/verdy2026/) はタイトルに中止、本文に日時・場所・料金・申込期間・小学1〜3年生対象の**個別企画**がある。企画の対象年齢をイベント全体に適用しない。単なる「終了」と「中止」を区別する。
- [robots.txt](https://seiseki.org/robots.txt)：管理画面禁止、公開イベントの禁止記載なし。[プライバシーポリシー](https://seiseki.org/privacy/) は個人情報の扱いで、自動取得・転載の包括許諾は確認できない。
- HTMLにJavaScript有効化の文言があるが、一覧・詳細本文はHTTP応答に含まれ、収集にJSは不要。

## 5. 郷土の森博物館

- 起点：[イベント案内](https://www.fuchu-cpf.or.jp/museum/event/index.html)。おすすめ記事と `/museum/event/1000080/index.html` 等のカテゴリを辿る。ページ番号だけで記事か一覧かを判断せず、`#content` の構造を見る。
- 詳細例：[バックヤードツアー](https://www.fuchu-cpf.or.jp/museum/event/1008189.html)。`#content h1`、`#content dl dt` と直後の `dd` に開催日、開催時間、会場、料金、応募方法。2026年12月9日9時半〜正午、3,000円、11月15日応募締切のHTML記載あり。年齢は未記載。
- [鉄道模型博](https://www.fuchu-cpf.or.jp/museum/event/1005994.html) のように開催期間・休館日・個別企画の複数 `dl` がある。見出しの文脈とともに保存し、最初の日時だけで全内容を代表させない。
- 静的一覧で月指定・通常の次ページは未確認。スケジュールリンク `/cgi-evt/event.cgi?s_category=2` は[robots.txt](https://www.fuchu-cpf.or.jp/robots.txt) の `/cgi-evt/`禁止に該当するため取得しない。`/cgi-bin/`、`/cgi-mgr/` も禁止。
- [ご利用上の注意](https://www.fuchu-cpf.or.jp/museum/access/1000423.html) は来館者向けの注意であり、Web再利用許諾ではない。著作権表示あり。静的ページから日時等は抽出できるが、全イベントの網羅性は未確認。

## 6. ルミエール府中

- 指定の[イベントURL](https://www.lumiere-fuchu.jp/event/) とトップは通常HTTPで `/welcome/` に転送され、市民会館と図書館の選択画面を返した。ヘッドレスChromeで12秒の仮想時間を与えたDOMも同じ案内のみ。これだけで「JS必須」とは判断できない。
- 初期HTMLにはXO Event CalendarのJS、`xo_event_calendar_month` と `admin-ajax.php` 設定があるが、実イベント一覧・月切替・詳細・対象年齢・日時のDOMには到達できていない。架空の一般的なCookieを付けた検証でも転送は同じ。実際の選択操作とCookie条件は未調査。
- [robots.txt](https://www.lumiere-fuchu.jp/robots.txt)：`/__contents/wp-admin/`禁止、同配下 `admin-ajax.php`許可。案内画面から利用条件ページは確認できず、取得経路とともに次回調査対象とする。

## 7. 府中の森芸術劇場

- 指定の[トップ](https://www.fuchu-cpf.or.jp/theater/) は総合案内。[劇場お勧めラインナップ](https://www.fuchu-cpf.or.jp/theater/1000331/recommend/index.html)、`/theater/1000331/1003104/index.html` 等の静的一覧を辿れる。
- 詳細例：[どりーむコンサートVol.135](https://www.fuchu-cpf.or.jp/theater/1000331/recommend/1007742.html)。`#content h1`、`#content dl dt` と直後の `dd` に開催日・開催時間・会場・料金・発売情報。2026年11月7日14時開演。本文に未就学児入場不可があり、青少年席の22歳以下という価格条件とは分ける必要がある。
- 月指定・ページ送りは静的一覧では未確認。全催物スケジュール `/cgi-evt/event.cgi?s_category=1` は共通robotsの `/cgi-evt/`禁止対象につき取得しない。静的一覧だけの収集は全催物を保証できない。
- サイトは著作権表示とプライバシーポリシーへのリンクを持つ。今回、自動取得・再利用の包括許諾は確認できていない。JS有効化の注意はあるが、静的HTMLに必要項目があり収集にJSは不要。

## 残る確認

観光協会の通常HTTPデータ経路・月切替操作、ルミエールの市民会館選択後の正規遷移、各サイトの再利用条件が未確定。調査した詳細は例であり、全記事で同じ構造や全項目の記載を保証しない。収集器ではレイアウト不一致を成功0件とせず、上限ページ数・同一URL巡回防止・同一ホストへのアクセス間隔を設ける。

### 観光協会のブラウザ追加確認

ヘッドレスChromeで15秒の仮想時間を与えたDOMでは一覧が生成された（初回起動には実時間約1分半）。通常HTTPだけでは空だが、JS後の `#mitte-event-list .block-item.photo-list` にイベント、`h4` にタイトル、`a.atag[href]` に詳細URL、`p.event-date` に日付範囲、`p.summary` に概要がある。詳細URL例は [府中まんなかイルミ](https://www.kankou-fuchu.com/?p=we-page-event-entry&event=400214&cat=17414&type=event)。

`#we-page-eventlist .calendar-container li` に `.calendar-year` / `.calendar-month` があり、2025年12月〜2027年1月の月一覧を確認。月切替操作は未検証。詳細は次の追加確認を参照。初回実装ではブラウザ対応の追加候補とし、通常HTTPの空枠を正常0件にしない。上記比較表の「未対応」はJS対応が必要なことを示す。

詳細もChromeで確認できた。`#mitte-event-entry > ul.content-container > li.active` または要求したイベントIDの `li#event-entry400214` に限定する。DOMには過去記事やサンプルを含む複数記事が同時に入るため、コンテナ全体の本文を抽出しない。タイトル `.we-title h3`、本文 `.photo-list .block-text`、定型項目は同本文の `table tr th` と直後の `td`。会場・主催者・開催期間（開始日）・開催期間（終了日）等がある。本文内にも装飾用tableがあるため、項目名を照合する。年齢は見つからなければ不明。

月クリック候補は `#we-page-eventlist .calendar-container li#calendar12`（調査時2026年12月）。公式JSの当該liにclick handlerがあり、対応する `.content-container > li` のactive状態を切り替える。URLクエリによる月指定は確認できない。月番号のdata-titleは年を跨いで重複するので `.calendar-year` と `.calendar-month` を照合する。実際のクリック操作は未実施。

### 収集補助実装での再確認

2026年11月5日00:33 JST頃、chromedpで2026年11月・12月を実際にクリックし、各月の `#mitte-event-list > ul.content-container > li` の `data-loaded="true"` を待機した。複数月のイベントはDOMに残り、合計4件を取得できた。詳細400214から開催期間2026年11月19日〜12月25日と会場「府中駅南側ペデストリアンデッキ」を取得した。上の「月切替操作未実施」はこの検証により解消した。

ページ全体のload完了を待つと外部地図等の読み込みで75秒のタイムアウトになる例があった。`Page.navigate` 後に必要DOMとイベントデータを待つ方式では、一覧と詳細の一連の実検証が約22秒で完了した。固定秒数だけ待つ実装にせず、読込済み状態を確認する。
