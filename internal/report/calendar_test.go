package report

import (
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"net/url"
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

func TestRegistrationDoesNotInventPerformanceEnd(t *testing.T) {
	for _, tc := range []struct{ text, want string }{{"11:00開演・14:00開演", "20261107/20261108"}, {"9時〜16時", "20261107T090000/20261107T160000"}, {"10:00〜11:00 / 14:00〜15:00", "20261107/20261108"}} {
		links := calendarLinks(event.Event{Title: "公演", TimeText: tc.text, Dates: []event.DateRange{{Start: "2026-11-07"}}})
		u, _ := url.Parse(links[0].URL)
		if u.Query().Get("dates") != tc.want {
			t.Fatalf("%s: %s", tc.text, u.Query().Get("dates"))
		}
	}
}

func TestAmbiguousMeridiemNeverBecomesEarlyMorning(t *testing.T) {
	for _, text := range []string{"午後1時〜3時", "午後1:00〜3:00", "6:00 PM〜8:00 PM", "100:00〜11:00", "10:00〜11:00 / 14:00開演"} {
		links := calendarLinks(event.Event{TimeText: text, Dates: []event.DateRange{{Start: "2026-11-07"}}})
		u, _ := url.Parse(links[0].URL)
		if u.Query().Get("dates") != "20261107/20261108" {
			t.Fatalf("invented time %s: %s", text, u.Query().Get("dates"))
		}
	}
}
