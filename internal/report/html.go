// Package report creates self-contained Japanese event handouts.
package report

import (
	"html/template"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

func safeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return ""
	}
	return raw
}

// WriteHTML renders a self-contained report. Unknown fields remain explicitly unknown.
func WriteHTML(w io.Writer, r event.Report) error {
	return htmlTemplate.Execute(w, r)
}

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"url": safeURL,
	"displayMessage": func(s string) string {
		for _, marker := range []string{"exception ", "at tick", "context deadline exceeded", "dial tcp", "net/http", "chrome failed", "ERR_"} {
			if strings.Contains(s, marker) {
				return "取得できなかった情報があります。取得状況と実行ログをご確認ください。"
			}
		}
		return s
	},
	"ageStatus": func(s string) string {
		if s == "eligible" {
			return "明示された年齢条件の範囲内"
		}
		return "要確認"
	},
	"sourceStatus": func(s string) string {
		switch s {
		case "success":
			return "取得済み"
		case "partial":
			return "一部取得"
		case "error":
			return "取得失敗"
		case "disabled":
			return "保留"
		default:
			return "要確認"
		}
	},
	"uncertainDate": func(e event.Event) bool {
		if len(e.Dates) == 0 {
			return true
		}
		for _, d := range e.Dates {
			if d.Start == "" {
				return true
			}
		}
		return false
	},
	"fallback": func(s, fallback string) string {
		if strings.TrimSpace(s) == "" {
			return fallback
		}
		return s
	},
	"generated": func(t time.Time) string {
		if t.IsZero() {
			return "不明"
		}
		return t.In(time.FixedZone("JST", 9*60*60)).Format("2006-01-02 15:04 JST")
	},
}).Parse(`<!doctype html>
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src 'none'; base-uri 'none'; form-action 'none'"><title>府中の子ども向けイベント</title>
<style>
*{box-sizing:border-box}body{margin:0;background:#f4f5f1;color:#25332f;font-family:-apple-system,BlinkMacSystemFont,"Hiragino Sans","Noto Sans CJK JP",sans-serif;line-height:1.7}main{max-width:850px;margin:auto;padding:24px 16px}h1{font-size:1.6rem;line-height:1.4}h2{font-size:1.2rem;margin:0 0 12px}h3{font-size:1rem}.meta{color:#52635c;font-size:.9rem}.card,.notice{background:white;border:1px solid #d5ded7;border-radius:12px;padding:20px;margin:16px 0;overflow-wrap:anywhere}.card{break-inside:avoid}dl{margin:12px 0}dt{font-weight:600;color:#456052}dd{margin:0 0 8px;white-space:pre-wrap}a{color:#225e50;text-underline-offset:3px}.review{background:#fff5d9;border-radius:6px;padding:8px}.summary{white-space:pre-wrap}ul{padding-left:1.4em}footer{font-size:.85rem;color:#52635c}@media(max-width:480px){main{padding:16px 12px}.card{padding:16px}h1{font-size:1.35rem}}@page{size:A4;margin:12mm}@media print{body{background:white;font-size:10pt}main{padding:0;max-width:none}.card{border-radius:4px;margin:10px 0;padding:12px}a{color:inherit}.notice{padding:10px}h1{font-size:18pt}}
</style></head><body><main>
<header><h1>府中の子ども向けイベント</h1><p class="meta">対象期間：{{fallback .From "不明"}} 〜 {{fallback .Until "不明"}}<br>生成日時：{{generated .GeneratedAt}}<br>対象の目安：{{.AgeMonths}}か月 ／ おすすめ{{len .Events}}件・期間情報{{len .Information}}件・要確認{{len .Review}}件</p><p>日時・参加条件は変更される場合があります。お出かけ前に出典の公式情報をご確認ください。</p></header>
{{if .Warnings}}<aside class="notice"><h2>確認事項</h2><ul>{{range .Warnings}}<li>{{displayMessage .}}</li>{{end}}</ul></aside>{{end}}
<section id="events" data-event-section aria-label="おすすめイベント"><h2>おすすめイベント</h2>{{range .Events}}{{template "eventCard" .}}{{else}}<p class="notice">おすすめに該当するイベントはありません。</p>{{end}}</section>
{{if .Information}}<section data-event-section aria-label="期間限定のお出かけ情報"><h2>期間限定のお出かけ情報</h2>{{range .Information}}<article class="card" data-event><h3>{{.Title}}</h3><p>{{fallback .DateText "期間要確認"}}</p>{{if url .URL}}<a href="{{url .URL}}">開催期間・詳細を確認</a>{{end}}</article>{{end}}</section>{{end}}
{{if .Review}}<section data-event-section aria-label="要確認の候補"><h2>要確認の候補</h2><p>日程や参加条件、個別企画の確認が必要な候補です。</p>{{range .Review}}{{template "eventCard" .}}{{end}}</section>{{end}}
{{if .Sources}}<aside class="notice"><h2>情報源ごとの取得状況</h2><ul>{{range .Sources}}<li>{{if and (ne .Status "disabled") (url .URL)}}<a href="{{url .URL}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}：{{sourceStatus .Status}}（{{.Count}}件）{{if .Error}} — 取得失敗・一部取得：{{displayMessage .Error}}{{end}}</li>{{end}}</ul></aside>{{end}}
<footer>未記載の条件は推測していません。対象年齢不明の候補を含みます。</footer></main></body></html>{{define "eventCard"}}<article class="card" data-event><h2>{{.Title}}</h2>{{if uncertainDate .}}<p class="review">開催日未確定</p>{{end}}{{if .Summary}}<p class="summary">{{.Summary}}</p>{{end}}<dl><dt>開催日</dt><dd>{{if .DateText}}{{.DateText}}{{else if .Dates}}{{range .Dates}}{{fallback .Start "開催日不明"}}{{if .End}} 〜 {{.End}}{{end}}<br>{{end}}{{else}}開催日不明{{end}}</dd><dt>時間</dt><dd>{{fallback .TimeText "時間不明"}}</dd><dt>会場</dt><dd>{{fallback .Venue "会場不明"}}</dd><dt>対象年齢</dt><dd>{{fallback .AgeText "対象年齢不明"}}{{if .AgeStatus}}（{{ageStatus .AgeStatus}}）{{end}}</dd><dt>参加条件・予約</dt><dd>{{fallback .Reservation "参加条件・予約の記載なし（要確認）"}}</dd><dt>参加費</dt><dd>{{fallback .Fee "参加費不明"}}</dd><dt>予約締切</dt><dd>{{fallback .DeadlineText "締切の記載なし（要確認）"}}</dd></dl>{{if .Recommendation}}<p class="review"><strong>興味の補助情報</strong><br>{{.Recommendation}}</p>{{end}}{{if .Assessment}}<div class="review"><strong>判定理由</strong><ul>{{range .Assessment.Reasons}}<li>{{.}}</li>{{end}}</ul>{{if .Assessment.Evidence}}<p>根拠：{{range .Assessment.Evidence}}{{.}} {{end}}</p>{{end}}</div>{{end}}{{if .NeedsReview}}<div class="review">要確認<ul>{{range .NeedsReview}}<li>{{.}}</li>{{end}}</ul></div>{{end}}<p>出典：{{range .Sources}}{{if url .URL}}<a href="{{url .URL}}" rel="noreferrer">{{fallback .Name "公式情報"}}</a> {{else}}{{fallback .Name "出典URL不明"}} {{end}}{{end}}{{if url .URL}} <a href="{{url .URL}}" rel="noreferrer">詳細情報</a>{{end}}</p></article>{{end}}`))
