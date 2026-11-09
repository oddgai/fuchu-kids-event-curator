package collect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

// LumiereEvents reads only published posts exposed by the public WordPress API.
// Publication timestamps are only evidence for explicitly labeled year estimates;
// the event's month/day always comes from the announcement.
func LumiereEvents(ctx context.Context, c *Client, maxPages, maxEvents int, now time.Time) ([]event.Event, error) {
	if maxPages <= 0 {
		maxPages = 2
	}
	if maxEvents <= 0 {
		maxEvents = 30
	}
	const endpoint = "https://www.lumiere-fuchu.jp/wp-json/wp/v2/event"
	type rendered struct {
		Rendered string `json:"rendered"`
	}
	type wpPost struct {
		Date    string   `json:"date"`
		Status  string   `json:"status"`
		Link    string   `json:"link"`
		Title   rendered `json:"title"`
		Content rendered `json:"content"`
	}
	source := Source{ID: "", Name: "ルミエール府中", URL: "https://www.lumiere-fuchu.jp/event/"}
	var out []event.Event
	seen := map[string]bool{}
	for page := 1; page <= maxPages; page++ {
		raw := fmt.Sprintf("%s?status=publish&per_page=100&page=%d", endpoint, page)
		b, _, err := c.Get(ctx, raw)
		if err != nil {
			return out, err
		}
		var posts []wpPost
		if err := json.Unmarshal(b, &posts); err != nil {
			return out, fmt.Errorf("公開イベントJSON: %w", err)
		}
		for _, post := range posts {
			if post.Status != "publish" {
				continue
			}
			link := absolute(source.URL, post.Link)
			if link == "" || seen[link] {
				continue
			}
			titleDoc, err := goquery.NewDocumentFromReader(strings.NewReader(post.Title.Rendered))
			if err != nil {
				return out, err
			}
			title := clean(titleDoc.Text())
			if title == "" {
				continue
			}
			seen[link] = true
			if len(out) >= maxEvents {
				return out, fmt.Errorf("詳細取得件数の上限に達しました")
			}
			normalized := []byte("<main><h1>" + html.EscapeString(title) + "</h1>" + post.Content.Rendered + "</main>")
			e, err := ParseDetail(source, Candidate{Title: title, URL: link, DateText: title}, normalized, now)
			if err != nil {
				return out, err
			}
			if published, err := time.Parse("2006-01-02T15:04:05", post.Date); err == nil {
				estimateEventYear(&e, published)
			}
			// An image-only announcement still has a useful source link and unknown conditions.
			d, _ := goquery.NewDocumentFromReader(bytes.NewReader([]byte(post.Content.Rendered)))
			if clean(d.Text()) == "" {
				e.Summary = ""
				e.NeedsReview = append(e.NeedsReview, "告知本文は画像中心です。日時・参加条件は出典で確認してください")
			}
			out = append(out, e)
		}
		if len(posts) < 100 {
			return out, nil
		}
	}
	return out, fmt.Errorf("一覧ページ数の上限に達しました")
}
