package collect

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestSeisekiSavedHTML(t *testing.T) {
	b, err := os.ReadFile("testdata/seiseki.html")
	if err != nil {
		t.Fatal(err)
	}
	s := Sources()[0]
	list, _, err := ParseList(s, b, s.URL)
	if err != nil || len(list) != 1 {
		t.Fatalf("candidates=%v err=%v", list, err)
	}
	d, _ := os.ReadFile("testdata/seiseki-detail.html")
	e, err := ParseDetail(s, list[0], d, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if e.Title != "親子であそぼう" || len(e.Dates) != 2 || e.Dates[1].Start != "2026-11-08" {
		t.Fatalf("event=%+v", e)
	}
	if e.Venue != "河川敷" || e.MinAgeMonths == nil || *e.MinAgeMonths != 0 || e.Fee != "無料" {
		t.Fatalf("fields=%+v", e)
	}
	if e.DeadlineText != "2026年11月1日" || e.Dates[0].Start != "2026-11-07" {
		t.Fatal("deadline mixed with event date")
	}
}
func TestMissingStructureIsFailure(t *testing.T) {
	if _, _, err := ParseList(Sources()[0], []byte(`<html><h1>メンテナンス中</h1></html>`), Sources()[0].URL); err == nil {
		t.Fatal("structure change silently accepted")
	}
}
func TestDates(t *testing.T) {
	for _, tc := range []struct {
		in          string
		want        int
		first, last string
	}{
		{"2026年11月7日（土）・8日（日）", 2, "2026-11-07", "2026-11-08"},
		{"2026.11.07〜2026.11.09", 1, "2026-11-07", "2026-11-09"},
		{"2026年12月31日～2027年1月2日", 1, "2026-12-31", "2027-01-02"},
		{"11月7日", 0, "", ""}, {"2026年2月30日", 0, "", ""}, {"2026年11月123日", 0, "", ""},
	} {
		t.Run(tc.in, func(t *testing.T) {
			v := ParseDates(tc.in)
			if len(v) != tc.want {
				t.Fatalf("got %+v", v)
			}
			if len(v) > 0 {
				end := v[len(v)-1].End
				if end == "" {
					end = v[len(v)-1].Start
				}
				if v[0].Start != tc.first || end != tc.last {
					t.Fatalf("got %+v", v)
				}
			}
		})
	}
}
func TestMuseumDoesNotMixSubeventRestrictions(t *testing.T) {
	s := Sources()[1]
	e, err := ParseDetail(s, Candidate{Title: "模型博", URL: s.URL}, []byte(`<div id="content"><h1>模型博</h1><dl><dt>開催期間</dt><dd>2026年11月21日～2026年11月29日（休館日を除く）</dd></dl><h2>工作教室</h2><dl><dt>対象</dt><dd>小学生以上</dd><dt>会場</dt><dd>工作室</dd></dl></div>`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if e.AgeText != "" || e.Venue != "" {
		t.Fatalf("subevent restriction leaked: %+v", e)
	}
}

func TestDatesExcludeTimesAndHandleImplicitYearRollover(t *testing.T) {
	for _, tc := range []struct{ in, start, end string }{
		{"2026年11月5日 10～15時", "2026-11-05", ""},
		{"2026年11月5日 10:00～15:00", "2026-11-05", ""},
		{"2026年12月30日～1月3日", "2026-12-30", "2027-01-03"},
	} {
		got := ParseDates(tc.in)
		if len(got) != 1 || got[0].Start != tc.start || got[0].End != tc.end {
			t.Errorf("%s: %+v", tc.in, got)
		}
	}
}

func TestDetailDoesNotExtractColonFieldsFromSubevents(t *testing.T) {
	s := Sources()[1]
	e, err := ParseDetail(s, Candidate{Title: "展示", URL: s.URL}, []byte(`<div id="content"><h1>展示</h1><p>開催日：2026年11月5日</p><h2>工作企画</h2><p>日時：2026年12月5日</p><p>対象：小学生以上</p><p>会場：工作室</p><p>申込締切：2026年11月1日</p></div>`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if e.AgeText != "" || e.Venue != "" || e.DeadlineText != "" || e.DateText != "2026年11月5日" {
		t.Fatalf("subevent fields leaked: %+v", e)
	}
}

func TestAmbiguousAgeDoesNotBecomeNumericBounds(t *testing.T) {
	for _, age := range []string{"3歳以上（3歳未満も保護者同伴で参加可）", "未就学児不可", "小学生以上を推奨", "未就学児と小学生", "小学生以上"} {
		s := Sources()[1]
		e, err := ParseDetail(s, Candidate{Title: "体験", URL: s.URL}, []byte(`<div id="content"><dl><dt>開催日</dt><dd>2026年11月5日</dd><dt>対象</dt><dd>`+age+`</dd></dl></div>`), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if e.MinAgeMonths != nil || e.MaxAgeMonths != nil {
			t.Errorf("%s: bounds %+v", age, e)
		}
		if len(e.NeedsReview) == 0 {
			t.Errorf("%s: no review", age)
		}
	}
}

func TestExplicitlyEmptyList(t *testing.T) {
	s := Sources()[0]
	for _, html := range []string{`<div class="eventlist">該当するイベントはありません</div>`, `<div class="eventlist">現在開催予定のイベントはありません</div>`} {
		list, _, err := ParseList(s, []byte(html), s.URL)
		if err != nil || len(list) != 0 {
			t.Fatalf("list %v err %v", list, err)
		}
	}
	if _, _, err := ParseList(s, []byte(`<div class="eventlist"></div>`), s.URL); err == nil {
		t.Fatal("unconfirmed empty list accepted")
	}
}

func TestStaticSourceListsAndDetail(t *testing.T) {
	cases := []struct {
		id, html string
		count    int
	}{
		{"museum", `<div id="content"><a href="../../museum/event/123.html">親子体験</a><a href="../../museum/access/123.html">アクセス</a></div>`, 1},
		{"theater", `<a href="../theater/1000331/1003104/123.html"><img alt="親子コンサート"></a>`, 1},
		{"park", `<a class="news-list__item" href="/news/123/"><span class="news-list__date">2026年1月1日</span><span class="news-list__title">体験イベント</span></a>`, 1},
		{"machidukuri", `<article class="mc-event"><h3>はたらくくるま</h3><span class="date-wrapper">2026年11月8日</span><a href="/mc-events/123/">詳細</a></article>`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			var source Source
			for _, s := range Sources() {
				if s.ID == tc.id {
					source = s
				}
			}
			list, _, err := ParseList(source, []byte(tc.html), source.URL)
			if err != nil || len(list) != tc.count {
				t.Fatalf("%+v %v", list, err)
			}
			if tc.id == "park" && list[0].DateText != "" {
				t.Fatal("publication date treated as event date")
			}
		})
	}
}

func TestDatesDoNotIncludeClosureNotes(t *testing.T) {
	got := ParseDates("2026年11月21日～2026年11月29日 （注）休館日11月24日を除く")
	if len(got) != 1 || got[0].End != "2026-11-29" {
		t.Fatalf("closure parsed as event date: %+v", got)
	}
}

func TestTheaterPreservesAdmissionBanBeforeSubsections(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{`<p>料金：一般3000円。未就学児入場不可。</p>`, "未就学児入場不可"},
		{`<p>未就学児の入場不可。</p>`, "未就学児の入場不可"},
		{`<p>未就学児入場不可。</p><p>ただし保護者同伴で可。</p>`, "ただし"},
		{`<p>公演の案内です。</p><h2>関連企画</h2><p>未就学児入場不可。</p>`, ""},
	} {
		s := Sources()[3]
		e, err := ParseDetail(s, Candidate{Title: "公演", URL: s.URL}, []byte(`<div id="content"><h1>公演</h1>`+tc.body+`</div>`), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if tc.want == "" {
			if e.AgeText != "" {
				t.Fatalf("subsection leaked: %q", e.AgeText)
			}
		} else if !strings.Contains(e.AgeText, tc.want) {
			t.Fatalf("missing admission condition: %q", e.AgeText)
		}
	}
}

func TestMachidukuriUsesDisplayedCalendarYear(t *testing.T) {
	var s Source
	for _, v := range Sources() {
		if v.ID == "machidukuri" {
			s = v
		}
	}
	b := []byte(`<article class="mc-event"><h3>親子であそぼう</h3><p><span class="date-wrapper">2026年10月31日 – 2026年11月1日</span></p><div class="longdesc"><p>開催日時：10月31日(土)12:00～16:00、11月1日(日)10:40～17:00</p><p>会場：けやき並木通り</p></div></article>`)
	e, err := ParseDetail(s, Candidate{Title: "親子であそぼう", URL: s.URL}, b, time.Now())
	if err != nil || len(e.Dates) != 1 || e.Dates[0].Start != "2026-10-31" || e.Dates[0].End != "2026-11-01" {
		t.Fatalf("event=%+v err=%v", e, err)
	}
	if e.Venue != "けやき並木通り" || e.TimeText == "" {
		t.Fatalf("missing detail %+v", e)
	}
}

func TestContentAndGlobalAdmissionAreRetained(t *testing.T) {
	s := Source{ID: "theater", Name: "劇場", URL: "https://example.org/event"}
	b := []byte(`<div id="content"><h1>音楽会</h1><p>親子で聴く音楽</p><h2>プログラム</h2><p>曲目の紹介</p><ul class="info"><li>小学校入学前のお子様の入場はご遠慮ください。</li></ul></div>`)
	e, err := ParseDetail(s, Candidate{Title: "音楽会", URL: s.URL}, b, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.ContentText, "曲目の紹介") || !strings.Contains(e.ParticipationText, "小学校入学前") {
		t.Fatalf("evidence lost: %+v", e)
	}
}
