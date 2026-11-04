// Package event defines the shared collection and report format.
package event

import "time"

// DateRange uses inclusive ISO dates in Asia/Tokyo. Empty values are unknown.
type DateRange struct {
	Start string `json:"start"`
	End   string `json:"end,omitempty"`
}

type SourceRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Event struct {
	ID             string      `json:"id"`
	Title          string      `json:"title"`
	URL            string      `json:"url"`
	Sources        []SourceRef `json:"sources"`
	Dates          []DateRange `json:"dates,omitempty"`
	DateText       string      `json:"date_text,omitempty"`
	TimeText       string      `json:"time_text,omitempty"`
	Venue          string      `json:"venue,omitempty"`
	AgeText        string      `json:"age_text,omitempty"`
	MinAgeMonths   *int        `json:"min_age_months,omitempty"`
	MaxAgeMonths   *int        `json:"max_age_months,omitempty"`
	AgeStatus      string      `json:"age_status"`
	Fee            string      `json:"fee,omitempty"`
	Reservation    string      `json:"reservation,omitempty"`
	DeadlineText   string      `json:"deadline_text,omitempty"`
	Summary        string      `json:"summary,omitempty"`
	Recommendation string      `json:"recommendation,omitempty"`
	NeedsReview    []string    `json:"needs_review,omitempty"`
	FetchedAt      time.Time   `json:"fetched_at"`
}

type SourceResult struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Status    string    `json:"status"` // success, partial, error, disabled
	Count     int       `json:"count"`
	Error     string    `json:"error,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
}

type Report struct {
	GeneratedAt time.Time      `json:"generated_at"`
	From        string         `json:"from"`
	Until       string         `json:"until"`
	AgeMonths   int            `json:"age_months"`
	Events      []Event        `json:"events"`
	Sources     []SourceResult `json:"sources"`
	Warnings    []string       `json:"warnings,omitempty"`
}
