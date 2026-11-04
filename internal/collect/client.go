package collect

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

const userAgent = "fuchu-kids-event-curator/0.1 (+https://github.com/oddgai/fuchu-kids-event-curator)"
const maxBody = 4 << 20

type Client struct {
	HTTP     *http.Client
	Interval time.Duration
	last     map[string]time.Time
}

func NewClient(interval time.Duration) *Client {
	return &Client{HTTP: &http.Client{Timeout: 25 * time.Second}, Interval: interval, last: map[string]time.Time{}}
}
func (c *Client) Get(ctx context.Context, raw string) ([]byte, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, raw, err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, raw, fmt.Errorf("unsupported URL")
	}
	if u.Hostname() == "www.fuchu-cpf.or.jp" && strings.HasPrefix(u.Path, "/cgi-evt/") {
		return nil, raw, fmt.Errorf("robots.txt disallows /cgi-evt/")
	}
	// Restrict redirects to the original host; a redirect to another site needs explicit review.
	client := *c.HTTP
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Host != u.Host || len(via) >= 5 {
			return fmt.Errorf("unexpected redirect")
		}
		return nil
	}
	for attempt := 0; attempt < 2; attempt++ {
		wait := time.Until(c.last[u.Host].Add(c.Interval))
		if attempt > 0 && wait < time.Second {
			wait = time.Second
		}
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, raw, ctx.Err()
			case <-timer.C:
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
		if err != nil {
			return nil, raw, err
		}
		req.Header.Set("User-Agent", userAgent)
		c.last[u.Host] = time.Now()
		res, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, raw, ctx.Err()
			}
			if attempt == 0 {
				continue
			}
			return nil, raw, err
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
		res.Body.Close()
		if (res.StatusCode == 429 || res.StatusCode >= 500) && attempt == 0 {
			continue
		}
		if res.StatusCode != 200 {
			return nil, raw, fmt.Errorf("HTTP %d", res.StatusCode)
		}
		if readErr != nil {
			return nil, raw, readErr
		}
		if len(body) > maxBody {
			return nil, raw, fmt.Errorf("response exceeds %d bytes", maxBody)
		}
		if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "html") && ct != "" && !strings.Contains(ct, "text/plain") && !strings.Contains(ct, "application/json") {
			return nil, raw, fmt.Errorf("not HTML: %s", ct)
		}
		return body, res.Request.URL.String(), nil
	}
	return nil, raw, fmt.Errorf("request failed")
}

type Options struct {
	IDs                      []string
	MaxPages, MaxEvents      int
	From, Until, BrowserPath string
}

// Run is deliberately sequential: no concurrent requests to a source host.
func Run(ctx context.Context, c *Client, o Options) ([]event.Event, []event.SourceResult) {
	var events []event.Event
	var results []event.SourceResult
	wanted := map[string]bool{}
	for _, id := range o.IDs {
		wanted[id] = true
	}
	for _, s := range Sources() {
		if len(wanted) > 0 && !wanted[s.ID] {
			continue
		}
		now := time.Now()
		r := event.SourceResult{ID: s.ID, Name: s.Name, URL: s.URL, FetchedAt: now, Status: "success"}
		if s.DisabledReason != "" {
			r.Status = "disabled"
			r.Error = s.DisabledReason
			results = append(results, r)
			continue
		}

		if s.ID == "lumiere" {
			found, err := LumiereEvents(ctx, c, o.MaxPages, o.MaxEvents, now)
			events = append(events, found...)
			r.Count = len(found)
			if err != nil {
				r.Status = "partial"
				if len(found) == 0 {
					r.Status = "error"
				}
				r.Error = err.Error()
			}
			results = append(results, r)
			continue
		}
		var candidates []Candidate
		seen := map[string]bool{}
		pages := []string{s.URL}
		if s.ID == "machidukuri" { // Month selection is explicit; no dependence on the site's current month.
			start, _ := time.Parse("2006-01-02", o.From)
			end, _ := time.Parse("2006-01-02", o.Until)
			pages = nil
			for m := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(end); m = m.AddDate(0, 1, 0) {
				pages = append(pages, fmt.Sprintf("%s?yr=%d&month=%d&dy=1", s.URL, m.Year(), m.Month()))
			}
		}
		limit := o.MaxPages
		if limit <= 0 {
			limit = 2
		}
		maxEvents := o.MaxEvents
		if maxEvents <= 0 {
			maxEvents = 30
		}
		visited := map[string]bool{}
		for p := 0; len(pages) > 0 && p < limit; p++ {
			u := pages[0]
			pages = pages[1:]
			if visited[u] {
				continue
			}
			visited[u] = true
			b, actual, err := c.Get(ctx, u)
			if s.ID == "kankou" {
				b, err = BrowserHTML(ctx, u, o.BrowserPath, requestedMonths(o.From, o.Until))
				actual = u
			}
			if err != nil {
				r.Status = "partial"
				r.Error = err.Error()
				continue
			}
			list, next, err := ParseList(s, b, actual)
			if err != nil {
				r.Status = "partial"
				r.Error = err.Error()
				continue
			}
			for _, v := range list {
				if !seen[v.URL] {
					seen[v.URL] = true
					candidates = append(candidates, v)
				}
			}
			if next != "" && !visited[next] {
				pages = append(pages, next)
			}
		}
		if len(pages) > 0 {
			r.Status = "partial"
			r.Error = joinError(r.Error, "一覧ページ数の上限に達しました")
		}
		if len(candidates) > maxEvents {
			candidates = candidates[:maxEvents]
			r.Status = "partial"
			r.Error = joinError(r.Error, "詳細取得件数の上限に達しました")
		}
		for _, v := range candidates {
			if ranges := ParseDates(v.DateText); len(ranges) > 0 && o.From != "" && o.Until != "" {
				overlap := false
				for _, d := range ranges {
					end := d.End
					if end == "" {
						end = d.Start
					}
					if d.Start <= o.Until && end >= o.From {
						overlap = true
					}
				}
				if !overlap {
					continue
				}
			}
			b, _, err := c.Get(ctx, v.URL)
			if s.ID == "kankou" {
				b, err = BrowserHTML(ctx, v.URL, o.BrowserPath, nil)
			}
			if err != nil {
				r.Status = "partial"
				r.Error = joinError(r.Error, err.Error())
				continue
			}
			e, err := ParseDetail(s, v, b, now)
			if err != nil {
				r.Status = "partial"
				r.Error = joinError(r.Error, err.Error())
				continue
			}
			events = append(events, e)
			r.Count++
		}
		if r.Count == 0 && r.Status == "partial" {
			r.Status = "error"
		}
		results = append(results, r)
	}
	return events, results
}
func joinError(a, b string) string {
	if a == "" {
		return b
	}
	if strings.Contains(a, b) {
		return a
	}
	return a + " / " + b
}

func requestedMonths(from, until string) []string {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01-02", until)
	if err != nil {
		return nil
	}
	var out []string
	for m := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(end); m = m.AddDate(0, 1, 0) {
		out = append(out, m.Format("2006-01"))
	}
	return out
}
