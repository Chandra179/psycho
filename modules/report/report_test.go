package report

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/zlogger"
)

func TestRecordedCalculationsRenderedWithoutRecalculation(t *testing.T) {
	a := testAnalysis()
	a.CalculationDetails = &analyze.CalculationDetails{ModelFingerprint: "historical-model", Traits: map[string]*analyze.ScoreCalculation{
		"openness": {Baseline: .42, ContributionTotal: .123456789, ModelScore: .54, CalibrationApplied: true, CalibrationOffset: .16, FinalScore: .7, Terms: []analyze.ScoreTerm{{Category: "recorded_category", MatchedCount: 7, TotalWords: 570, WordPercent: 1.2280701754385965, Weight: .123, Contribution: .15105263157894738}}},
	}}
	var out strings.Builder
	if err := RenderAnalysis("../../templates", a, &out, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Baseline 0.4200", "calibration &#43;0.1600", "recorded_category", "7 / 570", `aria-valuenow="70"`, "Download calculation details"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing recorded value %q", want)
		}
	}
	if strings.Contains(out.String(), ">article</td>") {
		t.Fatal("used rounded legacy evidence instead of recorded operands")
	}
	if strings.Contains(out.String(), "<pre") {
		t.Fatal("the raw JSON must be a download, not an inline block")
	}
	_, link, ok := strings.Cut(out.String(), "data:application/json;charset=utf-8;base64,")
	if !ok {
		t.Fatal("missing calculation download link")
	}
	link, _, _ = strings.Cut(link, `"`)
	raw, err := base64.StdEncoding.DecodeString(link)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"0.15105263157894738", "historical-model"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("downloaded JSON is missing recorded value %q", want)
		}
	}
}

func TestReportEscapesUntrustedFieldsAndStandaloneCSS(t *testing.T) {
	a := testAnalysis()
	attack := `<script>window.injected=1</script><img src=x onerror=alert(1)>`
	a.AnalysisID = attack
	a.PercentileReference.Corpus = attack
	trait := a.Traits["openness"]
	trait.Evidence[0].Category = attack
	trait.Evidence[0].MatchedWords = []string{attack}
	a.Traits["openness"] = trait
	a.CalculationDetails = &analyze.CalculationDetails{ModelFingerprint: attack}
	for _, standalone := range []bool{false, true} {
		var out strings.Builder
		var err error
		if standalone {
			err = RenderStandaloneAnalysis("../../templates", a, &out)
		} else {
			err = RenderAnalysis("../../templates", a, &out, true)
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "<script>") || strings.Contains(out.String(), "<img src=x") {
			t.Fatal("unescaped user HTML")
		}
		if standalone {
			if !strings.Contains(out.String(), "<style>") || strings.Contains(out.String(), `href="/assets/app.css"`) {
				t.Fatal("offline report requires server styles")
			}
		} else if !strings.Contains(out.String(), `href="/assets/app.css"`) {
			t.Fatal("missing local stylesheet")
		}
	}
}

func testAnalysis() *Analysis {
	return &Analysis{
		AnalysisID:         "test-id",
		WordCount:          570,
		DictionaryCoverage: 0.667,
		ConfidenceFlag:     "medium",
		PercentileReference: &ingest.PercentileReference{
			Method: ingest.PercentileMethodEmpirical, Corpus: "Reference essay sample", SampleSize: 2400,
		},
		Traits: map[string]Trait{
			"openness": {Score: 0.7, Percentile: 91, ConfidenceInterval: []float64{0.45, 0.9},
				Evidence: []ContributionJSON{
					{Category: "article", WordPercent: 9.3, Weight: -0.01, Contribution: -0.07, MatchedWords: []string{"the", "a"}},
					{Category: "positive_emotion", WordPercent: 1.2, Weight: 0.02, Contribution: 0.024, MatchedWords: []string{"happy"}},
					{Category: "inclusive", WordPercent: 0.9, Weight: 0.01, Contribution: 0.009, MatchedWords: []string{"with"}},
					{Category: "cause", WordPercent: 0.4, Weight: 0.03, Contribution: 0.012, MatchedWords: []string{"because"}},
				}},
			"neuroticism": {Score: 0.3, Percentile: 8},
			"cognitive_style": {Score: 0.5, Percentile: 50, Evidence: []ContributionJSON{
				{Category: "long_word_ratio", WordPercent: 12.4, Weight: 0.02, Contribution: 0.248},
			}},
			// agreeableness deliberately absent: must be skipped, not zero-filled.
		},
		Values:        map[string]float64{"value_universalism": 0.88, "value_achievement": 0.18},
		ValueEvidence: map[string][]string{"value_universalism": {"just", "world"}},
		Summary:       SummaryVariables{AnalyticalThinking: 0.16, Clout: 0.06, Authenticity: 0.71, EmotionalTone: 0.48},
	}
}

