package collect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

// BrowserHTML waits for the actual event data, rather than the initially empty shell.
func BrowserHTML(ctx context.Context, rawURL, browserPath string, months []string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" || u.Hostname() != "www.kankou-fuchu.com" {
		return nil, fmt.Errorf("観光協会以外のブラウザ取得は対象外です")
	}
	ctx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if browserPath != "" {
		opts = append(opts, chromedp.ExecPath(browserPath))
	}
	allocator, closeAllocator := chromedp.NewExecAllocator(ctx, opts...)
	defer closeAllocator()
	browser, closeBrowser := chromedp.NewContext(allocator)
	defer closeBrowser()
	_, err = chromedp.Run(browser, func(ctx context.Context, target *chromedp.Target) (chromedp.Void, error) {
		result, err := cdp.Call(ctx, target, page.Navigate, page.NavigateParams{URL: rawURL})
		if err == nil && result.ErrorText != "" {
			err = fmt.Errorf("navigation: %s", result.ErrorText)
		}
		return chromedp.Void{}, err
	})
	if err != nil {
		return nil, err
	}
	if _, err = chromedp.Run(browser, chromedp.WaitReady("#we-page-eventlist")); err != nil {
		return nil, err
	}
	// All requested months remain in the DOM. Validate every month against its year label.
	if months == nil {
		months = []string{}
	}
	for _, month := range months {
		if _, err := time.Parse("2006-01", month); err != nil {
			return nil, fmt.Errorf("invalid month %q: %w", month, err)
		}
	}
	encoded, _ := json.Marshal(months)
	detail := u.Query().Get("p") == "we-page-event-entry"
	var script string
	if detail {
		id, _ := json.Marshal(u.Query().Get("event"))
		script = fmt.Sprintf(`new Promise((resolve,reject)=>{const end=Date.now()+55000;const tick=()=>{const el=document.getElementById('event-entry'+%s);if(el&&el.classList.contains('active')&&el.querySelector('.block-text'))return resolve(true);if(Date.now()>end)return reject(new Error('詳細データの待機期限'));setTimeout(tick,150)};tick()})`, id)
	} else {
		script = fmt.Sprintf(`new Promise((resolve,reject)=>{const wanted=%s,end=Date.now()+55000;let index=0,clicked=false;const tick=()=>{const tabs=Array.from(document.querySelectorAll('#we-page-eventlist .calendar-container li'));if(!tabs.length){if(Date.now()>end)return reject(new Error('月一覧の待機期限'));return setTimeout(tick,150)}if(!wanted.length){const current=document.querySelector('#mitte-event-list > ul.content-container > li.active');if(current?.getAttribute('data-loaded')==='true')return resolve(true);if(Date.now()>end)return reject(new Error('一覧データの待機期限'));return setTimeout(tick,150)}if(index>=wanted.length)return resolve(true);const ym=wanted[index].split('-'),tab=tabs.find(el=>parseInt(el.querySelector('.calendar-year')?.textContent)===+ym[0]&&parseInt(el.querySelector('.calendar-month')?.textContent)===+ym[1]);if(!tab)return reject(new Error('要求月が一覧にありません: '+wanted[index]));if(!clicked){tab.click();clicked=true}const n=tabs.indexOf(tab),body=document.querySelectorAll('#mitte-event-list > ul.content-container > li')[n];if(tab.classList.contains('active')&&body&&body.getAttribute('data-loaded')==='true'){index++;clicked=false}if(Date.now()>end)return reject(new Error('月データの待機期限'));setTimeout(tick,150)};tick()})`, encoded)
	}
	_, err = chromedp.Run(browser, chromedp.Evaluate[bool](script, chromedp.EvalAwaitPromise))
	if err != nil {
		return nil, err
	}
	result, err := chromedp.Run(browser, chromedp.Evaluate[string](`document.documentElement.outerHTML`))
	return []byte(result), err
}

func ParseKankouList(b []byte, pageURL string) ([]Candidate, string, error) {
	d, err := goquery.NewDocumentFromReader(bytes.NewReader(b))
	if err != nil {
		return nil, "", err
	}
	var out []Candidate
	seen := map[string]bool{}
	d.Find("#mitte-event-list .photo-list").Each(func(_ int, s *goquery.Selection) {
		u := absolute(pageURL, s.Find("a.atag").AttrOr("href", ""))
		title := clean(s.Find("h4").Text())
		if u != "" && title != "" && !seen[u] {
			seen[u] = true
			out = append(out, Candidate{Title: title, URL: u, DateText: clean(s.Find(".event-date").Text())})
		}
	})
	if len(out) == 0 {
		return nil, "", fmt.Errorf("観光協会のイベントデータを確認できません（空一覧または未読込）")
	}
	return out, "", nil
}

func ParseKankouDetail(s Source, c Candidate, b []byte, now time.Time) (event.Event, error) {
	d, err := goquery.NewDocumentFromReader(bytes.NewReader(b))
	if err != nil {
		return event.Event{}, err
	}
	u, err := url.Parse(c.URL)
	if err != nil {
		return event.Event{}, err
	}
	id := u.Query().Get("event")
	var root *goquery.Selection
	d.Find("#mitte-event-entry > ul.content-container > li").Each(func(_ int, el *goquery.Selection) {
		if el.AttrOr("data-id", "") == id || el.AttrOr("id", "") == "event-entry"+id {
			root = el
		}
	})
	if root == nil || id == "" || !root.HasClass("active") {
		return event.Event{}, fmt.Errorf("要求したイベントの詳細を確認できません")
	}
	c.Title = clean(root.Find(".we-title h3").First().Text())
	if c.Title == "" {
		return event.Event{}, fmt.Errorf("イベントタイトルを確認できません")
	}
	body := root.Find(".photo-list .block-text").Clone()
	fields := map[string]string{}
	body.Find("table tr").Each(func(_ int, tr *goquery.Selection) {
		k := clean(tr.ChildrenFiltered("th").Text())
		v := clean(tr.ChildrenFiltered("td").Text())
		if k != "" && fields[k] == "" {
			fields[k] = v
		}
	})
	if start := fields["開催期間（開始日）"]; start != "" {
		c.DateText = start
		if end := fields["開催期間（終了日）"]; end != "" {
			c.DateText += "〜" + end
		}
	}
	body.Find("table").Remove()
	content, _ := body.Html()
	var normalized strings.Builder
	normalized.WriteString("<main>")
	normalized.WriteString(content)
	normalized.WriteString("<dl>")
	for k, v := range fields {
		normalized.WriteString("<dt>" + html.EscapeString(k) + "</dt><dd>" + html.EscapeString(v) + "</dd>")
	}
	normalized.WriteString("</dl></main>")
	normalizedSource := s
	normalizedSource.ID = "" // Parse the normalized main element without dispatching to this adapter again.
	e, err := ParseDetail(normalizedSource, c, []byte(normalized.String()), now)
	if fields["会場"] != "" {
		e.Venue = fields["会場"]
	}
	if fields["開催期間（開始日）"] != "" {
		e.DateText = c.DateText
		e.Dates = ParseDates(c.DateText)
	}
	return e, err
}
