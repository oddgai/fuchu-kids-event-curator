package report

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"strings"
	"testing"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

func TestCalendarSpansWeeksWithoutBridgingMissingDays(t *testing.T) {
	r := event.Report{From: "2026-11-06", Until: "2026-11-09", Events: []event.Event{
		{Title: "連続開催", Dates: []event.DateRange{{Start: "2026-11-05", End: "2026-11-10"}}},
		{Title: "別々の日", Dates: []event.DateRange{{Start: "2026-11-07"}, {Start: "2026-11-09"}}},
	}}
	var b bytes.Buffer
	if err := WriteHTML(&b, r); err != nil {
		t.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(&b)
	if err != nil {
		t.Fatal(err)
	}
	bars := doc.Find(`.calendar-event[title="連続開催"]`)
	if bars.Length() != 2 {
		t.Fatalf("want one continuous bar per week, got %d", bars.Length())
	}
	for i, want := range []string{"grid-column:5 / span 3;grid-row:1", "grid-column:1 / span 1;grid-row:1"} {
		if got, _ := bars.Eq(i).Attr("style"); got != want {
			t.Errorf("segment %d: %q, want %q", i, got, want)
		}
		if got, _ := bars.Eq(i).Attr("data-open"); got != "detail-0" {
			t.Errorf("segment opens different details: %s", got)
		}
	}
	separate := doc.Find(`.calendar-event[title="別々の日"]`)
	if separate.Length() != 2 {
		t.Fatalf("separate days: %d", separate.Length())
	}
	separate.Each(func(_ int, bar *goquery.Selection) {
		style, _ := bar.Attr("style")
		if !strings.Contains(style, "span 1;") || !strings.Contains(style, "grid-row:2") {
			t.Errorf("nonconsecutive or overlapping event incorrectly placed: %s", style)
		}
	})
}

func TestCalendarSpansMonthBoundaryWithoutDuplicateBars(t *testing.T) {
	r := event.Report{From: "2026-11-30", Until: "2026-12-02", Information: []event.Event{
		{Title: "期間情報", Dates: []event.DateRange{
			{Start: "2026-11-30", End: "2026-12-02"},
			{Start: "2026-11-30", End: "2026-12-01"},
		}},
	}}
	var b bytes.Buffer
	if err := WriteHTML(&b, r); err != nil {
		t.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(&b)
	if err != nil {
		t.Fatal(err)
	}
	bars := doc.Find(".calendar-event.seasonal")
	if bars.Length() != 2 {
		t.Fatalf("overlapping ranges or neighboring-month cells duplicated bars: %d", bars.Length())
	}
	for i, want := range []string{"grid-column:1 / span 1;grid-row:1", "grid-column:2 / span 2;grid-row:1"} {
		if got, _ := bars.Eq(i).Attr("style"); got != want {
			t.Errorf("month segment %d: %s, want %s", i, got, want)
		}
	}
	if !bars.First().HasClass("continues-after") || bars.First().HasClass("continues-before") || !bars.Last().HasClass("continues-before") || bars.Last().HasClass("continues-after") {
		t.Fatal("month boundary continuation markers do not match actual duration")
	}
}

func TestHTMLPreservesUncertaintyAndSources(t *testing.T) {
	r := event.Report{GeneratedAt: time.Date(2026, 11, 5, 1, 2, 0, 0, time.UTC), From: "2026-11-05", Until: "2026-12-04", Events: []event.Event{{Title: "<script>alert(1)</script>", URL: "javascript:alert(2)", Sources: []event.SourceRef{{Name: "公式", URL: "https://example.org/event"}}, Reservation: "要予約", DeadlineText: "11月10日", NeedsReview: []string{"詳細要確認"}}}, Sources: []event.SourceResult{{Name: "失敗した情報源", Status: "error", Error: "通信失敗"}}}
	var b bytes.Buffer
	if err := WriteHTML(&b, r); err != nil {
		t.Fatal(err)
	}
	html := b.String()
	for _, want := range []string{"&lt;script&gt;", "https://example.org/event", "開催日未確定", "要予約", "11月10日", "詳細要確認", "失敗した情報源", "通信失敗", "2026-11-05", "2026-12-04", "10:02"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(html, "<script>alert(1)</script>") || strings.Contains(html, "javascript:") {
		t.Fatal("unsafe event content rendered")
	}
}
func TestHTMLEmpty(t *testing.T) {
	var b bytes.Buffer
	if err := WriteHTML(&b, event.Report{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "該当するイベントはありません") {
		t.Fatal("empty state missing")
	}
}

func TestHTMLExplainsStatusesAndHighlightsUnconfirmedEvents(t *testing.T) {
	r := event.Report{Events: []event.Event{{Title: "日程を確認する候補", AgeStatus: "unknown", Recommendation: "自然体験のキーワードを含む"}, {Title: "年齢条件あり", AgeStatus: "eligible", Dates: []event.DateRange{{Start: "2026-11-10"}}}}, Sources: []event.SourceResult{{Name: "A", Status: "success"}, {Name: "B", Status: "partial"}, {Name: "C", Status: "error"}, {Name: "D", Status: "disabled"}, {Name: "E", Status: "unexpected_internal_value"}}}
	var b bytes.Buffer
	if err := WriteHTML(&b, r); err != nil {
		t.Fatal(err)
	}
	html := b.String()
	for _, want := range []string{"取得済み", "一部取得", "取得失敗", "保留", "開催日未確定", "興味の補助情報", "自然体験のキーワードを含む"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing explanation %q", want)
		}
	}
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	doc.Find("script,style").Remove()
	visibleText := doc.Text()
	for _, technical := range []string{"unknown", "eligible", "success", "partial", "disabled", "unexpected_internal_value"} {
		if strings.Contains(visibleText, technical) {
			t.Errorf("technical status exposed: %s", technical)
		}
	}
	if strings.Count(html, "開催日未確定") != 2 {
		t.Error("known event incorrectly marked unconfirmed")
	}
}

func TestDeferredSourceDoesNotCreateClickableLink(t *testing.T) {
	var out bytes.Buffer
	err := WriteHTML(&out, event.Report{Sources: []event.SourceResult{{Name: "保留中の情報源", URL: "https://deferred.example.org/", Status: "disabled"}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `href="https://deferred.example.org/"`) {
		t.Fatal("deferred source became a clickable link")
	}
	if !strings.Contains(out.String(), "保留中の情報源") {
		t.Fatal("source status missing")
	}
}

func TestSeparateInformationAndReviewSections(t *testing.T) {
	var out bytes.Buffer
	r := event.Report{Information: []event.Event{{Title: "期間のイルミネーション", DateText: "11月〜12月", URL: "https://example.org/info"}}, Review: []event.Event{{Title: "日程未定の公園遊び", Assessment: &event.Assessment{Category: "recommend", Reasons: []string{"遊びの記載あり"}, Evidence: []string{"お子様用の遊具"}}}}}
	if err := WriteHTML(&out, r); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"期間限定のお出かけ情報", "要確認の候補", "期間のイルミネーション", "日程未定の公園遊び", "お子様用の遊具"} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("missing %s", text)
		}
	}
}

func TestCollectionErrorsAreReadableWithoutStackTraces(t *testing.T) {
	var b bytes.Buffer
	raw := `exception "Uncaught (in promise) Error: 詳細データの待機期限" at tick (<anonymous>:1:257)`
	r := event.Report{Warnings: []string{"観光協会: " + raw}, Sources: []event.SourceResult{{Name: "観光協会", Status: "partial", Error: raw}}}
	if err := WriteHTML(&b, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "at tick") || strings.Contains(b.String(), "anonymous") {
		t.Fatal("stack trace exposed")
	}
	if !strings.Contains(b.String(), "取得できなかった情報") {
		t.Fatal("failure not explained")
	}
}

func TestUnknownFieldsAreOmitted(t *testing.T) {
	var b bytes.Buffer
	r := event.Report{From: "2026-11-05", Until: "2026-11-30", Events: []event.Event{{Title: "親子の遊び", Venue: "会場不明", Fee: "不明", Dates: []event.DateRange{{Start: "2026-11-07"}}}}}
	if err := WriteHTML(&b, r); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"<dt>会場</dt>", "<dt>参加費</dt>", "<dt>時間</dt>", "<dt>対象年齢</dt>"} {
		if strings.Contains(b.String(), text) {
			t.Errorf("unknown field exposed %s", text)
		}
	}
	if !strings.Contains(b.String(), "Googleカレンダーに登録") {
		t.Fatal("registration link missing")
	}
}
