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

// WriteHTML renders a self-contained calendar with details; unknown metadata is omitted.
func WriteHTML(w io.Writer, r event.Report) error {
	return htmlTemplate.Execute(w, calendarView(r))
}

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"url":           safeURL,
	"known":         known,
	"calendarLinks": calendarLinks,
	"weekdays":      func() []string { return []string{"月", "火", "水", "木", "金", "土", "日"} },
	"extraCount":    func(items []calendarItem) int { return len(items) - 3 },
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
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; script-src 'sha256-PZTRjZoxtXMtyY/WrVbG9VonHcJd9YhdiCVplW8Q21k='; img-src data:; font-src 'none'; base-uri 'none'; form-action 'none'"><title>府中の子ども向けイベント</title>
<style>
*{box-sizing:border-box}body{margin:0;background:#f4f5f1;color:#25332f;font-family:-apple-system,BlinkMacSystemFont,"Hiragino Sans","Noto Sans CJK JP",sans-serif;line-height:1.7}main{max-width:850px;margin:auto;padding:24px 16px}h1{font-size:1.6rem;line-height:1.4}h2{font-size:1.2rem;margin:0 0 12px}h3{font-size:1rem}.meta{color:#52635c;font-size:.9rem}.card,.notice{background:white;border:1px solid #d5ded7;border-radius:12px;padding:20px;margin:16px 0;overflow-wrap:anywhere}.card{break-inside:avoid}dl{margin:12px 0}dt{font-weight:600;color:#456052}dd{margin:0 0 8px;white-space:pre-wrap}a{color:#225e50;text-underline-offset:3px}.review{background:#fff5d9;border-radius:6px;padding:8px}.summary{white-space:pre-wrap}ul{padding-left:1.4em}footer{font-size:.85rem;color:#52635c}@media(max-width:480px){main{padding:16px 12px}.card{padding:16px}h1{font-size:1.35rem}}@page{size:A4;margin:12mm}@media print{body{background:white;font-size:10pt}main{padding:0;max-width:none}.card{border-radius:4px;margin:10px 0;padding:12px}a{color:inherit}.notice{padding:10px}h1{font-size:18pt}}

main{max-width:1240px}header>h1{font-size:1.35rem}header .meta{line-height:1.8}.eyebrow{font-size:11px;letter-spacing:.18em;color:#687a75}.calendar-toolbar{display:flex;justify-content:space-between;align-items:center;margin-top:28px}.calendar-toolbar h2{font-size:28px;margin:4px 0}.month-controls{display:flex;gap:8px}.month-controls button{background:#fff;border:1px solid #dce3e0;border-radius:50%;width:42px;height:42px;font-size:25px;cursor:pointer}.month-controls button:disabled{opacity:.3}.legend{display:flex;align-items:center;gap:8px;font-size:12px;color:#52635c}.dot{display:inline-block;width:9px;height:9px;border-radius:50%}.recommended{background:#d6ede6;color:#174f40}.seasonal{background:#f3e2ee;color:#783f6c}.pending{background:#fff0ce;color:#765925}.weekdays,.month-grid{display:grid;grid-template-columns:repeat(7,minmax(0,1fr))}.weekdays{background:#fff;border:1px solid #dce3e0;border-bottom:0;border-radius:16px 16px 0 0}.weekdays div{text-align:center;padding:14px;font-size:13px;color:#667570}.month-grid{border-left:1px solid #dce3e0;border-top:1px solid #dce3e0;background:#fff}.day{min-height:140px;min-width:0;border-right:1px solid #dce3e0;border-bottom:1px solid #dce3e0;padding:7px 5px}.other-month{background:#f6f8f7;color:#a6afaa}.day-number{display:flex;align-items:center;justify-content:center;width:28px;height:28px;font-size:13px;margin:0 auto 7px}.today{background:#235b4b;color:#fff;border-radius:50%}.calendar-event{border:0;border-radius:5px;width:100%;display:block;margin:3px 0;padding:6px;text-align:left;cursor:pointer;font:inherit;font-size:12px;line-height:1.4}.calendar-event span{display:block;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.extra-item{display:none}.day.expanded .extra-item{display:block}.more-events{background:none;border:0;color:#53665f;padding:4px;font-size:12px;cursor:pointer}.screen-review{margin-top:30px}.pending-row{display:block;width:100%;text-align:left;border:0;border-bottom:1px solid #dce3e0;background:transparent;padding:14px 4px;color:#374d43;font:inherit;cursor:pointer}.pending-row span{float:right;font-size:12px;color:#6b7e75}dialog{border:0;border-radius:18px;padding:24px;max-width:640px;width:calc(100% - 24px);max-height:90vh;color:#25332f}dialog::backdrop{background:#182d2766;backdrop-filter:blur(3px)}dialog .card{margin:0;border:0;padding:8px}#dialog-close{position:sticky;top:0;float:right;border:0;border-radius:50%;background:#edf2ee;width:34px;height:34px;font-size:24px;cursor:pointer}.calendar-links a{display:inline-block;background:#e3efe9;border-radius:8px;padding:9px 12px}.calendar-links small{display:block;color:#687a75;font-size:12px;margin-top:5px}.interactive .detail-lists{display:none}.calendar-ui,.screen-review{display:none}.interactive .calendar-ui,.interactive .screen-review{display:block}.exporting .calendar-ui,.exporting .screen-review,.exporting dialog{display:none}.exporting .detail-lists{display:block}@media(max-width:600px){.day{min-height:106px;padding:4px 2px}.calendar-event{font-size:10px;padding:4px 3px}.calendar-toolbar h2{font-size:24px}.pending-row span{float:none;display:block}.weekdays div{padding:10px 0}main{padding:16px 8px}}@media print{.calendar-ui,.screen-review,dialog{display:none!important}.detail-lists{display:block!important}}
</style></head><body><main>
<header><h1>府中の子ども向けイベント</h1><p class="meta">対象期間：{{fallback .From "不明"}} 〜 {{fallback .Until "不明"}}<br>生成日時：{{generated .GeneratedAt}}<br>対象の目安：{{.AgeMonths}}か月 ／ おすすめ{{len .Events}}件・期間情報{{len .Information}}件・要確認{{len .Review}}件</p><p>日時・参加条件は変更される場合があります。お出かけ前に出典の公式情報をご確認ください。</p></header>
{{if .Warnings}}<details class="notice"><summary>取得状況・確認事項（{{len .Warnings}}件）</summary><ul>{{range .Warnings}}<li>{{displayMessage .}}</li>{{end}}</ul></details>{{end}}
<div class="calendar-ui"><div class="calendar-toolbar"><div><span class="eyebrow">FUCHU · FAMILY DAYS</span><h2 id="month-label">カレンダー</h2></div><div class="month-controls"><button type="button" id="month-prev" aria-label="前の月">‹</button><button type="button" id="month-next" aria-label="次の月">›</button></div></div><p class="legend"><span class="dot recommended"></span>おすすめ <span class="dot seasonal"></span>期間限定 </p>
{{range $index,$month:=.Months}}<section class="month" data-month-label="{{.Label}}" {{if $index}}hidden{{end}}><div class="weekdays">{{range weekdays}}<div>{{.}}</div>{{end}}</div><div class="month-grid">{{range .Days}}<div class="day {{if .Other}}other-month{{end}}"><span class="day-number {{if .Today}}today{{end}}">{{.Number}}</span>{{range $i,$item:=.Items}}<button type="button" class="calendar-event {{$item.Kind}} {{if ge $i 3}}extra-item{{end}}" data-open="{{$item.Target}}" title="{{$item.Title}}"><span>{{$item.Title}}</span></button>{{end}}{{if gt (len .Items) 3}}<button type="button" class="more-events">他 {{extraCount .Items}}件</button>{{end}}</div>{{end}}</div></section>{{end}}</div>
<section class="screen-review"><h2>要確認の候補</h2><p>日程や内容の確認が必要なものは、カレンダーと分けています。</p>{{range .Entries}}{{if eq .Kind "pending"}}<button class="pending-row" type="button" data-open="{{.Target}}">{{.Event.Title}} <span>詳細を見る ↗</span></button>{{end}}{{end}}</section>
<dialog id="event-dialog" aria-label="イベント詳細"><button type="button" id="dialog-close" aria-label="閉じる">×</button><div id="dialog-body"></div></dialog>
{{range .Entries}}<template id="{{.Target}}">{{template "eventCard" .Event}}</template>{{end}}
<div class="detail-lists"><section id="events" data-event-section aria-label="おすすめイベント"><h2>おすすめイベント</h2>{{range .Events}}{{template "eventCard" .}}{{else}}<p class="notice">おすすめに該当するイベントはありません。</p>{{end}}</section>
{{if .Information}}<section data-event-section aria-label="期間限定のお出かけ情報"><h2>期間限定のお出かけ情報</h2>{{range .Information}}<article class="card" data-event><h3>{{.Title}}</h3><p>{{fallback .DateText "期間要確認"}}</p>{{if url .URL}}<a href="{{url .URL}}">開催期間・詳細を確認</a>{{end}}</article>{{end}}</section>{{end}}
{{if .Review}}<section data-event-section aria-label="要確認の候補"><h2>要確認の候補</h2><p>日程や参加条件、個別企画の確認が必要な候補です。</p>{{range .Review}}{{template "eventCard" .}}{{end}}</section>{{end}}
</div>{{if .Sources}}<aside class="notice"><h2>情報源ごとの取得状況</h2><ul>{{range .Sources}}<li>{{if and (ne .Status "disabled") (url .URL)}}<a href="{{url .URL}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}：{{sourceStatus .Status}}（{{.Count}}件）{{if .Error}} — 取得失敗・一部取得：{{displayMessage .Error}}{{end}}</li>{{end}}</ul></aside>{{end}}
<footer>未記載の条件は推測していません。対象年齢不明の候補を含みます。</footer></main><script>(()=>{document.documentElement.classList.add('interactive');const months=Array.from(document.querySelectorAll('.month'));let index=0;const label=document.getElementById('month-label'),prev=document.getElementById('month-prev'),next=document.getElementById('month-next');function show(){months.forEach((m,i)=>m.hidden=i!==index);if(months[index])label.textContent=months[index].dataset.monthLabel;prev.disabled=index===0;next.disabled=index===months.length-1}prev.addEventListener('click',()=>{if(index>0){index--;show()}});next.addEventListener('click',()=>{if(index<months.length-1){index++;show()}});show();const dialog=document.getElementById('event-dialog'),body=document.getElementById('dialog-body');document.addEventListener('click',e=>{const open=e.target.closest('[data-open]');if(open){const t=document.getElementById(open.dataset.open);if(t&&t.content){body.replaceChildren(t.content.cloneNode(true));dialog.showModal()}}const more=e.target.closest('.more-events');if(more){const expanded=more.parentElement.classList.toggle('expanded');more.textContent=expanded?'折りたたむ':'他 '+more.parentElement.querySelectorAll('.extra-item').length+'件'}});document.getElementById('dialog-close').addEventListener('click',()=>dialog.close());dialog.addEventListener('click',e=>{if(e.target===dialog)dialog.close()})})();</script></body></html>{{define "eventCard"}}<article class="card" data-event><h2>{{.Title}}</h2>{{if uncertainDate .}}<p class="review">開催日未確定</p>{{end}}{{if .Summary}}<p class="summary">{{.Summary}}</p>{{end}}<dl>{{if known .DateText}}<dt>開催日</dt><dd>{{.DateText}}</dd>{{else if .Dates}}<dt>開催日</dt><dd>{{range .Dates}}{{.Start}}{{if .End}} 〜 {{.End}}{{end}}<br>{{end}}</dd>{{end}}{{if known .TimeText}}<dt>時間</dt><dd>{{.TimeText}}</dd>{{end}}{{if known .Venue}}<dt>会場</dt><dd>{{.Venue}}</dd>{{end}}{{if known .AgeText}}<dt>対象年齢</dt><dd>{{.AgeText}}</dd>{{end}}{{if known .Reservation}}<dt>参加条件・予約</dt><dd>{{.Reservation}}</dd>{{end}}{{if known .Fee}}<dt>参加費</dt><dd>{{.Fee}}</dd>{{end}}{{if known .DeadlineText}}<dt>予約締切</dt><dd>{{.DeadlineText}}</dd>{{end}}</dl><div class="calendar-links">{{range calendarLinks .}}<p><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">Googleカレンダーに登録 · {{.Label}}</a><small>{{.Note}}</small></p>{{end}}</div>{{if .Recommendation}}<p class="review"><strong>興味の補助情報</strong><br>{{.Recommendation}}</p>{{end}}{{if .Assessment}}<div class="review"><strong>判定理由</strong><ul>{{range .Assessment.Reasons}}<li>{{.}}</li>{{end}}</ul>{{if .Assessment.Evidence}}<p>根拠：{{range .Assessment.Evidence}}{{.}} {{end}}</p>{{end}}</div>{{end}}{{if .NeedsReview}}<div class="review">要確認<ul>{{range .NeedsReview}}<li>{{.}}</li>{{end}}</ul></div>{{end}}<p>出典：{{range .Sources}}{{if url .URL}}<a href="{{url .URL}}" rel="noreferrer">{{fallback .Name "公式情報"}}</a> {{else}}{{fallback .Name "出典URL不明"}} {{end}}{{end}}{{if url .URL}} <a href="{{url .URL}}" rel="noreferrer">詳細情報</a>{{end}}</p></article>{{end}}`))
