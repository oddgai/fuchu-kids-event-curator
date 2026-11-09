package report

import (
	"context"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCalendarInteractionBrowser(t *testing.T) {
	executable := os.Getenv("REPORT_TEST_BROWSER")
	if executable == "" {
		t.Skip("set REPORT_TEST_BROWSER")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.ExecPath(executable))
	allocator, done := chromedp.NewExecAllocator(ctx, options...)
	defer done()
	browser, closeBrowser := chromedp.NewContext(allocator)
	defer closeBrowser()
	r := event.Report{From: "2026-11-01", Until: "2026-12-10", Events: []event.Event{
		{Title: "親子で遊ぶ日", Dates: []event.DateRange{{Start: "2026-11-07"}}, TimeText: "10:00〜11:30", Venue: "府中公園", URL: "https://example.org/play", Sources: []event.SourceRef{{Name: "公園", URL: "https://example.org/play"}}, Assessment: &event.Assessment{Category: "recommend", Reasons: []string{"幼児歓迎"}}},
		{Title: "期間のある催し", Dates: []event.DateRange{{Start: "2026-11-06", End: "2026-11-10"}}},
	}}
	path := filepath.Join(t.TempDir(), "calendar.html")
	f, _ := os.Create(path)
	if err := WriteHTML(f, r); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := chromedp.Run(browser, chromedp.Navigate((&url.URL{Scheme: "file", Path: path}).String())); err != nil {
		t.Fatal(err)
	}
	check := func(script string) {
		t.Helper()
		ok, err := chromedp.Run(browser, chromedp.Evaluate[bool](script))
		if err != nil || !ok {
			t.Fatalf("browser check failed %s: %v", script, err)
		}
	}
	check(`document.documentElement.classList.contains('interactive') && document.getElementById('month-label').textContent==='2026年11月'`)
	check(`(()=>{document.querySelector('.calendar-event[title="親子で遊ぶ日"]').click();const d=document.getElementById('event-dialog');return d.open&&d.textContent.includes('親子で遊ぶ日')&&!!d.querySelector('a[href*="calendar.google.com"]')})()`)
	check(`(()=>{const d=document.getElementById('event-dialog'),a=d.querySelector('.calendar-links a'),q=new URL(a.href).searchParams;return a.textContent==='Googleカレンダーに登録'&&q.get('dates')==='20261107T100000/20261107T113000'&&q.get('location')==='府中公園'&&q.get('details')==='https://example.org/play'&&d.textContent.includes('府中公園')&&!d.textContent.includes('判定理由')&&!d.textContent.includes('詳細情報')&&d.querySelectorAll('a[href="https://example.org/play"]').length===1})()`)
	check(`(()=>{document.getElementById('dialog-close').click();return !document.getElementById('event-dialog').open})()`)
	check(`(()=>{document.getElementById('month-next').click();return document.getElementById('month-label').textContent==='2026年12月'})()`)
	check(`(()=>{document.getElementById('month-prev').click();return document.getElementById('month-label').textContent==='2026年11月'})()`)
	checkSpanningBars := func() {
		t.Helper()
		check(`(()=>{const bars=Array.from(document.querySelectorAll('.calendar-event[title="期間のある催し"]'));if(bars.length!==2||bars[0].dataset.open!==bars[1].dataset.open)return false;const box=bars[0].getBoundingClientRect(),start=document.querySelector('[data-date="2026-11-06"]').getBoundingClientRect(),end=document.querySelector('[data-date="2026-11-08"]').getBoundingClientRect(),single=document.querySelector('.calendar-event[title="親子で遊ぶ日"]').getBoundingClientRect();return Math.abs(box.left-start.left)<=4&&Math.abs(box.right-end.right)<=4&&single.top>=box.bottom})()`)
	}
	checkSpanningBars()
	check(`(()=>{for(const bar of document.querySelectorAll('.calendar-event[title="期間のある催し"]')){bar.click();const d=document.getElementById('event-dialog');if(!d.open||!d.textContent.includes('期間のある催し'))return false;d.close()}return true})()`)
	if _, err := chromedp.Run(browser, func(ctx context.Context, t *chromedp.Target) (chromedp.Void, error) {
		_, err := cdp.Call(ctx, t, emulation.SetDeviceMetricsOverride, emulation.SetDeviceMetricsOverrideParams{Width: 390, Height: 844, DeviceScaleFactor: 1, Mobile: true})
		return chromedp.Void{}, err
	}); err != nil {
		t.Fatal(err)
	}
	check(`document.documentElement.scrollWidth <= window.innerWidth && document.querySelector('.month-grid').getBoundingClientRect().right <= window.innerWidth`)
	checkSpanningBars()
	if path := os.Getenv("CALENDAR_SCREENSHOT"); path != "" {
		image, err := chromedp.Run(browser, chromedp.FullScreenshot(90))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, image, 0600); err != nil {
			t.Fatal(err)
		}
	}

}
