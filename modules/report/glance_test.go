package report

import (
	"os"
	"strings"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
)

func glanceAnalysis(words int, coverage float64, pos, neg int) *Analysis {
	return &Analysis{
		WordCount:          words,
		DictionaryCoverage: coverage,
		CalculationDetails: &analyze.CalculationDetails{
			CategoryCounts: map[analyze.Category]int{"positive_emotion": pos, "negative_emotion": neg},
		},
	}
}

func TestBuildGlanceExplainsQualityAndEmotion(t *testing.T) {
	g := BuildGlance(glanceAnalysis(527, 0.73, 4, 11))
	if !strings.Contains(g.Read, "527 words") || !strings.Contains(g.Read, "73%") {
		t.Fatalf("Read = %q", g.Read)
	}
	if len(g.QualityReasons) == 0 {
		t.Fatal("527 words is medium quality and must carry a reason")
	}
	if g.Emotion != "Emotion words found: 11 negative-feeling, 4 positive-feeling." {
		t.Fatalf("Emotion = %q", g.Emotion)
	}
	if g.SupportNote == "" {
		t.Fatal("2.1% negative words with 2.75x more negative than positive should add the support note")
	}
	if g.Caveat != analyze.ReadingCaveat {
		t.Fatalf("Caveat = %q", g.Caveat)
	}
}

func TestBuildGlanceSupportNoteNeedsBothConditions(t *testing.T) {
	if BuildGlance(glanceAnalysis(1000, 0.8, 5, 12)).SupportNote != "" {
		t.Fatal("1.2% negative words is below the share threshold")
	}
	if BuildGlance(glanceAnalysis(500, 0.8, 8, 12)).SupportNote != "" {
		t.Fatal("negatives under twice the positives must not trigger the note")
	}
}

func TestBuildGlanceLegacyAnalysisHasNoEmotionLine(t *testing.T) {
	g := BuildGlance(&Analysis{WordCount: 1200, DictionaryCoverage: 0.8})
	if g.Emotion != "" || g.SupportNote != "" {
		t.Fatalf("legacy analysis without calculation details must not invent counts: %+v", g)
	}
	if g.Read == "" || g.Caveat == "" {
		t.Fatalf("size and caveat must still render: %+v", g)
	}
}

func TestGlanceCopyHasNoEmDash(t *testing.T) {
	g := BuildGlance(glanceAnalysis(300, 0.4, 1, 9))
	all := strings.Join(append([]string{g.Read, g.Emotion, g.Caveat, g.SupportNote}, g.QualityReasons...), " ")
	if strings.ContainsRune(all, '—') {
		t.Fatalf("user-facing copy must not contain em-dashes: %q", all)
	}
}

func TestSummaryCardsCarryMeaningAndEmotionDetail(t *testing.T) {
	v := BuildReport(glanceAnalysis(527, 0.73, 4, 11))
	var tone SummaryCard
	for _, c := range v.Summary {
		if c.Meaning == "" {
			t.Errorf("%s card has no meaning line", c.Key)
		}
		if c.Key == "emotional_tone" {
			tone = c
		}
	}
	if tone.Detail != "Based on 11 negative and 4 positive emotion words." {
		t.Fatalf("emotional tone detail = %q", tone.Detail)
	}
	for _, c := range v.Summary {
		if c.Key != "emotional_tone" && c.Detail != "" {
			t.Errorf("%s card must not carry emotion counts", c.Key)
		}
	}
}

func TestTraitCardsShowMeaningAndRankBesideScore(t *testing.T) {
	a := testAnalysis()
	a.PercentileReference = &ingest.PercentileReference{Method: ingest.PercentileMethodEmpirical, SampleSize: 3992}
	a.Traits["conscientiousness"] = Trait{Score: 0.44, Percentile: 1, ConfidenceInterval: []float64{.2, .66}}
	v := BuildReport(a)
	var found bool
	for _, tv := range v.Traits {
		if tv.Meaning == "" {
			t.Errorf("%s card has no meaning line", tv.Key)
		}
		if tv.Key == "conscientiousness" {
			found = true
		}
	}
	if !found {
		t.Fatal("conscientiousness card missing")
	}

	var out strings.Builder
	if err := RenderAnalysis("../../templates", a, &out, false); err != nil {
		t.Fatal(err)
	}
	main, _, _ := strings.Cut(out.String(), "Calculation details and limitations")
	if strings.Contains(main, "Higher than about") {
		t.Error("the rank sentence was removed from the cards")
	}
	if strings.Contains(strings.ToLower(main), "percentile") {
		t.Error("main reading must not use the word percentile")
	}
}

func TestLegacyPayloadStillRendersWithoutRank(t *testing.T) {
	a := testAnalysis()
	a.PercentileReference = nil
	a.CalculationDetails = nil
	var out strings.Builder
	if err := RenderAnalysis("../../templates", a, &out, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "reference texts.") && strings.Contains(out.String(), "Higher than about") {
		t.Fatal("no recorded reference means no rank line")
	}
	if strings.Contains(out.String(), "At a glance") {
		t.Fatal("the At a glance section was removed from the page")
	}
}

func TestWordCountFormattedTheSameEverywhere(t *testing.T) {
	a := testAnalysis()
	a.WordCount = 1133
	var out strings.Builder
	if err := RenderAnalysis("../../templates", a, &out, false); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if strings.Contains(html, "1133 words") {
		t.Error("word counts must use thousands separators everywhere")
	}
	if got := strings.Count(html, "1,133 words"); got < 2 {
		t.Errorf("header and details should both say 1,133 words, found %d", got)
	}
}

func TestQualityReasonDoesNotMisstateCoverage(t *testing.T) {
	// 58% coverage is most of the text, so the reason must not say "few".
	for _, r := range analyze.QualityReasonsWithNoise(10785, 0.58, 0) {
		if strings.Contains(strings.ToLower(r), "few of") {
			t.Errorf("reason misstates 58%% coverage: %q", r)
		}
	}
}

func TestReportShowsMeasureTooltipAndHeading(t *testing.T) {
	a := testAnalysis()
	a.Traits["conscientiousness"] = Trait{Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{.47, .53}}
	var out strings.Builder
	if err := RenderAnalysis("../../templates", a, &out, false); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if strings.Contains(html, "would likely score") {
		t.Error("per-card range sentence should be gone")
	}
	if !strings.Contains(html, "ⓘ") || !strings.Contains(html, "How curious and open to new ideas") {
		t.Error("measure names need a visible tap cue and a plain-language tooltip")
	}
	if strings.Contains(html, "band explanation") || strings.Contains(html, "Big Five bands") {
		t.Error("moderate/high/low badges and the band legend should be gone")
	}
	if !strings.Contains(html, "<h2 class=\"text-sm font-medium text-stone-700\">Evidence by measure</h2>") {
		t.Error("evidence blocks need a parent heading")
	}
}

func TestReportOffersBrowserActionsAndNoServerLinks(t *testing.T) {
	a := testAnalysis()
	var frag strings.Builder
	if err := RenderAnalysisFS(os.DirFS("../../templates"), a, &frag, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag.String(), `data-psycho-action="print"`) || strings.Contains(frag.String(), "/pdf") || strings.Contains(frag.String(), "/analysis/") {
		t.Error("report must offer print and no server links")
	}
	var page strings.Builder
	if err := RenderStandaloneAnalysisFS(os.DirFS("../../templates"), []byte("body{}"), a, &page); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.String(), "<style>body{}</style>") || strings.Contains(page.String(), "data-psycho-action") {
		t.Error("saved report embeds CSS and has no dead buttons")
	}
}
