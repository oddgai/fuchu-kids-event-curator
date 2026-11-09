package collect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLumierePublishedImageOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"status":"publish","date":"2026-10-15T09:00:01","link":"https://www.lumiere-fuchu.jp/event/434/","title":{"rendered":"【11月8日 開催】音楽会"},"content":{"rendered":"<p><img src='poster.jpg'></p>"}},{"status":"draft","link":"https://www.lumiere-fuchu.jp/event/9/","title":{"rendered":"非公開"},"content":{"rendered":"<p>非公開記事</p>"}}]`))
	}))
	defer srv.Close()
	c := NewClient(0)
	c.HTTP = &http.Client{Transport: lumiereRewriteTransport{base: srv.URL, inner: http.DefaultTransport}}
	got, err := LumiereEvents(context.Background(), c, 2, 30, time.Now())
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	if len(got[0].Dates) != 0 || got[0].AgeStatus != "unknown" || len(got[0].NeedsReview) == 0 {
		t.Fatalf("%+v", got[0])
	}
}

func TestLumiereEstimatesTitleYearFromModificationDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"status":"publish","date":"2025-10-15T09:00:01","modified":"2026-10-30T09:00:01","link":"https://www.lumiere-fuchu.jp/event/434/","title":{"rendered":"【11月8日 開催】ファミリーコンサート"},"content":{"rendered":"<p><img src='poster.jpg'></p>"}}]`))
	}))
	defer srv.Close()
	c := NewClient(0)
	c.HTTP = &http.Client{Transport: lumiereRewriteTransport{base: srv.URL, inner: http.DefaultTransport}}
	got, err := LumiereEvents(context.Background(), c, 2, 30, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	e := got[0]
	if len(e.Dates) != 1 || e.Dates[0].Start != "2026-11-08" || !strings.Contains(e.DateNote, "2026-10-30") {
		t.Fatalf("lost title date or modification evidence: %+v", e)
	}
	for _, note := range e.NeedsReview {
		if strings.Contains(note, "掲載日から補完していません") {
			t.Fatal("contradictory date warning retained")
		}
	}
}

type lumiereRewriteTransport struct {
	base  string
	inner http.RoundTripper
}

func (t lumiereRewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	u, _ := url.Parse(t.base)
	copy := r.Clone(r.Context())
	copy.URL.Host = u.Host
	copy.URL.Scheme = u.Scheme
	res, err := t.inner.RoundTrip(copy)
	if res != nil {
		res.Request = r
	}
	return res, err
}
