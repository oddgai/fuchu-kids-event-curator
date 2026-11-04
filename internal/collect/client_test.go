package collect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPRetriesAndLimits(t *testing.T) {
	n := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer s.Close()
	c := NewClient(0)
	b, _, err := c.Get(context.Background(), s.URL)
	if err != nil || string(b) != "ok" || n != 2 {
		t.Fatalf("%q %v attempts=%d", b, err, n)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := c.Get(ctx, s.URL); err == nil {
		t.Fatal("cancel ignored")
	}
}
func TestRobotsDisallowedCalendar(t *testing.T) {
	c := NewClient(time.Millisecond)
	if _, _, err := c.Get(context.Background(), "https://www.fuchu-cpf.or.jp/cgi-evt/event.cgi"); err == nil {
		t.Fatal("robots-disallowed CGI accepted")
	}
}