func TestBuildReportSuperset(t *testing.T) {
	v := BuildReport(testAnalysis())

	if got := len(v.Traits); got != 3 {
		t.Fatalf("expected 3 traits (missing key skipped), got %d", got)
	}
	if v.Traits[0].Name != "Openness" || v.Traits[0].Label != "high" {
		t.Fatalf("unexpected first trait: %+v", v.Traits[0])
	}
	if !v.Traits[0].HasScoreRange || v.Traits[0].ScoreRangeLow != 45 || v.Traits[0].ScoreRangeHigh != 90 {
		t.Fatalf("rough score range not scaled to 0-100: %+v", v.Traits[0])
	}
	if v.Traits[0].Score100 != 70 || v.Traits[0].SignalDescription != analyze.DimensionBandDescription("openness", .7) {
		t.Fatalf("high score should use a text-pattern description: %+v", v.Traits[0])
	}
	if len(v.Traits[0].Evidence) != 4 {
		t.Fatalf("all contribution rows should be retained, got %d", len(v.Traits[0].Evidence))
	}
	if v.Traits[1].Name != "Neuroticism" || v.Traits[1].HasScoreRange {
		t.Fatalf("trait without range must set HasScoreRange=false: %+v", v.Traits[1])
	}
	if v.Traits[1].Score100 != 30 || v.Traits[1].SignalDescription != analyze.DimensionBandDescription("neuroticism", .3) {
		t.Fatalf("low score should use a text-pattern description: %+v", v.Traits[1])
	}
	if v.PercentileReferenceDescription != "Percentiles compare scores with 2400 texts in Reference essay sample. This is a comparison within that text sample, not a general-population estimate." {
		t.Fatalf("unexpected empirical reference description: %q", v.PercentileReferenceDescription)
	}
	if v.Coverage != 67 {
		t.Fatalf("coverage should be a 0-100 integer, got %d", v.Coverage)
	}
}

func TestBuildReportScoreBandWording(t *testing.T) {
	cases := []struct {
		name, label, description string
		score                    float64
	}{
		{"low", "low", analyze.DimensionBandDescription("openness", .349), 0.349},
		{"moderate threshold", "moderate", analyze.DimensionBandDescription("openness", .35), 0.35},
		{"moderate middle", "moderate", analyze.DimensionBandDescription("openness", .5), 0.5},
		{"high", "high", analyze.DimensionBandDescription("openness", .65), 0.65},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &Analysis{Traits: map[string]Trait{"openness": {Score: tc.score}}}
			got := BuildReport(a).Traits[0]
			if got.Label != tc.label || got.SignalDescription != tc.description {
				t.Fatalf("score %.3f produced label/description %q / %q", tc.score, got.Label, got.SignalDescription)
			}
			if tc.name == "moderate" && strings.Contains(strings.ToLower(got.SignalDescription), "curious") {
				t.Fatal("moderate band must not make a directional personality claim")
			}
		})
	}
}

func TestBuildReportValues(t *testing.T) {
	v := BuildReport(testAnalysis())
	if len(v.Values) != 2 {
		t.Fatalf("zero values must be dropped, got %d", len(v.Values))
	}
	if v.Values[0].Name != "Universalism" || v.Values[0].Percent != .88 || v.Values[0].Description == "" {
		t.Fatalf("top value wrong: %+v", v.Values[0])
	}
	if v.Values[1].Percent != .18 || v.Values[0].HasCounts {
		t.Fatalf("legacy percentages should remain recorded, with no invented counts: %+v", v.Values)
	}
}

func TestMainReadingHasOneScorePerMeasure(t *testing.T) {
	var out strings.Builder
	if err := RenderAnalysis("../../templates", testAnalysis(), &out, false); err != nil {
		t.Fatal(err)
	}
	main := strings.Split(out.String(), "Calculation details and limitations")[0]
	for _, removed := range []string{"percentile", "heuristic bounds", "Rough score range", "How to read these scores", "Summary of text signals", "What you value", "seen in:"} {
		if strings.Contains(main, removed) {
			t.Errorf("main reading still contains %q", removed)
		}
	}
	if strings.Count(main, `role="meter"`) != 7 || strings.Count(main, "Estimated text score") != 7 {
		t.Fatal("each available trait/summary must have exactly one score row")
	}
}

