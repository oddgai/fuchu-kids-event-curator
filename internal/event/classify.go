package event

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Assessment describes content suitability, independently of the publication period.
type Assessment struct {
	Category string   `json:"category"`
	Reasons  []string `json:"reasons"`
	Evidence []string `json:"evidence,omitempty"`
}

type Curated struct{ Events, Information, Review, Excluded []Event }

func normalized(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		switch r {
		case '：':
			return ':'
		case '　', ' ', '\t':
			return -1
		}
		return r
	}, s)
}
func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
func assessment(kind, reason, evidence string) Assessment {
	a := Assessment{Category: kind, Reasons: []string{reason}}
	if evidence != "" {
		a.Evidence = []string{evidence}
	}
	return a
}
func positiveChild(s string) bool {
	return containsAny(s, "ファミリーコンサート", "小さなお子様から", "小さな子どもから", "お子様移動遊具", "子ども向け遊具", "子供向け遊具", "幼児向け", "落ち葉プール", "でんしゃであそぼう", "親子であそぼう", "絵本", "読み聞かせ", "手あそび", "リトミック", "乗車体験", "未就学児入場可", "未就学児も入場可", "小学校入学前のお子様の入場も可能")
}

// Classify uses general expressions and source preferences, never a URL verdict list.
func Classify(e Event, ageMonths int) Assessment { return classify(e, ageMonths, true) }

func classify(e Event, ageMonths int, availability bool) Assessment {
	title := normalized(e.Title)
	content := normalized(e.Title + "\n" + e.Summary + "\n" + e.ContentText)
	conditions := normalized(e.AgeText + "\n" + e.ParticipationText + "\n" + e.VerifiedText)
	evidence := strings.TrimSpace(e.AgeText + " " + e.VerifiedText)
	if availability && (title == "雨天中止" || strings.HasPrefix(title, "【中止】") || strings.Contains(title, "開催中止のお知らせ")) {
		return assessment("exclude", "開催中止と明記されています", e.Title)
	}
	if availability && (containsAny(title, "受付終了", "定員到達", "満員", "定員に達しました") || containsAny(normalized(e.Reservation), "受付終了", "定員に達しました")) {
		return assessment("exclude", "申込受付が終了または定員に達しています", e.Title+" "+e.Reservation)
	}

	for _, sentence := range strings.FieldsFunc(conditions, func(r rune) bool { return r == '。' || r == '\n' }) {
		if !availability {
			break
		}
		if containsAny(sentence, "雨天", "荒天", "場合", "可能性", "ではありません", "ではない") {
			continue
		}
		if containsAny(sentence, "開催中止です", "開催中止となりました", "開催中止いたします") || sentence == "開催中止" {
			return assessment("exclude", "開催中止と明記されています", sentence)
		}
		if containsAny(sentence, "申込受付終了", "受付は終了", "受付を終了", "受付終了しました", "受付終了です", "満員御礼", "定員に達しました") {
			return assessment("exclude", "申込受付が終了または定員に達しています", sentence)
		}
	}
	// Do not convert an exception-bearing condition into an unconditional restriction.
	exception := containsAny(conditions, "ただし", "例外", "同伴で参加可", "同伴なら", "保護者同伴で可", "未満も")
	forbidden := regexp.MustCompile(`(?:未就学児|小学校入学前|小学生未満)[^。\n]{0,35}(?:入場|同伴)[^。\n]{0,15}(?:不可|ご遠慮|できません|お断り)`)
	allowed := containsAny(conditions, "小学校入学前のお子様の入場も可能", "未就学児も入場可能", "未就学児入場可", "0歳から入場可能")
	for _, sentence := range strings.FieldsFunc(conditions, func(r rune) bool { return r == '。' || r == '\n' }) {
		if containsAny(sentence, "無印の公演", "他の公演", "別の公演") {
			continue
		}
		if forbidden.MatchString(sentence) && ageMonths < 60 {
			if exception {
				return assessment("review", "入場制限に例外があります。条件の確認が必要です", sentence)
			}
			return assessment("exclude", "未就学児の入場・同伴が制限されています", sentence)
		}
	}
	if e.MinAgeMonths != nil && ageMonths < *e.MinAgeMonths || e.MaxAgeMonths != nil && ageMonths > *e.MaxAgeMonths {
		return assessment("exclude", "明示された対象年齢の範囲外です", evidence)
	}
	// Ranges are checked before standalone minimums, so 5-12歳 is not read as 12歳以上.
	ageRange := regexp.MustCompile(`([0-9]{1,2})(?:歳)?[〜～~－-]([0-9]{1,2})歳`)
	if m := ageRange.FindStringSubmatch(conditions); len(m) > 0 {
		lo, _ := strconv.Atoi(m[1])
		hi, _ := strconv.Atoi(m[2])
		if lo > hi {
			return assessment("review", "対象年齢の範囲が不整合です", m[0])
		}
		if ageMonths < lo*12 || ageMonths >= (hi+1)*12 {
			if exception {
				return assessment("review", "対象年齢の例外条件を確認してください", m[0])
			}
			return assessment("exclude", "明示された対象年齢の範囲外です", m[0])
		}
	}
	minAge := regexp.MustCompile(`([0-9]{1,2})歳(?:以上|から|くらい[〜～~－-])`)
	for _, m := range minAge.FindAllStringSubmatch(conditions, -1) {
		n, _ := strconv.Atoi(m[1])
		if ageMonths < n*12 {
			if exception {
				return assessment("review", "対象年齢の例外条件を確認してください", m[0])
			}
			return assessment("exclude", "対象年齢または対象の目安が現在の月齢より上です", m[0])
		}
	}
	maxAge := regexp.MustCompile(`([0-9]{1,2})歳(以下|未満)`)
	if m := maxAge.FindStringSubmatch(conditions); len(m) > 0 {
		n, _ := strconv.Atoi(m[1])
		max := (n + 1) * 12
		if m[2] == "未満" {
			max = n * 12
		}
		if ageMonths >= max && !exception {
			return assessment("exclude", "明示された対象年齢の範囲外です", m[0])
		}
	}
	if ageMonths < 60 && !exception && containsAny(conditions, "小学生以上", "中学生以上", "高校生以上", "成人対象") {
		return assessment("exclude", "対象が就学後または大人の催しです", evidence)
	}

	// A seasonal attraction remains useful period information even when lit at night.
	if containsAny(title, "イルミ", "ライトアップ") && !positiveChild(content) {
		return assessment("information", "期間限定のお出かけ情報として掲載します", e.Title)
	}
	playpark := strings.Contains(title, "プレーパーク")
	if m := regexp.MustCompile(`([0-9]{1,2})(?::[0-9]{2}|時)`).FindStringSubmatch(normalized(e.TimeText + "\n" + e.VerifiedText)); len(m) > 0 {
		hour, _ := strconv.Atoi(m[1])
		if hour >= 18 && !playpark {
			return assessment("exclude", "夜間開始のため、昼間を優先する希望に合いません", strings.TrimSpace(e.TimeText+" "+e.VerifiedText))
		}
	}
	if containsAny(title, "イベント一覧", "催し一覧", "芸術文化祭") {
		return assessment("review", "複数イベントの集合案内です。個別企画の確認が必要です", e.Title)
	}
	if playpark {
		a := assessment("recommend", "プレーパークは好みに合う遊びの催しです", e.Title)
		if containsAny(title, "ナイト") || regexp.MustCompile(`(?:18|19|20)[:時]`).MatchString(normalized(e.TimeText)) {
			a.Reasons = append(a.Reasons, "夜のプレーパークもOKという評価を反映。時間はご確認ください")
		}
		return a
	}
	if allowed || positiveChild(content+conditions) {
		why := "幼児歓迎や具体的な遊びの記載があります"
		return assessment("recommend", why, firstChildEvidence(e))
	}
	if containsAny(title, "グルメ", "キッチンカーカレンダー", "落語", "酉の市", "クリーンアクション", "ゴミ拾い", "バックヤードツアー", "大人の") || strings.Contains(title, "物産館") && containsAny(content, "販売", "野菜", "食べ物", "試食") {
		return assessment("exclude", "飲食中心・大人向け・現在の年齢には早い内容として優先度を下げています", e.Title)
	}
	for _, source := range append(append([]SourceRef(nil), e.Sources...), SourceRef{URL: e.URL}) {
		u, err := url.Parse(source.URL)
		if err == nil && strings.EqualFold(u.Hostname(), "fuchupark-area.com") {
			return assessment("recommend", "公園サイトのイベントを優先する希望に合います", e.Title)
		}
	}
	return assessment("review", "幼児が楽しめる具体的な根拠を確認できません", "")
}

