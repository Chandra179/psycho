package report

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
)

func TestReadingPreservesAllMeasuresAndCanonicalBoundaries(t *testing.T) {
	a := testAnalysis()
	for _, key := range traitOrder {
		a.Traits[key] = Trait{Score: .65, Percentile: 78, ConfidenceInterval: []float64{.4, .9}}
	}
	v := BuildReport(a)
	if len(v.BigFive) != 5 || len(v.Additional) != 4 || len(v.Summary) != 4 {
		t.Fatalf("missing measures: %+v", v)
	}
	for _, row := range v.Traits {
		if row.Label != analyze.DimensionLabel(row.Key, .65) || row.Score100 != 65 {
			t.Fatalf("changed score/band: %+v", row)
		}
	}
	if v.BigFive[0].Label != "high" || v.Additional[1].Label != "moderate" || !strings.Contains(v.Additional[1].SignalDescription, "35–65/100") {
		t.Fatal("65/100 boundary rules drifted")
	}
	if v.Bands[0].Range != "0–34" || v.Bands[1].Range != "35–64" || v.Bands[2].Range != "65–100" {
		t.Fatalf("wrong legend: %+v", v.Bands)
	}
	var out strings.Builder
	if err := RenderAnalysis("../../templates", a, &out, false); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), `role="progressbar"`) != 13 {
		t.Fatal("not all 13 scores rendered")
	}
}

func TestValueExcerptsRoundTripCountsAndEscaping(t *testing.T) {
	a := testAnalysis()
	attack := `<img src=x onerror=alert(1)>`
	a.ValueExcerpts = map[string][]ingest.TextExcerpt{"value_universalism": {{Segments: []ingest.TextSegment{{Text: "A "}, {Text: attack, Matched: true}, {Text: " world."}}}}}
	a.CalculationDetails = &analyze.CalculationDetails{Values: map[string]analyze.ValueCalculation{"value_universalism": {MatchedCount: 5, TotalWords: 570, Percent: .88}}}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var restored Analysis
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.ValueExcerpts, a.ValueExcerpts) {
		t.Fatal("excerpts lost in JSON")
	}
	v := BuildReport(&restored)
	if !v.Values[0].HasCounts || v.Values[0].MatchedCount != 5 || v.Values[0].TotalWords != 570 {
		t.Fatal("did not consume recorded counts")
	}
	for _, fullPage := range []bool{false, true} {
		var out strings.Builder
		if err := RenderAnalysis("../../templates", &restored, &out, fullPage); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), attack) || !strings.Contains(out.String(), "<mark>&lt;img") || !strings.Contains(out.String(), "5 of 570 words") {
			t.Fatalf("unsafe or missing context: %s", out.String())
		}
	}
}

func TestLegacyValuesDoNotInventExcerptsOrCounts(t *testing.T) {
	var out strings.Builder
	if err := RenderAnalysis("../../templates", testAnalysis(), &out, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Counts not recorded", "Text excerpts were not recorded", "Sampled matching words: just, world", "0.88% of all words"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing legacy fallback %q", want)
		}
	}
}
