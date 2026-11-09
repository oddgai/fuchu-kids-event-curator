package report

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

type CalendarLink struct{ URL, When, Note string }

type calendarTime struct {
	start, end int // Minutes after midnight; a provisional end can cross midnight.
	tentative  bool
}

func calendarLinks(e event.Event) []CalendarLink {
	var out []CalendarLink
	var times []calendarTime
	for _, text := range []string{e.TimeText, e.DateText, e.Title} {
		if !clockPattern.MatchString(normalizeClock(text)) {
			continue
		}
		dateMentions := regexp.MustCompile(`(?:[0-9]{1,2}月)?[0-9]{1,2}日|[0-9]{1,2}/[0-9]{1,2}`).FindAllString(normalizeClock(text), -1)
		if len(dateMentions) > 1 || len(e.Dates) > 1 && len(dateMentions) > 0 {
			// Without a structured association, do not combine one day's time
			// with another day's date. Keep date-specific prose for review.
			break
		}
		times = performanceTimes(text)
		break
	}
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
		add := func(dates, when, note string) {
			q := url.Values{"action": {"TEMPLATE"}, "text": {e.Title}, "dates": {dates}, "ctz": {"Asia/Tokyo"}, "stz": {"Asia/Tokyo"}, "etz": {"Asia/Tokyo"}}
			if u := eventURL(e); u != "" {
				q.Set("details", u)
			}
			if known(e.Venue) {
				q.Set("location", e.Venue)
			}
			if e.DateNote != "" {
				note = e.DateNote + " " + note
			}
			out = append(out, CalendarLink{URL: "https://calendar.google.com/calendar/r/eventedit?" + q.Encode(), When: when, Note: note})
		}
		if !start.Equal(end) || len(times) == 0 {
			add(start.Format("20060102")+"/"+end.AddDate(0, 0, 1).Format("20060102"), start.Format("1月2日"), "終日で登録画面を開きます。保存前に日時をご確認ください。")
			continue
		}
		for _, slot := range times {
			from := start.Add(time.Duration(slot.start) * time.Minute)
			until := start.Add(time.Duration(slot.end) * time.Minute)
			note := "日本時間の予定です。保存前に日時をご確認ください。"
			if slot.tentative {
				note = "終了時刻は仮（開始から1時間後）です。保存前に終了時刻をご確認ください。"
			}
			add(from.Format("20060102T150405")+"/"+until.Format("20060102T150405"), from.Format("1月2日 15:04")+"の回", note)
		}
	}
	return out
}

func eventURL(e event.Event) string {
	if u := safeURL(e.URL); u != "" {
		return u
	}
	for _, source := range e.Sources {
		if u := safeURL(source.URL); u != "" {
			return u
		}
	}
	return ""
}

func normalizeClock(text string) string {
	text = strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		return r
	}, text)
	return strings.ReplaceAll(text, "：", ":")
}

var clockPattern = regexp.MustCompile(`(?i)(午前|午後|am|pm)?\s*([0-9]{1,2})(?::([0-9]{2})|時(?:([0-9]{1,2})分?)?(半)?)(?:\s*(am|pm))?`)

func performanceTimes(text string) []calendarTime {
	text = normalizeClock(text)
	matches := clockPattern.FindAllStringSubmatchIndex(text, -1)
	type clock struct {
		begin, end, minute int
		period             string
		suffixPeriod       bool
	}
	var clocks []clock
	period := ""
	for _, m := range matches {
		get := func(i int) string {
			if m[2*i] < 0 {
				return ""
			}
			return text[m[2*i]:m[2*i+1]]
		}
		// Reject partial matches in malformed values such as 100:00 or 10:000.
		if m[4] > 0 && isDigit(text[m[4]-1]) || m[1] < len(text) && isDigit(text[m[1]]) {
			return nil
		}
		hour, _ := strconv.Atoi(get(2))
		minute, _ := strconv.Atoi(get(3) + get(4))
		if get(5) != "" {
			minute = 30
		}
		explicit := strings.ToLower(get(1) + get(6))
		if explicit != "" {
			period = explicit
		}
		if hour > 23 || minute > 59 || explicit != "" && hour > 12 {
			return nil
		}
		if period != "" && hour <= 12 {
			hour %= 12
			if period == "午後" || period == "pm" {
				hour += 12
			} else if period != "午前" && period != "am" {
				return nil
			}
		}
		clocks = append(clocks, clock{m[0], m[1], hour*60 + minute, period, get(6) != ""})
	}
	if len(clocks) == 0 {
		return nil
	}
	prefixLabels := regexp.MustCompile(`(?:開場|開演|開始|受付|終演|終了)\s*$`).MatchString(text[:clocks[0].begin])
	role := func(i int) string {
		previous, next := 0, len(text)
		if i > 0 {
			previous = clocks[i-1].end
		}
		if i+1 < len(clocks) {
			next = clocks[i+1].begin
		}
		pattern, labelText := `^(開場|受付|開演|開始|終演|終了)`, strings.TrimSpace(text[clocks[i].end:next])
		if prefixLabels {
			pattern, labelText = `(開場|受付|開演|開始|終演|終了)$`, strings.TrimSpace(text[previous:clocks[i].begin])
		}
		if m := regexp.MustCompile(pattern).FindStringSubmatch(labelText); len(m) > 0 {
			return m[1]
		}
		return ""
	}
	var out []calendarTime
	for i := 0; i < len(clocks); i++ {
		c := clocks[i]
		next := len(text)
		if i+1 < len(clocks) {
			next = clocks[i+1].begin
		}
		after := strings.TrimSpace(text[c.end:next])
		currentRole := role(i)
		doors := currentRole == "開場" || currentRole == "受付"
		if currentRole == "終演" || currentRole == "終了" {
			continue
		}
		if i+1 < len(clocks) && (regexp.MustCompile(`^[〜～~－–-]$`).MatchString(after) || role(i+1) == "終演" || role(i+1) == "終了") {
			until := clocks[i+1].minute
			if c.period == "" && clocks[i+1].suffixPeriod && c.minute/60 <= 12 {
				c.minute %= 12 * 60
				if clocks[i+1].period == "pm" {
					c.minute += 12 * 60
				}
			}
			if until <= c.minute {
				return nil
			}
			if !doors {
				out = append(out, calendarTime{start: c.minute, end: until})
			}
			i++
		} else if !doors {
			out = append(out, calendarTime{start: c.minute, end: c.minute + 60, tentative: true})
		}
	}
	return out
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