func firstChildEvidence(e Event) string {
	for _, s := range []string{e.VerifiedText, e.AgeText, e.ParticipationText, e.ContentText, e.Summary, e.Title} {
		for _, line := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '。' }) {
			if positiveChild(normalized(line)) {
				r := []rune(line)
				if len(r) > 160 {
					line = string(r[:160]) + "…"
				}
				return line
			}
		}
	}
	return e.Title
}

func Curate(events []Event, from, until string, ageMonths int) Curated {
	out := Curated{}
	for _, original := range events {
		e := cloneEvent(original)
		a := Classify(e, ageMonths)
		e.Assessment = &a
		overlap, unknown := dateMatch(e.Dates, from, until)
		if !overlap && !unknown {
			e.NeedsReview = appendUnique(e.NeedsReview, "掲載対象期間外です")
			out.Excluded = append(out.Excluded, e)
			continue
		}
		if a.Category == "exclude" {
			out.Excluded = append(out.Excluded, e)
			continue
		}
		if unknown {
			e.NeedsReview = appendUnique(e.NeedsReview, "開催年・開催日が未確定です。おすすめ一覧から分けています")
			out.Review = append(out.Review, e)
			continue
		}
		switch a.Category {
		case "recommend":
			e.Recommendation = ""
			out.Events = append(out.Events, e)
		case "information":
			out.Information = append(out.Information, e)
		default:
			out.Review = append(out.Review, e)
		}
	}
	out.Events = mergeCuratedDuplicates(out.Events)
	out.Information = mergeCuratedDuplicates(out.Information)
	out.Review = mergeCuratedDuplicates(out.Review)
	return out
}
func mergeCuratedDuplicates(events []Event) []Event {
	out := make([]Event, 0, len(events))
	positions := map[string]int{}
	for _, e := range events {
		key := eventKey(e)
		if i, ok := positions[key]; ok && key != "" {
			for _, s := range e.Sources {
				found := false
				for _, old := range out[i].Sources {
					if old == s {
						found = true
					}
				}
				if !found {
					out[i].Sources = append(out[i].Sources, s)
				}
			}
			continue
		}
		if key != "" {
			positions[key] = len(out)
		}
		out = append(out, e)
	}
	return out
}

// stripAvailability is used only by the content-feedback evaluation, not publication.
func stripAvailability(title string) string {
	return regexp.MustCompile(`(?:[（(]受付終了[）)]|[（(]満員[）)]|※ご好評につき定員に達しました)`).ReplaceAllString(title, "")
}
