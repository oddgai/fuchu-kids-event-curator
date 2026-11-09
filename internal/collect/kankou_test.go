package collect

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestKankouList(t *testing.T) {
	b := []byte(`<div id="mitte-event-list"><ul class="content-container"><li><div class="photo-list"><div class="block-text"><h4>光の祭り</h4><a class="atag" href="?p=we-page-event-entry&amp;event=42"></a><p class="event-date">2026.11.19 - 2026.12.25</p></div></div></li></ul></div>`)
	got, next, err := ParseKankouList(b, "https://www.kankou-fuchu.com/")
	if err != nil || len(got) != 1 || next != "" || got[0].Title != "光の祭り" {
		t.Fatalf("%+v %s %v", got, next, err)
	}
	if _, _, err := ParseKankouList([]byte(`<div id="mitte-event-list"></div>`), "https://www.kankou-fuchu.com/"); err == nil {
		t.Fatal("empty shell accepted")
	}
}
func TestKankouDetailScopesRequestedID(t *testing.T) {
	b := []byte(`<div id="mitte-event-entry"><ul class="content-container"><li id="event-entry1"><h3>別記事</h3><div class="photo-list"><div class="block-text"><table><tr><th>対象</th><td>小学生</td></tr></table></div></div></li><li id="event-entry42" class="active"><div class="we-title"><h3>光の祭り</h3></div><div class="photo-list"><div class="block-text"><table><tr><th>会場</th><td>駅前</td></tr><tr><th>開催期間（開始日）</th><td>2026/11/19</td></tr><tr><th>開催期間（終了日）</th><td>2026/12/25</td></tr></table></div></div></li></ul></div>`)
	got, err := ParseKankouDetail(Source{ID: "kankou", Name: "観光協会"}, Candidate{URL: "https://www.kankou-fuchu.com/?event=42"}, b, time.Now())
	if err != nil || got.Title != "光の祭り" || got.Venue != "駅前" || got.AgeText != "" || got.AgeStatus != "unknown" || len(got.Dates) != 1 || got.Dates[0].End != "2026-12-25" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestKankouLiveBrowser(t *testing.T) {
	if os.Getenv("FUCHU_KANKOU_LIVE") != "1" {
		t.Skip("explicit live check only")
	}
	b, err := BrowserHTML(context.Background(), Sources()[5].URL, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", []string{"2026-11", "2026-12"})
	if err != nil {
		t.Fatal(err)
	}
	list, _, err := ParseKankouList(b, Sources()[5].URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("list candidates: %d", len(list))
	b, err = BrowserHTML(context.Background(), "https://www.kankou-fuchu.com/?p=we-page-event-entry&event=400214&cat=17414&type=event", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseKankouDetail(Sources()[5], Candidate{URL: "https://www.kankou-fuchu.com/?event=400214"}, b, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got.Venue == "" || len(got.Dates) == 0 {
		t.Fatalf("%+v", got)
	}
	t.Logf("detail: %s %+v %s", got.Title, got.Dates, got.Venue)
}