func TestPercentileReferenceDescriptions(t *testing.T) {
	fallback := &ingest.PercentileReference{Method: ingest.PercentileMethodNormalApproximation}
	if got := percentileText(61, fallback); got != "Model-estimated 61st percentile from a normal approximation." {
		t.Fatalf("unexpected fallback percentile text: %q", got)
	}
	if got := percentileReferenceDescription(fallback); !strings.Contains(got, "mean score of 50") || !strings.Contains(got, "standard deviation of 15") {
		t.Fatalf("fallback reference description omits model parameters: %q", got)
	}
	if got := percentileReferenceDescription(&ingest.PercentileReference{Method: ingest.PercentileMethodEmpirical, SampleSize: 321}); !strings.Contains(got, "321 texts") || strings.Contains(got, "texts in .") {
		t.Fatalf("empirical metadata without a corpus name should remain readable: %q", got)
	}
	if got := percentileText(91, nil); got != "Percentile method not recorded (91st percentile)." {
		t.Fatalf("missing metadata must not imply a method: %q", got)
	}
	if got := percentileReferenceDescription(nil); !strings.Contains(got, "not recorded") {
		t.Fatalf("missing reference metadata must remain explicit: %q", got)
	}
}

func TestRenderFragmentAndPage(t *testing.T) {
	a := testAnalysis()
	var frag strings.Builder
	if err := RenderAnalysis("../../templates", a, &frag, false); err != nil {
		t.Fatalf("fragment render: %v", err)
	}
	if strings.Contains(frag.String(), "<html") {
		t.Error("fragment render must not emit a full document")
	}
	for _, want := range []string{
		"Your writing profile", "Approximate reference-text percentile: 91st.",
		"Recorded heuristic bounds: 45–90/100 (unvalidated)", "Calculation details and limitations",
		"not a percentile range or a statistically validated confidence interval", "not a validated individual personality measure",
		"Percentiles compare scores with 2400 texts in Reference essay sample",
		">the<", ">happy<", "Words longer than six bytes (legacy model proxy)",
	} {
		if !strings.Contains(frag.String(), want) {
			t.Errorf("fragment missing %q", want)
		}
	}
	if strings.Count(frag.String(), "<td class=\"py-1.5 pr-3 text-stone-600\">") < 4 {
		t.Error("evidence table should render all four weighted category rows")
	}
	for _, layout := range []string{"max-w-[60rem]", "max-w-prose", "report-score-row", "role=\"region\"", "tabindex=\"0\"", "overflow-x-auto", "min-w-[50rem]"} {
		if !strings.Contains(frag.String(), layout) {
			t.Errorf("report is missing responsive/readability layout feature %q", layout)
		}
	}

	var page strings.Builder
	if err := RenderAnalysis("../../templates", a, &page, true); err != nil {
		t.Fatalf("page render: %v", err)
	}
	if !strings.Contains(page.String(), "<!DOCTYPE html>") || !strings.Contains(page.String(), "Your writing profile") {
		t.Error("full page must wrap the report body in a document")
	}
}

func TestIndexUsesConversationWidthAndKeepsReportUnconstrained(t *testing.T) {
	index, err := os.ReadFile("../../templates/index.html")
	if err != nil {
		t.Fatalf("read index template: %v", err)
	}
	markup := string(index)
	formSectionEnd := strings.Index(markup, "</section>")
	resultStart := strings.Index(markup, `<div id="result"`)
	if formSectionEnd < 0 || resultStart < 0 || formSectionEnd > resultStart {
		t.Fatal("inline report target must sit outside the narrow upload section")
	}
	if !strings.Contains(markup, `class="max-w-[60rem] mx-auto px-4 sm:px-6 lg:px-8 py-12"`) {
		t.Fatal("upload page should align to the report conversation width")
	}
	if !strings.Contains(markup, `class="max-w-[60rem] mx-auto px-4 sm:px-6 lg:px-8 mt-8 mb-12`) {
		t.Fatal("upload page footer should align to the same width")
	}
}

func stubAnalyzeFn(id string) ingest.AnalyzeFunc {
	return func(_ context.Context, text string) (ingest.AnalysisOutput, error) {
		traits := map[string]any{
			"openness": Trait{Score: 0.7, Percentile: 91, ConfidenceInterval: []float64{0.45, 0.9},
				Evidence: []ContributionJSON{{Category: "article", WordPercent: 9.3, Weight: -0.01, Contribution: -0.07, MatchedWords: []string{"the", "a"}}}},
		}
		return ingest.AnalysisOutput{
			AnalysisID:         id,
			WordCount:          570,
			DictionaryCoverage: 0.667,
			ConfidenceFlag:     "medium",
			Traits:             traits,
			Values:             map[string]float64{"value_universalism": 0.88},
			ValueEvidence:      map[string][]string{"value_universalism": {"just"}},
			PercentileReference: &ingest.PercentileReference{
				Method: ingest.PercentileMethodEmpirical, Corpus: "Reference essay sample", SampleSize: 2400,
			},
			Summary: SummaryVariables{},
		}, nil
	}
}

