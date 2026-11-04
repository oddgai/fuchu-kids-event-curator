package event

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Select keeps possible events within the inclusive date and explicit age bounds.
// Unknown conditions remain visible for manual review. The input is never mutated.
func Select(events []Event, from, until string, ageMonths int) []Event {
	result := make([]Event, 0, len(events))
	positions := make(map[string]int)
	for _, original := range events {
		title := strings.TrimSpace(original.Title)
		if title == "雨天中止" || strings.HasPrefix(title, "【中止】") || strings.Contains(title, "開催中止のお知らせ") {
			continue
		}
		e := cloneEvent(original)
		schoolCondition := strings.TrimSpace(e.AgeText) == "小学生以上" || regexp.MustCompile(`未就学児(?:の)?入場不可`).MatchString(e.AgeText)
		uncertainSchool := false
		for _, word := range []string{"ただし", "未満も", "同伴", "例外", "除く", "推奨", "目安", "原則", "ではない", "ではありません"} {
			if strings.Contains(e.AgeText, word) {
				uncertainSchool = true
				break
			}
		}
		if schoolCondition {
			// A child under five has not reached school entry; from five onward
			// month age alone cannot establish the child's school stage.
			if ageMonths < 60 && !uncertainSchool {
				continue
			}
			e.NeedsReview = appendUnique(e.NeedsReview, "学校段階による参加条件・例外を要確認（月齢から就学を確定していません）")
		}
		overlaps, unknown := dateMatch(e.Dates, from, until)
		if !overlaps && !unknown {
			continue
		}
		if unknown {
			e.NeedsReview = appendUnique(e.NeedsReview, "開催日を要確認")
		}
		if e.MinAgeMonths != nil && ageMonths < *e.MinAgeMonths || e.MaxAgeMonths != nil && ageMonths > *e.MaxAgeMonths {
			continue
		}
		if e.MinAgeMonths == nil && e.MaxAgeMonths == nil {
			e.AgeStatus = "unknown"
			e.NeedsReview = appendUnique(e.NeedsReview, "対象年齢を要確認")
		} else {
			e.AgeStatus = "eligible"
		}
		e.Recommendation = ""
		for _, keyword := range []string{"工作", "遊び", "あそび", "絵本", "読み聞かせ", "音楽", "リトミック", "体験", "人形劇"} {
			if strings.Contains(e.Title+" "+e.Summary, keyword) {
				e.Recommendation = "原文に「" + keyword + "」の記載があります。"
				break
			}
		}
		key := eventKey(e)
		if index, ok := positions[key]; ok && key != "" {
			for _, source := range e.Sources {
				found := false
				for _, existing := range result[index].Sources {
					if existing == source {
						found = true
						break
					}
				}
				if !found {
					result[index].Sources = append(result[index].Sources, source)
				}
			}
			for _, reason := range e.NeedsReview {
				result[index].NeedsReview = appendUnique(result[index].NeedsReview, reason)
			}
			continue
		}
		if key != "" {
			positions[key] = len(result)
		}
		result = append(result, e)
	}
	return result
}

func dateMatch(dates []DateRange, from, until string) (bool, bool) {
	if len(dates) == 0 {
		return false, true
	}
	matched, unknown := false, false
	for _, d := range dates {
		end := d.End
		if end == "" {
			end = d.Start
		}
		startDate, errStart := time.Parse("2006-01-02", d.Start)
		endDate, errEnd := time.Parse("2006-01-02", end)
		if errStart != nil || errEnd != nil || endDate.Before(startDate) {
			unknown = true
			continue
		}
		if (until == "" || d.Start <= until) && (from == "" || end >= from) {
			matched = true
		}
	}
	return matched, unknown
}

func cloneEvent(e Event) Event {
	e.Sources = append([]SourceRef(nil), e.Sources...)
	e.Dates = append([]DateRange(nil), e.Dates...)
	e.NeedsReview = append([]string(nil), e.NeedsReview...)
	if e.MinAgeMonths != nil {
		v := *e.MinAgeMonths
		e.MinAgeMonths = &v
	}
	if e.MaxAgeMonths != nil {
		v := *e.MaxAgeMonths
		e.MaxAgeMonths = &v
	}
	return e
}

func appendUnique(values []string, value string) []string {
	for _, v := range values {
		if v == value {
			return values
		}
	}
	return append(values, value)
}

func eventKey(e Event) string {
	// Unknown dates cannot establish that two listings describe the same occurrence.
	if len(e.Dates) == 0 {
		return ""
	}
	dates := append([]DateRange(nil), e.Dates...)
	for i := range dates {
		if _, err := time.Parse("2006-01-02", dates[i].Start); err != nil {
			return ""
		}
		if dates[i].End == "" {
			dates[i].End = dates[i].Start
		}
	}
	sort.Slice(dates, func(i, j int) bool {
		if dates[i].Start == dates[j].Start {
			return dates[i].End < dates[j].End
		}
		return dates[i].Start < dates[j].Start
	})
	identity := strings.TrimSpace(e.Title)
	if parsed, err := url.Parse(e.URL); err == nil && parsed.Host != "" {
		parsed.Fragment = ""
		parsed.Host = strings.ToLower(parsed.Host)
		identity = parsed.String()
	}
	if identity == "" {
		return ""
	}
	key, _ := json.Marshal([]any{identity, strings.TrimSpace(e.Venue), dates, strings.TrimSpace(e.TimeText)})
	return string(key)
}
