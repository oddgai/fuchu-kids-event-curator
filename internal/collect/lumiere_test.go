package collect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestLumierePublishedImageOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"status":"publish","link":"https://www.lumiere-fuchu.jp/event/434/","title":{"rendered":"【11月8日 開催】音楽会"},"content":{"rendered":"<p><img src='poster.jpg'></p>"}},{"status":"draft","link":"https://www.lumiere-fuchu.jp/event/9/","title":{"rendered":"非公開"},"content":{"rendered":"<p>非公開記事</p>"}}]`))
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
