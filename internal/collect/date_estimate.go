package collect

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

// estimateEventYear chooses the first occurrence on or after the page's update
// date, never the run date. Estimates remain labeled throughout the report.
func estimateEventYear(e *event.Event, updated time.Time) {
	if len(e.Dates) > 0 || updated.IsZero() || nonEventDateText(e.DateText) {
		return
	}
	updated = updated.In(time.FixedZone("JST", 9*60*60))
	updated = time.Date(updated.Year(), updated.Month(), updated.Day(), 0, 0, 0, 0, time.UTC)
	// A valid February 29 can be eight years away across a non-leap century.
	for year := updated.Year(); year <= updated.Year()+8; year++ {
		dates := parseDates(e.DateText, year)
		if len(dates) != 1 || dates[0].End != "" {
			continue
		}
		day, err := time.Parse("2006-01-02", dates[0].Start)
		if err != nil || day.Before(updated) {
			continue
		}
		weekdays := regexp.MustCompile(`[（(]([月火水木金土日])(?:曜日)?[）)]`).FindAllStringSubmatch(e.DateText, -1)
		if len(weekdays) > 1 || len(weekdays) == 1 && weekdays[0][1] != string([]rune("日月火水木金土")[day.Weekday()]) {
			// Do not move to a later year just to make the weekday agree.
			return
		}
		e.Dates = dates
		e.DateNote = fmt.Sprintf("開催年は推定：ページ日付%s以降で最初に来る%sとして仮表示しています。出典で開催年をご確認ください。", updated.Format("2006-01-02"), day.Format("2006-01-02"))
		// ParseDetail may have added this before a source-specific API provides
		// update evidence. Remove only the now-obsolete date warning.
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
