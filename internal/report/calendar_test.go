package report

import (
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"net/url"
	"strings"
	"testing"
)

func TestGoogleCalendarLinks(t *testing.T) {
	e := event.Event{Title: "親子 & 遊び", URL: "https://example.org/e", Venue: "公園", Dates: []event.DateRange{{Start: "2026-11-07"}}, TimeText: "10:00〜11:30"}
	links := calendarLinks(e)
	if len(links) != 1 {
		t.Fatal(links)
	}
	u, _ := url.Parse(links[0].URL)
	q := u.Query()
	if q.Get("dates") != "20261107T100000/20261107T113000" || q.Get("ctz") != "Asia/Tokyo" || q.Get("text") != e.Title {
		t.Fatal(q)
	}
	if q.Get("location") != "公園" || q.Get("details") != e.URL {
		t.Fatalf("location or URL-only description missing: %v", q)
	}
	e.Dates = []event.DateRange{{Start: "2026-12-31", End: "2027-01-02"}}
	e.TimeText = ""
	links = calendarLinks(e)
	u, _ = url.Parse(links[0].URL)
	if u.Query().Get("dates") != "20261231/20270103" {
		t.Fatal(u)
	}
	e.Dates = nil
	if len(calendarLinks(e)) != 0 {
		t.Fatal("unknown dates linked")
	}
	e.Dates = []event.DateRange{{Start: "2026-02-30"}}
	if len(calendarLinks(e)) != 0 {
		t.Fatal("invalid date linked")
	}
}
func TestMonthGrid(t *testing.T) {
	r := event.Report{From: "2026-11-05", Until: "2026-12-04", Events: []event.Event{{Title: "遊び", Dates: []event.DateRange{{Start: "2026-11-07"}, {Start: "2026-11-09"}}}}}
	v := calendarView(r)
	if len(v.Months) != 2 || v.Months[0].Days[0].ISO != "2026-10-26" {
		t.Fatalf("%+v", v.Months)
	}
	n := 0
	for _, d := range v.Months[0].Days {
		n += len(d.Items)
	}
	if n != 2 {
		t.Fatal(n)
	}
}

func TestCalendarLinksUsePerformanceTimes(t *testing.T) {
	for _, tc := range []struct {
		text      string
		want      []string
		tentative bool
	}{
		{"11時開演14時開演", []string{"20261107T110000/20261107T120000", "20261107T140000/20261107T150000"}, true},
		{"9時〜16時", []string{"20261107T090000/20261107T160000"}, false},
		{"10:00〜11:00 / 14:00〜15:00", []string{"20261107T100000/20261107T110000", "20261107T140000/20261107T150000"}, false},
		{"午後1時〜3時", []string{"20261107T130000/20261107T150000"}, false},
		{"6:00 PM〜8:00 PM", []string{"20261107T180000/20261107T200000"}, false},
		{"1:00〜3:00 PM", []string{"20261107T130000/20261107T150000"}, false},
		{"開演10時 終演11時", []string{"20261107T100000/20261107T110000"}, false},
		{"10時開演 11時終了", []string{"20261107T100000/20261107T110000"}, false},
		{"10:30開場 11:00開演", []string{"20261107T110000/20261107T120000"}, true},
		{"開場10時30分 開演11時", []string{"20261107T110000/20261107T120000"}, true},
		{"23時30分開演", []string{"20261107T233000/20261108T003000"}, true},
		{"100:00〜11:00", []string{"20261107/20261108"}, false},
		{"10:00〜11:00 / 14:00開演", []string{"20261107T100000/20261107T110000", "20261107T140000/20261107T150000"}, true},
	} {
		t.Run(tc.text, func(t *testing.T) {
			links := calendarLinks(event.Event{TimeText: tc.text, Dates: []event.DateRange{{Start: "2026-11-07"}}})
			if len(links) != len(tc.want) {
				t.Fatalf("links=%+v, want %d", links, len(tc.want))
			}
			tentative := false
			for i, link := range links {
				u, _ := url.Parse(link.URL)
				if u.Query().Get("dates") != tc.want[i] {
					t.Errorf("dates=%s, want %s", u.Query().Get("dates"), tc.want[i])
				}
				tentative = tentative || strings.Contains(link.Note, "終了時刻は仮")
			}
			if tentative != tc.tentative {
				t.Fatalf("tentative end warning=%v, want %v", tentative, tc.tentative)
			}
		})
	}
}

func TestDateSpecificTimesDoNotCrossProductDifferentDays(t *testing.T) {
	e := event.Event{Dates: []event.DateRange{{Start: "2026-11-07"}, {Start: "2026-11-08"}}, DateText: "11月7日10時、11月8日14時"}
	links := calendarLinks(e)
	if len(links) != 2 {
		t.Fatalf("invented date/time combinations: %+v", links)
	}
	for i, want := range []string{"20261107/20261108", "20261108/20261109"} {
		u, _ := url.Parse(links[i].URL)
		if u.Query().Get("dates") != want {
			t.Fatalf("ambiguous per-day time used: %s", u)
		}
	}
}
func TestCalendarLinksExtractTimeFromTitle(t *testing.T) {
	links := calendarLinks(event.Event{Title: "プレーパークを開催します 11/14(土)13：00～", Dates: []event.DateRange{{Start: "2026-11-14"}}, Venue: "府中公園", URL: "https://example.org/play", Summary: "案内本文"})
	if len(links) != 1 {
		t.Fatal(links)
	}
	u, _ := url.Parse(links[0].URL)
	if u.Query().Get("dates") != "20261114T130000/20261114T140000" || u.Query().Get("location") != "府中公園" || u.Query().Get("details") != "https://example.org/play" {
		t.Fatal(u)
	}
}

func TestCalendarLinkUsesSafeSourceURLAndOmitsUnknownLocation(t *testing.T) {
	e := event.Event{URL: "javascript:alert(1)", Venue: "会場不明", Summary: "不要な説明", Sources: []event.SourceRef{{Name: "公式", URL: "https://example.org/event"}}, Dates: []event.DateRange{{Start: "2026-11-07"}}}
	links := calendarLinks(e)
	u, _ := url.Parse(links[0].URL)
	if u.Query().Get("details") != "https://example.org/event" || u.Query().Has("location") {
		t.Fatal(u)
	}
}