func postForm(target string, fields url.Values, hx bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/report", strings.NewReader(fields.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if hx {
		req.Header.Set("HX-Request", "true")
	}
	rec := httptest.NewRecorder()
	MakeHandleReportForm(1_000_000, "../../templates", zlogger.New("prod"), stubAnalyzeFn("round-trip-id"))(rec, req)
	return rec
}

func TestFormHandlerConsentRequired(t *testing.T) {
	rec := postForm("", url.Values{"text": {"a perfectly fine sample of text"}}, true)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing consent must 400, got %d", rec.Code)
	}
}

func TestFormHandlerRendersFragmentAndFullPage(t *testing.T) {
	fields := url.Values{
		"text":    {"a perfectly fine sample of text"},
		"consent": {"on"},
	}
	rec := postForm("", fields, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("htmx request must 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Error("HX-Request must render the fragment, not the full page")
	}
	if !strings.Contains(rec.Body.String(), "Your writing profile") {
		t.Error("fragment must contain the report body")
	}
	for _, want := range []string{"Text-based measures", `aria-label="Big Five bands"`, `id="band-openness" role="tooltip"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("HTMX report is missing %q", want)
		}
	}

	rec = postForm("", fields, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("no-JS request must 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Error("non-HTMX request must render the full page")
	}
	if !strings.Contains(rec.Body.String(), "Your writing profile") {
		t.Error("full page must contain the report body")
	}
	for _, want := range []string{"Text-based measures", `aria-label="Big Five bands"`, `id="band-openness" role="tooltip"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("full-page report is missing %q", want)
		}
	}
}

func TestAnalysisOutputRoundTrip(t *testing.T) {
	// The stub handler already exercises this via Marshal/Unmarshal, but
	// assert the JSON contract explicitly: an AnalysisOutput encodes into a
	// shape Analysis can decode.
	res, err := stubAnalyzeFn("x")(nil, "text")
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var a Analysis
	if err := json.Unmarshal(out, &a); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if a.Traits["openness"].Score != 0.7 {
		t.Fatalf("traits did not survive the round trip: %+v", a.Traits)
	}
	// Regression guard: AnalysisOutput must carry snake_case tags. Without
	// them, fields like value_evidence silently drop during the round trip
	// (case-insensitive matching cannot bridge the underscore).
	if a.AnalysisID != "x" || a.WordCount != 570 || a.ConfidenceFlag != "medium" {
		t.Fatalf("scalar fields lost in round trip: %+v", a)
	}
	if len(a.ValueEvidence["value_universalism"]) != 1 || a.ValueEvidence["value_universalism"][0] != "just" {
		t.Fatalf("value evidence lost in round trip: %+v", a.ValueEvidence)
	}
	if a.PercentileReference == nil || a.PercentileReference.Method != ingest.PercentileMethodEmpirical || a.PercentileReference.SampleSize != 2400 {
		t.Fatalf("percentile reference lost in round trip: %+v", a.PercentileReference)
	}
	if got := a.Traits["openness"].Evidence[0].MatchedWords; len(got) != 2 || got[0] != "the" {
		t.Fatalf("matched word samples lost in round trip: %v", got)
	}
}

func TestLegacyPayloadWithoutOptionalEvidenceFields(t *testing.T) {
	const legacy = `{"analysis_id":"legacy","word_count":500,"dictionary_coverage":0.5,"confidence_flag":"medium","traits":{"openness":{"score":0.5,"percentile":50,"confidence_interval":[0.3,0.7],"evidence":[{"category":"article","word_percent":5,"weight":0.01,"contribution":0.05}]}},"values":{},"summary":{}}`
	var a Analysis
	if err := json.Unmarshal([]byte(legacy), &a); err != nil {
		t.Fatalf("decode legacy analysis: %v", err)
	}
	if a.PercentileReference != nil || len(a.Traits["openness"].Evidence[0].MatchedWords) != 0 {
		t.Fatalf("legacy fields should remain optional: %+v", a)
	}
	var rendered strings.Builder
	if err := RenderAnalysis("../../templates", &a, &rendered, false); err != nil {
		t.Fatalf("render legacy analysis: %v", err)
	}
	for _, want := range []string{"Percentile method not recorded", "Examples unavailable for this saved result", "article", "5.00%"} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("legacy report missing available information %q", want)
		}
	}
}
