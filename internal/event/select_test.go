package event

import (
	"reflect"
	"strings"
	"testing"
)

func TestSelectDates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dates  []DateRange
		want   int
		review bool
	}{
		{"start boundary", []DateRange{{Start: "2026-11-05"}}, 1, false},
		{"end boundary", []DateRange{{Start: "2026-12-04"}}, 1, false},
		{"spans window", []DateRange{{Start: "2026-10-01", End: "2027-01-01"}}, 1, false},
		{"outside", []DateRange{{Start: "2026-11-04"}}, 0, false},
		{"separate dates", []DateRange{{Start: "2026-11-04"}, {Start: "2026-12-05"}}, 0, false},
		{"unknown", nil, 1, true},
		{"invalid", []DateRange{{Start: "未定"}}, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Select([]Event{{Title: "催し", Dates: tc.dates}}, "2026-11-05", "2026-12-04", 32)
			if len(got) != tc.want {
				t.Fatalf("got %d events", len(got))
			}
			if tc.review && len(got[0].NeedsReview) == 0 {
				t.Fatal("missing date review")
			}
		})
	}
}

func TestSelectAgeAndRecommendation(t *testing.T) {
	ptr := func(n int) *int { return &n }
	for _, tc := range []struct {
		name     string
		min, max *int
		want     int
		status   string
	}{
		{"unknown", nil, nil, 1, "unknown"}, {"minimum boundary", ptr(32), nil, 1, "eligible"},
		{"maximum boundary", nil, ptr(32), 1, "eligible"}, {"too young", ptr(33), nil, 0, ""}, {"too old", nil, ptr(31), 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Select([]Event{{Title: "親子の工作", MinAgeMonths: tc.min, MaxAgeMonths: tc.max}}, "2026-11-05", "2026-12-04", 32)
			if len(got) != tc.want {
				t.Fatalf("got %d", len(got))
			}
			if tc.want == 0 {
				return
			}
			if got[0].AgeStatus != tc.status {
				t.Fatalf("status %q", got[0].AgeStatus)
			}
			if tc.status == "unknown" && len(got[0].NeedsReview) < 2 {
				t.Fatal("age review missing")
			}
			if !strings.Contains(got[0].Recommendation, "工作") || strings.Contains(got[0].Recommendation, "参加可能") {
				t.Fatalf("recommendation %q", got[0].Recommendation)
			}
		})
	}
}

func TestSelectConservativeDedupAndCopy(t *testing.T) {
	base := Event{Title: "工作", URL: "https://example.jp/event#info", Venue: "会館", Dates: []DateRange{{Start: "2026-11-06"}}, TimeText: "10:00", Sources: []SourceRef{{Name: "A", URL: "https://a.jp"}}, NeedsReview: []string{"原文確認"}}
	duplicate := base
	duplicate.URL = "https://example.jp/event"
	duplicate.Sources = []SourceRef{{Name: "B", URL: "https://b.jp"}}
	otherDay := base
	otherDay.Dates = []DateRange{{Start: "2026-11-07"}}
	otherVenue := base
	otherVenue.Venue = "別会館"
	otherTime := base
	otherTime.TimeText = "14:00"
	input := []Event{base, duplicate, otherDay, otherVenue, otherTime}
	got := Select(input, "2026-11-05", "2026-12-04", 32)
	if len(got) != 4 {
		t.Fatalf("got %d", len(got))
	}
	if len(got[0].Sources) != 2 {
		t.Fatalf("sources %v", got[0].Sources)
	}
	got[0].Sources[0].Name = "changed"
	got[0].Dates[0].Start = "changed"
	got[0].NeedsReview[0] = "changed"
	if !reflect.DeepEqual(input[0], base) || base.Sources[0].Name != "A" || base.Dates[0].Start != "2026-11-06" || base.NeedsReview[0] != "原文確認" {
		t.Fatal("input mutated")
	}
}

func TestSelectWithoutURLDeduplicatesOnlyMatchingOccurrences(t *testing.T) {
	events := []Event{
		{Title: "絵本", Venue: "図書館", Dates: []DateRange{{Start: "2026-11-06"}, {Start: "2026-11-08"}}, Sources: []SourceRef{{Name: "A", URL: "https://a.jp"}}},
		{Title: "絵本", Venue: "図書館", Dates: []DateRange{{Start: "2026-11-08"}, {Start: "2026-11-06"}}, Sources: []SourceRef{{Name: "B", URL: "https://b.jp"}}},
		{Title: "絵本", Venue: "図書館", Dates: []DateRange{{Start: "2026-11-07"}}},
	}
	got := Select(events, "2026-11-05", "2026-12-04", 32)
	if len(got) != 2 || len(got[0].Sources) != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestSelectExcludesExplicitCancellationNotWeatherConditions(t *testing.T) {
	for _, tc := range []struct {
		title string
		want  int
	}{
		{"雨天中止", 0}, {"【中止】親子工作会", 0}, {"開催中止のお知らせ", 0}, {"親子工作会 開催中止のお知らせ", 0},
		{"親子工作会（雨天中止）", 1}, {"雨天中止の場合があります", 1}, {"中止の場合の連絡について", 1},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got := Select([]Event{{Title: tc.title}}, "2026-11-05", "2026-12-04", 32)
			if len(got) != tc.want {
				t.Fatalf("got %d events, want %d", len(got), tc.want)
			}
		})
	}
}

func TestSelectExplicitSchoolRestrictionsConservatively(t *testing.T) {
	for _, tc := range []struct {
		ageText   string
		age, want int
	}{
		{"未就学児入場不可", 32, 0}, {"未就学児の入場不可。", 32, 0}, {"小学生以上", 32, 0},
		{"未就学児入場不可", 59, 0}, {"未就学児入場不可", 60, 1}, {"小学生以上", 72, 1},
		{"未就学児入場不可。ただし保護者同伴で可", 32, 1}, {"小学生以上（未満も参加可）", 32, 1},
		{"未就学児も入場可", 32, 1},
	} {
		t.Run(tc.ageText, func(t *testing.T) {
			got := Select([]Event{{Title: "公演", AgeText: tc.ageText, Dates: []DateRange{{Start: "2026-11-05"}}}}, "2026-11-05", "2026-12-04", tc.age)
			if len(got) != tc.want {
				t.Fatalf("got %d want %d", len(got), tc.want)
			}
			if tc.want == 1 && (got[0].AgeStatus != "unknown" || len(got[0].NeedsReview) == 0) {
				t.Fatalf("school condition not marked unknown: %+v", got[0])
			}
		})
	}
}
