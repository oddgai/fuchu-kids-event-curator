// Package collect fetches and parses source-specific event pages.
package collect

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
)

type Source struct{ ID, Name, URL, DisabledReason string }
type Candidate struct{ Title, URL, DateText string }

func Sources() []Source {
	return []Source{
		{ID: "seiseki", Name: "聖蹟桜ヶ丘エリアマネジメント", URL: "https://seiseki.org/event/"},
		{ID: "museum", Name: "府中市郷土の森博物館", URL: "https://www.fuchu-cpf.or.jp/museum/event/index.html"},
		{ID: "park", Name: "府中市公園", URL: "https://fuchupark-area.com/news/"},
		{ID: "theater", Name: "府中の森芸術劇場", URL: "https://www.fuchu-cpf.or.jp/theater/"},
		{ID: "machidukuri", Name: "まちづくり府中", URL: "https://machidukuri-fuchu.jp/calendar/", DisabledReason: "利用規約のリンク事前承諾条件を確認するまで、自動出力は保留"},
		{ID: "kankou", Name: "府中観光協会", URL: "https://www.kankou-fuchu.com/?p=we-page-eventlist&eventlist=17414&mode=list&type=event&nav=none"},
		{ID: "lumiere", Name: "ルミエール府中", URL: "https://www.lumiere-fuchu.jp/event/"},
	}
}
func clean(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\u3000", " ", "\u00a0", " ").Replace(s)), " ")
}
func absolute(base, ref string) string {
	b, e := url.Parse(base)
	if e != nil {
		return ""
	}
	u, e := url.Parse(ref)
	if e != nil {
		return ""
	}
	u = b.ResolveReference(u)
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	if u.Host != b.Host {
		return ""
	}
	u.Fragment = ""
	return u.String()
}
func ParseList(s Source, b []byte, pageURL string) ([]Candidate, string, error) {
	d, e := goquery.NewDocumentFromReader(bytes.NewReader(b))
	if e != nil {
		return nil, "", e
	}
	var out []Candidate
	seen := map[string]bool{}
	add := func(title, ref, date string) {
		u := absolute(pageURL, ref)
		title = clean(title)
		if u == "" || title == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, Candidate{title, u, clean(date)})
	}
	switch s.ID {
	case "seiseki":
		d.Find(".eventCell .eventCellText > a").Each(func(_ int, a *goquery.Selection) {
			title := a.Find("p").Not(".times").First().Text()
			add(title, a.AttrOr("href", ""), a.Find(".times").Text())
		})
	case "museum":
		d.Find("#content a[href]").Each(func(_ int, a *goquery.Selection) {
			ref := a.AttrOr("href", "")
			u := absolute(pageURL, ref)
			if strings.Contains(u, "/museum/") && strings.HasSuffix(u, ".html") && !strings.HasSuffix(u, "index.html") && !strings.Contains(u, "/access/") {
				add(a.Text(), ref, "")
			}
		})
	case "theater":
		d.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
			ref := a.AttrOr("href", "")
			u := absolute(pageURL, ref)
			if strings.Contains(u, "/theater/1000331/") && strings.HasSuffix(u, ".html") && !strings.HasSuffix(u, "index.html") {
				title := a.Text()
				if clean(title) == "" {
					title = a.Find("img").AttrOr("alt", "")
				}
				add(title, ref, "")
			}
		})
	case "park":
		d.Find("a.news-list__item").Each(func(_ int, a *goquery.Selection) {
			title := a.Find(".news-list__title").Text()
			if strings.Contains(title, "中止") || strings.Contains(title, "イベント") || strings.Contains(title, "パーク") || strings.Contains(title, "体験") {
				add(title, a.AttrOr("href", ""), "")
			}
		})
	case "machidukuri":
		d.Find("article.mc-event").Each(func(_ int, a *goquery.Selection) {
			link := a.Find("a[href*='/mc-events/']").First()
			add(a.Find("h3").Text(), link.AttrOr("href", ""), a.Find(".date-wrapper").Text())
		})
	case "lumiere":
		d.Find("main a[href*='/event/']").Each(func(_ int, a *goquery.Selection) {
			u := absolute(pageURL, a.AttrOr("href", ""))
			if u != s.URL {
				add(a.Text(), u, "")
			}
		})
	case "kankou":
		return ParseKankouList(b, pageURL)
	}
	if len(out) == 0 {
		emptyText := clean(d.Find(".eventlist").Text())
		if d.Find(".eventlist").Length() > 0 && (strings.Contains(emptyText, "該当するイベントはありません") || strings.Contains(emptyText, "現在開催予定のイベントはありません")) {
			return []Candidate{}, "", nil
		}
		return nil, "", fmt.Errorf("イベント一覧の構造を確認できません（0件または構造変更、転送）")
	}
	next := ""
	d.Find("a.next.page-numbers, a.next, a.nextpostslink[rel='next'], .pagination a[rel='next']").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		next = absolute(pageURL, a.AttrOr("href", ""))
		return next == ""
	})
	return out, next, nil
}

