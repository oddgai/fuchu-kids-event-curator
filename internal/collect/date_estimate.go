package collect

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

// estimateEventYear uses the announcement's publication date, never the run
// date. Only a single upcoming day within six months is eligible. This remains
// an explicit estimate throughout JSON, calendar display and registration.
func estimateEventYear(e *event.Event, published time.Time) {
	if len(e.Dates) > 0 || published.IsZero() || nonEventDateText(e.DateText) {
		return
	}
	published = time.Date(published.Year(), published.Month(), published.Day(), 0, 0, 0, 0, time.UTC)
	for _, year := range []int{published.Year(), published.Year() + 1} {
		dates := parseDates(e.DateText, year)
		if len(dates) != 1 || dates[0].End != "" {
			continue
		}
		day, err := time.Parse("2006-01-02", dates[0].Start)
		if err != nil || day.Before(published) || day.After(published.AddDate(0, 0, 183)) {
			continue
		}
		weekdays := regexp.MustCompile(`[（(]([月火水木金土日])(?:曜日)?[）)]`).FindAllStringSubmatch(e.DateText, -1)
		if len(weekdays) > 1 || len(weekdays) == 1 && weekdays[0][1] != string([]rune("日月火水木金土")[day.Weekday()]) {
			continue
		}
		e.Dates = dates
		e.DateNote = fmt.Sprintf("開催年は推定：掲載日%sをもとに%d年として仮表示しています。出典で開催年をご確認ください。", published.Format("2006-01-02"), day.Year())
		// ParseDetail may have added this before a source-specific API provides
		// publication evidence. Remove only the now-obsolete date warning.
		var remaining []string
		for _, note := range e.NeedsReview {
			if !strings.HasPrefix(note, "開催年・日程を原文で確認してください（掲載日から補完していません）") {
				remaining = append(remaining, note)
			}
		}
		e.NeedsReview = remaining
		return
	}
}

func nonEventDateText(text string) bool {
	return regexp.MustCompile(`締め?切り?|〆切|必着|(?:申込|申し込み|受付|予約|募集)[^。\n]{0,20}(?:開始|期間|期限)`).MatchString(text)
}
