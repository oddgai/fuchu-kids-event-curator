package event

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Feedback struct {
	EventID            string `json:"event_id"`
	Title              string `json:"title"`
	URL                string `json:"url"`
	Decision           string `json:"decision"`
	Reason             string `json:"reason"`
	EvidenceText       string `json:"evidence_text"`
	PreferredTreatment string `json:"preferred_treatment"`
}
type FeedbackSet struct {
	Reviews        []Feedback `json:"reviews"`
	ReviewedAt     string     `json:"reviewed_at"`
	ChildAgeMonths int        `json:"child_age_months"`
}

func LoadFeedback(path string) (FeedbackSet, error) {
	var f FeedbackSet
	file, err := os.Open(path)
	if err != nil {
		return f, err
	}
	defer file.Close()
	d := json.NewDecoder(io.LimitReader(file, 2<<20))
	if err := d.Decode(&f); err != nil {
		return f, err
	}
	for _, r := range f.Reviews {
		if r.Title == "" || r.URL == "" || (r.Decision != "ok" && r.Decision != "ng") {
			return f, fmt.Errorf("フィードバックのタイトル・URL・評価を確認してください")
		}
	}
	return f, nil
}

// ApplyFeedback copies only confirmed evidence; an OK/NG label never overrides a rule.
func ApplyFeedback(events []Event, f FeedbackSet) []Event {
	out := make([]Event, len(events))
	for i, e := range events {
		out[i] = cloneEvent(e)
		for _, r := range f.Reviews {
			if e.URL == r.URL && e.Title == r.Title && r.EvidenceText != "" {
				if reviewed, err := time.Parse("2006-01-02", f.ReviewedAt); err == nil && !e.FetchedAt.IsZero() && e.FetchedAt.After(reviewed.AddDate(0, 0, 90)) {
					out[i].NeedsReview = appendUnique(out[i].NeedsReview, "以前の人手確認から90日を超えています。条件を再確認してください")
					continue
				}
				out[i].VerifiedText = r.EvidenceText
				out[i].NeedsReview = appendUnique(out[i].NeedsReview, "参加条件の補足に人手確認済みの根拠を使用しています")
			}
		}
	}
	return out
}

type EvaluationRow struct {
	Title    string `json:"title"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Match    bool   `json:"match"`
}
type Evaluation struct {
	AgeMonths int             `json:"age_months"`
	Total     int             `json:"total"`
	Matched   int             `json:"matched"`
	Missing   int             `json:"missing"`
	Rows      []EvaluationRow `json:"rows"`
	Note      string          `json:"note"`
}

func EvaluateFeedback(events []Event, f FeedbackSet, age int) Evaluation {
	if f.ChildAgeMonths > 0 {
		age = f.ChildAgeMonths
	}
	result := Evaluation{AgeMonths: age, Note: "内容への評価の回帰比較です。開催期間・満員などの掲載可否と、未知イベントの精度とは異なります。"}
	events = ApplyFeedback(events, f)
	for _, r := range f.Reviews {
		result.Total++
		found := false
		for _, e := range events {
			if e.URL != r.URL || e.Title != r.Title {
				continue
			}
			found = true
			expected := r.Decision
			if r.PreferredTreatment != "" {
				expected = "information"
			}
			actual := classify(e, age, false).Category
			switch actual {
			case "recommend":
				actual = "ok"
			case "exclude":
				actual = "ng"
			}
			row := EvaluationRow{Title: r.Title, Expected: expected, Actual: actual, Match: expected == actual}
			result.Rows = append(result.Rows, row)
			if row.Match {
				result.Matched++
			}
			break
		}
		if !found {
			result.Missing++
			result.Rows = append(result.Rows, EvaluationRow{Title: r.Title, Expected: r.Decision, Actual: "missing"})
		}
	}
	return result
}
