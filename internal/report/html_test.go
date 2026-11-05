package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

func TestHTMLPreservesUncertaintyAndSources(t *testing.T) {
	r := event.Report{GeneratedAt: time.Date(2026, 11, 5, 1, 2, 0, 0, time.UTC), From: "2026-11-05", Until: "2026-12-04", Events: []event.Event{{Title: "<script>alert(1)</script>", URL: "javascript:alert(2)", Sources: []event.SourceRef{{Name: "公式", URL: "https://example.org/event"}}, Reservation: "要予約", DeadlineText: "11月10日", NeedsReview: []string{"詳細要確認"}}}, Sources: []event.SourceResult{{Name: "失敗した情報源", Status: "error", Error: "通信失敗"}}}
	var b bytes.Buffer
	if err := WriteHTML(&b, r); err != nil {
		t.Fatal(err)
	}
	html := b.String()
	for _, want := range []string{"&lt;script&gt;", "https://example.org/event", "対象年齢不明", "開催日不明", "要予約", "11月10日", "詳細要確認", "失敗した情報源", "通信失敗", "2026-11-05", "2026-12-04", "10:02"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(html, "<script>") || strings.Contains(html, "javascript:") {
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
	for _, want := range []string{"（要確認）", "明示された年齢条件の範囲内", "取得済み", "一部取得", "取得失敗", "保留", "開催日未確定", "興味の補助情報", "自然体験のキーワードを含む"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing explanation %q", want)
		}
	}
	for _, technical := range []string{"unknown", "eligible", "success", "partial", "disabled", "unexpected_internal_value"} {
		if strings.Contains(html, technical) {
			t.Errorf("technical status exposed: %s", technical)
		}
	}
	if strings.Count(html, "開催日未確定") != 1 {
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
