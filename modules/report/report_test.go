package report

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"psycho/modules/ingest"
	"psycho/zlogger"
)

func testAnalysis() *Analysis {
	return &Analysis{
		AnalysisID:         "test-id",
		WordCount:          570,
		DictionaryCoverage: 0.667,
		ConfidenceFlag:     "medium",
		Traits: map[string]Trait{
			"openness": {Score: 0.7, Percentile: 91, ConfidenceInterval: []float64{0.45, 0.9},
				Evidence: []ContributionJSON{{Category: "article", WordPercent: 9.3, Weight: -0.01, Contribution: -0.07}}},
			"neuroticism": {Score: 0.3, Percentile: 8},
			// agreeableness deliberately absent: must be skipped, not zero-filled.
		},
		Values:         map[string]float64{"value_universalism": 0.88, "value_achievement": 0.18},
		ValueEvidence:  map[string][]string{"value_universalism": {"just", "world"}},
		Summary:        SummaryVariables{AnalyticalThinking: 0.16, Clout: 0.06, Authenticity: 0.71, EmotionalTone: 0.48},
	}
}

func TestBuildReportSuperset(t *testing.T) {
	v := BuildReport(testAnalysis())

	if got := len(v.Traits); got != 2 {
		t.Fatalf("expected 2 traits (missing key skipped), got %d", got)
	}
	if v.Traits[0].Name != "Openness" || v.Traits[0].Label != "high" {
		t.Fatalf("unexpected first trait: %+v", v.Traits[0])
	}
	if !v.Traits[0].HasCI || v.Traits[0].CILo != 45 || v.Traits[0].CIHi != 90 {
		t.Fatalf("CI not scaled to 0-100: %+v", v.Traits[0])
	}
	if v.Traits[0].Blurb != traitBlurbs["openness"][0] {
		t.Fatalf("high score should take the above-average blurb, got %q", v.Traits[0].Blurb)
	}
	if v.Traits[1].Name != "Neuroticism" || v.Traits[1].HasCI {
		t.Fatalf("trait without CI must set HasCI=false: %+v", v.Traits[1])
	}
	if v.Traits[1].Blurb != traitBlurbs["neuroticism"][1] {
		t.Fatalf("low score should take the below-average blurb, got %q", v.Traits[1].Blurb)
	}
	if v.Coverage != 67 {
		t.Fatalf("coverage should be a 0-100 integer, got %d", v.Coverage)
	}
}

func TestBuildReportValues(t *testing.T) {
	v := BuildReport(testAnalysis())
	if len(v.Values) != 2 {
		t.Fatalf("zero values must be dropped, got %d", len(v.Values))
	}
	if v.Values[0].Name != "Universalism" || v.Values[0].Rank != 1 || v.Values[0].RelWidth != 100 {
		t.Fatalf("top value wrong: %+v", v.Values[0])
	}
	if v.Values[1].RelWidth != 20 {
		t.Fatalf("relative bar width should be 20%% of top, got %d", v.Values[1].RelWidth)
	}
}

func TestGenerateSnapshot(t *testing.T) {
	s := generateSnapshot(testAnalysis())
	for _, want := range []string{"Openness", "higher than 91%", "personal and honest", "reserved rather than dominant", "Universalism", "just"} {
		if !strings.Contains(s, want) {
			t.Errorf("snapshot missing %q: %s", want, s)
		}
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
	for _, want := range []string{"Your personality reading", "Higher than <b>91%</b>", "could plausibly land anywhere from 45 to 90", "Show the evidence behind every score", "Not a clinical or diagnostic assessment"} {
		if !strings.Contains(frag.String(), want) {
			t.Errorf("fragment missing %q", want)
		}
	}

	var page strings.Builder
	if err := RenderAnalysis("../../templates", a, &page, true); err != nil {
		t.Fatalf("page render: %v", err)
	}
	if !strings.Contains(page.String(), "<!DOCTYPE html>") || !strings.Contains(page.String(), "Your personality reading") {
		t.Error("full page must wrap the report body in a document")
	}
}

func stubAnalyzeFn(id string) ingest.AnalyzeFunc {
	return func(_ context.Context, sourceType, sourceDate, text string) (ingest.AnalysisOutput, error) {
		traits := map[string]any{
			"openness": Trait{Score: 0.7, Percentile: 91, ConfidenceInterval: []float64{0.45, 0.9}},
		}
		return ingest.AnalysisOutput{
			AnalysisID:         id,
			WordCount:          570,
			DictionaryCoverage: 0.667,
			ConfidenceFlag:     "medium",
			Traits:             traits,
			Values:             map[string]float64{"value_universalism": 0.88},
			ValueEvidence:      map[string][]string{"value_universalism": {"just"}},
			Summary:            SummaryVariables{},
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
	if !strings.Contains(rec.Body.String(), "Your personality reading") {
		t.Error("fragment must contain the report body")
	}

	rec = postForm("", fields, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("no-JS request must 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Error("non-HTMX request must render the full page")
	}
}

func TestAnalysisOutputRoundTrip(t *testing.T) {
	// The stub handler already exercises this via Marshal/Unmarshal, but
	// assert the JSON contract explicitly: an AnalysisOutput encodes into a
	// shape Analysis can decode.
	res, err := stubAnalyzeFn("x")(nil, "paste", "", "text")
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
}