// ParseDates only borrows year/month from earlier dates in the same field.
// It never uses publication dates or the current year to fill missing years.
func ParseDates(text string) []event.DateRange {
	for _, marker := range []string{"（注）", "(注)", "※"} {
		if i := strings.Index(text, marker); i >= 0 {
			text = text[:i]
		}
	}
	// Cut off a time expression before its range separator can become a date separator.
	if loc := regexp.MustCompile(`[0-9０-９]{1,2}(?:[:：][0-9０-９]{2}|(?:[〜～~–－-][0-9０-９]{1,2})?時)`).FindStringIndex(text); loc != nil {
		text = text[:loc[0]]
	}
	text = strings.NewReplacer("年", "-", "月", "-", "日", "", ".", "-", "/", "-", "（", "(", "）", ")").Replace(text)
	re := regexp.MustCompile(`(?:(\d{4})-)?(?:(\d{1,2})-)?(\d{1,2})(?:\([^)]*\))?`)
	idx := re.FindAllStringSubmatchIndex(text, -1)
	year, month := 0, 0
	var out []event.DateRange
	prevEnd := 0
	for _, m := range idx {
		// Do not accept a date parsed from only part of a longer number.
		if (m[0] > 0 && text[m[0]-1] >= '0' && text[m[0]-1] <= '9') || (m[1] < len(text) && text[m[1]] >= '0' && text[m[1]] <= '9') {
			continue
		}
		get := func(n int) int {
			if m[2*n] < 0 {
				return 0
			}
			v, _ := strconv.Atoi(text[m[2*n]:m[2*n+1]])
			return v
		}
		y, mo, day := get(1), get(2), get(3)
		// An explicitly descending month in a continuous range crosses New Year.
		if y == 0 && mo > 0 && month > 0 && mo < month && len(out) > 0 && strings.ContainsAny(text[prevEnd:m[0]], "〜～~–－-") {
			year++
		}
		if y > 0 {
			year = y
		}
		if mo > 0 {
			month = mo
		}
		// Stop at times or explanatory text rather than treating numbers as dates.
		if y == 0 && mo == 0 {
			if len(out) == 0 {
				continue
			}
			between := text[prevEnd:m[0]]
			if !strings.ContainsAny(between, "・、,〜～~–－-") {
				continue
			}
			if strings.ContainsAny(between, ":時") {
				continue
			}
		}
		if year == 0 || month == 0 || day == 0 {
			continue
		}
		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if t.Year() != year || int(t.Month()) != month || t.Day() != day {
			continue
		}
		iso := t.Format("2006-01-02")
		between := text[prevEnd:m[0]]
		isRange := len(out) > 0 && strings.ContainsAny(between, "〜～~–－-") && !strings.ContainsAny(between, "・、,")
		if isRange && iso >= out[len(out)-1].Start {
			out[len(out)-1].End = iso
		} else {
			out = append(out, event.DateRange{Start: iso})
		}
		prevEnd = m[1]
	}
	return out
}
func textLines(sel *goquery.Selection) []string {
	copy := sel.Clone()
	copy.Find("script,style,nav,footer,.update,.print,#sns").Remove()
	copy.Find("br").ReplaceWithHtml("\n")
	var lines []string
	copy.Find("p,dt,dd,li,h2,h3").Each(func(_ int, s *goquery.Selection) {
		if s.Find("p,dt,dd,li").Length() > 0 {
			return
		}
		if v := clean(s.Text()); v != "" {
			lines = append(lines, v)
		}
	})
	return lines
}
func ParseDetail(s Source, c Candidate, b []byte, now time.Time) (event.Event, error) {
	if s.ID == "kankou" {
		return ParseKankouDetail(s, c, b, now)
	}
	d, err := goquery.NewDocumentFromReader(bytes.NewReader(b))
	if err != nil {
		return event.Event{}, err
	}
	e := event.Event{Title: c.Title, URL: c.URL, DateText: c.DateText, FetchedAt: now, Sources: []event.SourceRef{{Name: s.Name, URL: c.URL}}, AgeStatus: "unknown"}
	var body *goquery.Selection
	switch s.ID {
	case "seiseki":
		body = d.Find(".newsSingleBlock.bottomSpace60")
		if t := clean(d.Find(".newsSingleBlockTitle h3").First().Text()); t != "" {
			e.Title = t
		}
		if dt := clean(d.Find(".newsSingleBlockTitle > p.times").Text()); dt != "" {
			e.DateText = dt
		}
	case "museum", "theater":
		body = d.Find("#content")
		if t := clean(body.Find("h1").First().Text()); t != "" {
			e.Title = t
		}
	case "park":
		body = d.Find(".news-detail__article")
		if body.Length() == 0 {
			body = d.Find("main.news-detail")
		}
		if t := clean(d.Find(".news-detail__title").First().Text()); t != "" {
			e.Title = t
		}
	case "machidukuri":
		body = d.Find(".mc-event,article").First()
	default:
		body = d.Find("main,article").First()
	}
	if body.Length() == 0 || len(clean(body.Text())) < 5 {
		return e, fmt.Errorf("詳細本文の構造を確認できません")
	}
	// Only the first metadata block before any subsection describes the whole event.
	fields := map[string]string{}
	metadata := body.Clone()
	// Remove the first subsection and everything after it, including nested subsections.
	var trim func(*goquery.Selection) bool
	trim = func(parent *goquery.Selection) bool {
		stopped := false
		parent.Children().Each(func(_ int, child *goquery.Selection) {
			if stopped {
				child.Remove()
				return
			}
			if goquery.NodeName(child) == "h2" {
				child.Remove()
				stopped = true
				return
			}
			if trim(child) {
				stopped = true
			}
		})
		return stopped
	}
	trim(metadata)
	metadata.Find("dt").Each(func(_ int, dt *goquery.Selection) {
		if s.ID == "museum" || s.ID == "theater" {
			if dt.ParentsFiltered("dl").First().PrevAllFiltered("h2,h3").Length() > 0 {
				return
			}
		}
		k := clean(dt.Text())
		v := clean(dt.NextFiltered("dd").Text())
		if fields[k] == "" {
			fields[k] = v
		}
	})
	lines := textLines(body)
	for _, line := range textLines(metadata) {
		parts := strings.SplitN(strings.ReplaceAll(line, "：", ":"), ":", 2)
		if len(parts) == 2 && len([]rune(parts[0])) <= 8 {
			k := clean(parts[0])
			if fields[k] == "" {
				fields[k] = clean(parts[1])
			}
		}
	}
	pick := func(keys ...string) string {
		for _, k := range keys {
			if fields[k] != "" {
				return fields[k]
			}
		}
		return ""
	}
	// Explicit day lists in the body are more precise than a broad listing range.
	if v := pick("開催期間", "開催日", "日時", "開催日時", "日程"); v != "" {
		e.DateText = v
	}
	e.TimeText = pick("開催時間", "時間")
	if e.TimeText == "" {
		if v := pick("日時", "開催日時"); v != "" && strings.ContainsAny(v, ":時") {
			e.TimeText = v
		}
	}

	if s.ID == "machidukuri" {
		if displayed := clean(body.Find(".date-wrapper").First().Text()); displayed != "" {
			e.DateText = displayed
		}
		if e.TimeText == "" {
			e.TimeText = clean(body.Find(".time-wrapper").First().Text())
		}
	}
	e.Venue = pick("会場", "場所", "開催場所")
	e.AgeText = pick("対象", "対象年齢", "参加対象")
	// Admission restrictions often appear in the fee or an ordinary paragraph.
	// Preserve their context, including exceptions, without assigning a numeric age.
	metadataText := clean(metadata.Text())
	if regexp.MustCompile(`未就学児(?:の)?入場不可`).MatchString(metadataText) {
		e.AgeText = metadataText
	} else if e.AgeText != "" {
		for _, word := range []string{"ただし", "未満も", "同伴で可", "例外"} {
			if strings.Contains(metadataText, word) && !strings.Contains(e.AgeText, word) {
				e.AgeText = metadataText
				break
			}
		}
	}
	e.Fee = pick("料金", "参加費", "費用")
	e.Reservation = pick("予約", "申込", "申し込み", "申込み", "申込方法")
	e.DeadlineText = pick("申込締切", "申込期限", "締切", "申込期間")
	if e.DateText == "" {
		e.DateText = e.Title
	}
	e.Dates = ParseDates(e.DateText)
	if strings.Contains(e.DateText, "除く") || strings.Contains(e.DateText, "休館") {
		e.NeedsReview = append(e.NeedsReview, "開催期間内の休館・除外日は出典の注記を確認してください")
	}
	if len(e.Dates) == 0 {
		e.NeedsReview = append(e.NeedsReview, "開催年・日程を原文で確認してください（掲載日から補完していません）")
	}
	if e.AgeText != "" {
		// Only a complete, affirmative numeric age condition establishes a bound.
		// School stages and exception-bearing prose require checking the original.
		age := strings.TrimSpace(e.AgeText)
		age = strings.Map(func(r rune) rune {
			if r >= '０' && r <= '９' {
				return r - '０' + '0'
			}
			return r
		}, age)
		if m := regexp.MustCompile(`^([0-9]+)歳(?:以上|から)(?:参加可(?:能)?)?$`).FindStringSubmatch(age); len(m) > 0 {
			n, err := strconv.Atoi(m[1])
			if err == nil {
				v := n * 12
				e.MinAgeMonths = &v
			}
		} else if m := regexp.MustCompile(`^([0-9]+)歳以下$`).FindStringSubmatch(age); len(m) > 0 {
			n, err := strconv.Atoi(m[1])
			if err == nil {
				v := (n+1)*12 - 1
				e.MaxAgeMonths = &v
			}
		} else {
			e.NeedsReview = append(e.NeedsReview, "対象年齢の条件・例外を原文で確認してください")
		}
	}
	// Keep a short introduction; do not copy entire source pages into reports.
	for _, line := range lines {
		if len([]rune(line)) >= 12 && !strings.Contains(line, "：") && !strings.Contains(line, ":") {
			e.Summary = line
			break
		}
	}
	runes := []rune(e.Summary)
	if len(runes) > 140 {
		e.Summary = string(runes[:140]) + "…"
	}
	if body.Find("h2").Length() > 1 {
		e.NeedsReview = append(e.NeedsReview, "複数企画を含むページです。企画ごとの対象・料金・開催日を出典で確認してください")
	}
	if body.Find("img").Length() > 0 {
		e.NeedsReview = append(e.NeedsReview, "チラシ画像等の文字は抽出していません。最新条件は出典で確認してください")
	}
	e.ID = fmt.Sprintf("%x", sha256.Sum256([]byte(e.URL)))[:16]
	return e, nil
}
