package event

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestVerifiedEvidenceScopedToExactEvent(t *testing.T) {
	f := FeedbackSet{Reviews: []Feedback{{URL: "https://example.org/1", Title: "体験", Decision: "ng", EvidenceText: "対象4歳くらい〜"}}}
	v := ApplyFeedback([]Event{{URL: "https://example.org/1", Title: "体験"}, {URL: "https://example.org/1", Title: "別の体験"}, {URL: "https://other.example/1", Title: "体験"}}, f)
	if !strings.Contains(v[0].VerifiedText, "4歳") || v[1].VerifiedText != "" || v[2].VerifiedText != "" {
		t.Fatalf("%+v", v)
	}
	if Classify(v[0], 32).Category != "exclude" {
		t.Fatal("verified condition ignored")
	}
}
func TestVerdictIsNotRuntimeOverride(t *testing.T) {
	f := FeedbackSet{Reviews: []Feedback{{URL: "https://example.org/1", Title: "ファミリーコンサート", Decision: "ng"}}}
	v := ApplyFeedback([]Event{{URL: "https://example.org/1", Title: "ファミリーコンサート"}}, f)
	if Classify(v[0], 32).Category != "recommend" {
		t.Fatal("URL verdict used instead of rule")
	}
}

func TestOldVerifiedEvidenceIsNotReusedForever(t *testing.T) {
	f := FeedbackSet{ReviewedAt: "2026-11-05", Reviews: []Feedback{{URL: "https://example.org/1", Title: "体験", Decision: "ng", EvidenceText: "対象4歳くらい〜"}}}
	future, _ := time.Parse(time.RFC3339, "2027-05-01T10:00:00+09:00")
	v := ApplyFeedback([]Event{{URL: "https://example.org/1", Title: "体験", FetchedAt: future}}, f)
	if v[0].VerifiedText != "" {
		t.Fatal("expired evidence reused")
	}
}

func TestReviewedContentIsNotConfusedWithPublicationEligibility(t *testing.T) {
	f, err := LoadFeedback("../../feedback/reviews.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/review_events.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []Event
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	cases = ApplyFeedback(cases, f)
	if len(cases) != 29 || len(f.Reviews) != 29 {
		t.Fatal("review dataset incomplete")
	}
	for _, r := range f.Reviews {
		found := false
		for _, e := range cases {
			if e.URL != r.URL || e.Title != r.Title {
				continue
			}
			found = true
			// OK described the activity, including some old/full sessions, not its current availability.
			content := e
			content.Title = stripAvailability(e.Title)
			content.Reservation = ""
			got := classify(content, 32, false)
			if r.Decision == "ok" && got.Category != "recommend" {
				t.Errorf("positive content lost %s: %+v", r.Title, got)
			}
			if r.Decision == "ng" && got.Category == "recommend" {
				t.Errorf("rejected content recommended %s: %+v", r.Title, got)
			}
			if r.PreferredTreatment != "" && got.Category != "information" {
				t.Errorf("seasonal info not separate: %+v", got)
			}
		}
		if !found {
			t.Errorf("missing %s", r.Title)
		}
	}
}
