package event

import "testing"

func TestContentAndConstraints(t *testing.T) {
	for _, tc := range []struct{ title, text, age, want string }{
		{"別のファミリーコンサート", "", "", "recommend"},
		{"親子コンサート", "", "対象：5-12歳", "exclude"},
		{"新しい公演", "", "小学校入学前のお子様の入場はご遠慮ください。", "exclude"},
		{"子育て講座", "", "高校生以上。未就学児 同伴不可", "exclude"},
		{"公開ミニライブ", "", "この公演は、小学校入学前のお子様の入場も可能な公演です。無印の公演は、未就学児の入場はご遠慮ください。", "recommend"},
		{"秋のマルシェ", "飲食出店とお子様移動遊具があります", "", "recommend"},
		{"グルメフェア", "食べ物の販売が中心です", "", "exclude"},
		{"キッチンカーカレンダー", "", "", "exclude"},
		{"落語会", "", "", "exclude"},
		{"親子の落語あそび", "2歳から楽しめる幼児向けの人形遊び", "対象：2歳以上", "recommend"},
		{"クリーンアクション", "ゴミ拾い", "", "exclude"},
		{"バックヤードツアー", "", "", "exclude"},
		{"冬のイルミネーション", "", "", "information"},
		{"プレーパーク", "", "対象：4歳くらい〜", "exclude"},
		{"プレーパーク（満員）", "", "", "exclude"},
		{"展示会", "", "", "review"},
	} {
		t.Run(tc.title+tc.age, func(t *testing.T) {
			a := Classify(Event{Title: tc.title, ContentText: tc.text, AgeText: tc.age}, 32)
			if a.Category != tc.want {
				t.Fatalf("want %s got %+v", tc.want, a)
			}
			if len(a.Reasons) == 0 {
				t.Fatal("reason missing")
			}
		})
	}
}
func TestContentPreferenceSeparateFromPublication(t *testing.T) {
	e := Event{Title: "プレーパーク", TimeText: "18:30〜20:00"}
	if a := Classify(e, 32); a.Category != "recommend" {
		t.Fatalf("night playpark exception: %+v", a)
	}
	e.Dates = []DateRange{{Start: "2026-07-31"}}
	if got := Curate([]Event{e}, "2026-11-01", "2026-11-30", 32); len(got.Events) != 0 || len(got.Excluded) != 1 {
		t.Fatalf("past event published: %+v", got)
	}
	e.Dates = nil
	if got := Curate([]Event{e}, "2026-11-01", "2026-11-30", 32); len(got.Events) != 0 || len(got.Review) != 1 {
		t.Fatalf("unknown date published: %+v", got)
	}
}
func TestGeneralRulesDoNotRequireKnownURL(t *testing.T) {
	for _, u := range []string{"", "https://new.example.com/x"} {
		a := Classify(Event{Title: "こどもの休日", URL: u, ContentText: "おもちゃを持って入らないでください。落ち葉プールで遊べます"}, 32)
		if a.Category != "recommend" {
			t.Fatal(a)
		}
	}
}
func TestNightEventWithoutPlaypark(t *testing.T) {
	a := Classify(Event{Title: "川辺テラス", TimeText: "18時～20時"}, 32)
	if a.Category != "exclude" {
		t.Fatal(a)
	}
}

func TestSeasonalInformationIndependentOfNightPreference(t *testing.T) {
	a := Classify(Event{Title: "冬のイルミネーション", TimeText: "18:00〜22:00"}, 32)
	if a.Category != "information" {
		t.Fatal(a)
	}
}
func TestOverallAvailabilityOverridesPreferredActivity(t *testing.T) {
	for _, text := range []string{"この催しは開催中止です。", "申込受付終了。定員に達しました"} {
		a := Classify(Event{Title: "プレーパーク", ParticipationText: text}, 32)
		if a.Category != "exclude" {
			t.Fatalf("%s %+v", text, a)
		}
		a = Classify(Event{Title: "プレーパーク", VerifiedText: text}, 32)
		if a.Category != "exclude" {
			t.Fatalf("verified %s %+v", text, a)
		}
	}
	for _, text := range []string{"雨天の場合は開催中止です", "受付終了日時：11月10日", "開催中止ではありません"} {
		if a := Classify(Event{Title: "プレーパーク", ParticipationText: text}, 32); a.Category != "recommend" {
			t.Fatalf("conditional %s %+v", text, a)
		}
	}
}
