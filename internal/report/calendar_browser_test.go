package report

import (
	"context"
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
	r := event.Report{From: "2026-11-01", Until: "2026-12-10", Events: []event.Event{{Title: "親子で遊ぶ日", Dates: []event.DateRange{{Start: "2026-11-07"}}}}}
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
	check(`(()=>{document.querySelector('.calendar-event').click();const d=document.getElementById('event-dialog');return d.open&&d.textContent.includes('親子で遊ぶ日')&&!!d.querySelector('a[href*="calendar.google.com"]')})()`)
	check(`(()=>{document.getElementById('dialog-close').click();return !document.getElementById('event-dialog').open})()`)
	check(`(()=>{document.getElementById('month-next').click();return document.getElementById('month-label').textContent==='2026年12月'})()`)
	check(`(()=>{document.getElementById('month-prev').click();return document.getElementById('month-label').textContent==='2026年11月'})()`)
}
