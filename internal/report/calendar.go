package report

import (
	"fmt"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"regexp"
	"sort"
	"strings"
	"time"
)

func eventSources(e event.Event) []event.SourceRef {
	var sources []event.SourceRef
	seen := map[string]bool{}
	for _, source := range e.Sources {
		source.URL = safeURL(source.URL)
		if source.URL != "" && seen[source.URL] {
			continue
		}
		seen[source.URL] = true
		sources = append(sources, source)
	}
	if len(sources) == 0 && safeURL(e.URL) != "" {
		sources = append(sources, event.SourceRef{Name: "公式情報", URL: e.URL})
	}
	return sources
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

type calendarItem struct {
	Target, Title, Time, Kind string
	Inferred                  bool
}
type calendarDay struct {
	ISO          string
	Number       int
	Other, Today bool
	Items        []calendarItem
}
type calendarMonth struct {
	Key, Label string
	Days       []calendarDay
	Weeks      []calendarWeek
}
type calendarWeek struct {
	Days []calendarDay
	Bars []calendarBar
}
type calendarBar struct {
	calendarItem
	Column, Span, Lane              int
	ContinuesBefore, ContinuesAfter bool
}
type detailEntry struct {
	Target, Kind string
	Event        event.Event
}
type calendarPage struct {
	event.Report
	Months            []calendarMonth
	Entries           []detailEntry
	HasEstimatedDates bool
}

func calendarView(r event.Report) calendarPage {
	v := calendarPage{Report: r}
	for gi, list := range [][]event.Event{r.Events, r.Information, r.Review} {
		kind := []string{"recommended", "seasonal", "pending"}[gi]
		for _, e := range list {
			v.Entries = append(v.Entries, detailEntry{fmt.Sprintf("detail-%d", len(v.Entries)), kind, e})
			v.HasEstimatedDates = v.HasEstimatedDates || e.DateNote != ""
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
					if eventOccursOn(entry.Event, d.ISO) {
						d.Items = append(d.Items, calendarItem{Target: entry.Target, Title: calendarTitle(entry.Event.Title), Time: entry.Event.TimeText, Kind: entry.Kind, Inferred: entry.Event.DateNote != ""})
					}
				}
			}
			m.Days = append(m.Days, d)
		}
		for i := 0; i < len(m.Days); i += 7 {
			days := m.Days[i : i+7]
			m.Weeks = append(m.Weeks, calendarWeek{Days: days, Bars: weekBars(days, v.Entries)})
		}
		v.Months = append(v.Months, m)
	}
	return v
}

func eventOccursOn(e event.Event, date string) bool {
	for _, d := range e.Dates {
		end := d.End
		if end == "" {
			end = d.Start
		}
		if _, err := time.Parse("2006-01-02", d.Start); err != nil {
			continue
		}
		if _, err := time.Parse("2006-01-02", end); err != nil {
			continue
		}
		if date >= d.Start && date <= end {
			return true
		}
	}
	return false
}

// weekBars joins consecutive visible dates of the same event, then packs bars
// into nonoverlapping lanes. Separate occurrences never bridge a missing day.
func weekBars(days []calendarDay, entries []detailEntry) []calendarBar {
	var bars []calendarBar
	for _, entry := range entries {
		for i := 0; i < len(days); {
			var item *calendarItem
			for j := range days[i].Items {
				if days[i].Items[j].Target == entry.Target {
					item = &days[i].Items[j]
					break
				}
			}
			if item == nil {
				i++
				continue
			}
			first := i
			for i++; i < len(days); i++ {
				found := false
				for _, candidate := range days[i].Items {
					if candidate.Target == entry.Target {
						found = true
						break
					}
				}
				if !found {
					break
				}
			}
			start, _ := time.Parse("2006-01-02", days[first].ISO)
			end, _ := time.Parse("2006-01-02", days[i-1].ISO)
			bars = append(bars, calendarBar{
				calendarItem: *item, Column: first + 1, Span: i - first,
				ContinuesBefore: eventOccursOn(entry.Event, start.AddDate(0, 0, -1).Format("2006-01-02")),
				ContinuesAfter:  eventOccursOn(entry.Event, end.AddDate(0, 0, 1).Format("2006-01-02")),
			})
		}
	}
	// Put longer events above single-day events and preserve source order on ties.
	sort.SliceStable(bars, func(i, j int) bool {
		if bars[i].Span != bars[j].Span {
			return bars[i].Span > bars[j].Span
		}
		return bars[i].Column < bars[j].Column
	})
	var occupied []uint8
	for i := range bars {
		bar := &bars[i]
		mask := uint8((1<<bar.Span)-1) << (bar.Column - 1)
		lane := 0
		for lane < len(occupied) && occupied[lane]&mask != 0 {
			lane++
		}
		if lane == len(occupied) {
			occupied = append(occupied, 0)
		}
		occupied[lane] |= mask
		bar.Lane = lane + 1
	}
	return bars
}

func calendarTitle(title string) string {
	return strings.TrimSpace(regexp.MustCompile(`^【[0-9０-９].*?開催】`).ReplaceAllString(title, ""))
}
