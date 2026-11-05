package report

import (
	"fmt"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CalendarLink struct{ URL, Label, Note string }

func calendarLinks(e event.Event) []CalendarLink {
	var out []CalendarLink
	for _, d := range e.Dates {
		start, err := time.Parse("2006-01-02", d.Start)
		if err != nil {
			continue
		}
		end := start
		if d.End != "" {
			end, err = time.Parse("2006-01-02", d.End)
			if err != nil || end.Before(start) {
				continue
			}
		}
		dates := start.Format("20060102") + "/" + end.AddDate(0, 0, 1).Format("20060102")
		note := "終日で登録画面を開きます。時刻や休館日を確認してください。"
		text := strings.NewReplacer("：", ":", "時", ":", "分", "").Replace(e.TimeText)
		text = strings.Map(func(r rune) rune {
			if r >= '０' && r <= '９' {
				return r - '０' + '0'
			}
			return r
		}, text)
		re := regexp.MustCompile(`(?:^|[^0-9])([0-9]{1,2}):([0-9]{2})?\s*[〜～~－–-]\s*([0-9]{1,2}):([0-9]{2})?(?:$|[^0-9])`)
		matches := re.FindAllStringSubmatch(text, -1)
		if start.Equal(end) && len(matches) == 1 && len(regexp.MustCompile(`[0-9]{1,2}:`).FindAllString(text, -1)) == 2 && !regexp.MustCompile(`(?i)(午前|午後|am|pm)`).MatchString(text) {
			h1, _ := strconv.Atoi(matches[0][1])
			m1, _ := strconv.Atoi(matches[0][2])
			h2, _ := strconv.Atoi(matches[0][3])
			m2, _ := strconv.Atoi(matches[0][4])
			if h1 < 24 && h2 < 24 && m1 < 60 && m2 < 60 && h2*60+m2 > h1*60+m1 {
				dates = fmt.Sprintf("%sT%02d%02d00/%sT%02d%02d00", start.Format("20060102"), h1, m1, end.Format("20060102"), h2, m2)
				note = "日本時間の予定です。保存前に日時をご確認ください。"
			}
		}
		q := url.Values{"action": {"TEMPLATE"}, "text": {e.Title}, "dates": {dates}, "ctz": {"Asia/Tokyo"}, "stz": {"Asia/Tokyo"}, "etz": {"Asia/Tokyo"}}
		details := e.Summary
		if safeURL(e.URL) != "" {
			details += "\n出典: " + e.URL
		}
		details += "\n" + e.DateText + "\n" + e.TimeText
		q.Set("details", details)
		if known(e.Venue) {
			q.Set("location", e.Venue)
		}
		out = append(out, CalendarLink{URL: "https://calendar.google.com/calendar/r/eventedit?" + q.Encode(), Label: start.Format("1月2日") + "の予定を登録", Note: note})
	}
	return out
}
func known(s string) bool {
	v := strings.TrimSpace(s)
	if v == "" {
		return false
	}
	for _, x := range []string{"不明", "未記載", "記載なし", "要確認"} {
		if v == x || strings.HasPrefix(v, x+"（") {
			return false
		}
	}
	return !strings.HasSuffix(v, "不明")
}

type calendarItem struct{ Target, Title, Time, Kind string }
type calendarDay struct {
	ISO          string
	Number       int
	Other, Today bool
	Items        []calendarItem
}
type calendarMonth struct {
	Key, Label string
	Days       []calendarDay
}
type detailEntry struct {
	Target, Kind string
	Event        event.Event
}
type calendarPage struct {
	event.Report
	Months  []calendarMonth
	Entries []detailEntry
}

func calendarView(r event.Report) calendarPage {
	v := calendarPage{Report: r}
	for gi, list := range [][]event.Event{r.Events, r.Information, r.Review} {
		kind := []string{"recommended", "seasonal", "pending"}[gi]
		for _, e := range list {
			v.Entries = append(v.Entries, detailEntry{fmt.Sprintf("detail-%d", len(v.Entries)), kind, e})
		}
	}
	start, err := time.Parse("2006-01-02", r.From)
	if err != nil {
		start = r.GeneratedAt
		if start.IsZero() {
			start = time.Now()
		}
	}
	until, err := time.Parse("2006-01-02", r.Until)
	if err != nil || until.Before(start) {
		until = start
	}
	if until.After(start.AddDate(1, 0, 0)) {
		until = start.AddDate(1, 0, 0)
	}
	today := time.Now().In(time.FixedZone("JST", 9*3600)).Format("2006-01-02")
	for month := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !month.After(until); month = month.AddDate(0, 1, 0) {
		m := calendarMonth{Key: month.Format("2006-01"), Label: month.Format("2006年1月")}
		offset := (int(month.Weekday()) + 6) % 7
		first := month.AddDate(0, 0, -offset)
		last := month.AddDate(0, 1, -1)
		cells := ((offset + last.Day() + 6) / 7) * 7
		for i := 0; i < cells; i++ {
			date := first.AddDate(0, 0, i)
			d := calendarDay{ISO: date.Format("2006-01-02"), Number: date.Day(), Other: date.Month() != month.Month(), Today: date.Format("2006-01-02") == today}
			if !d.Other && d.ISO >= r.From && (r.Until == "" || d.ISO <= r.Until) {
				for _, entry := range v.Entries {
					if entry.Kind == "pending" {
						continue
					}
					for _, rangeDate := range entry.Event.Dates {
						end := rangeDate.End
						if end == "" {
							end = rangeDate.Start
						}
						if d.ISO >= rangeDate.Start && d.ISO <= end {
							d.Items = append(d.Items, calendarItem{entry.Target, entry.Event.Title, entry.Event.TimeText, entry.Kind})
							break
						}
					}
				}
			}
			m.Days = append(m.Days, d)
		}
		v.Months = append(v.Months, m)
	}
	return v
}
