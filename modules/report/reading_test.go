package report

import (
	"encoding/json"
	"html"
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
	if len(v.Traits) != 9 || !v.HasBigFive || len(v.Summary) != 4 {
		t.Fatalf("missing measures: %+v", v)
	}
	for _, row := range v.Traits {
		if row.Label != analyze.DimensionLabel(row.Key, .65) || row.Score100 != 65 {
			t.Fatalf("changed score/band: %+v", row)
		}
	}
	// Every measure uses the shared rule: exactly 65/100 is the top band, as the legend says.
	if v.Traits[0].Label != "high" || v.Traits[5].Label != "promotion_focus" || v.Traits[6].Label != "high" || v.Traits[7].Label != "systematic" || v.Traits[8].Label != "high" {
		t.Fatalf("65/100 boundary rules drifted: %+v", v.Traits)
	}
	if !strings.Contains(v.Traits[6].SignalDescription, "65/100 or above") {
		t.Fatalf("top-band wording drifted: %q", v.Traits[6].SignalDescription)
	}
	if v.Bands[0].Range != "0–34" || v.Bands[1].Range != "35–64" || v.Bands[2].Range != "65–100" {
		t.Fatalf("wrong legend: %+v", v.Bands)
	}
	wantNames := []string{"Openness", "Conscientiousness", "Extraversion", "Agreeableness", "Neuroticism", "Goals: gain vs. safety", "Need for Cognition", "Cognitive Style", "Preference for certainty"}
	for _, variant := range []string{"fragment", "full page", "standalone"} {
		t.Run(variant, func(t *testing.T) {
			var out strings.Builder
			var err error
			switch variant {
			case "fragment":
				err = RenderAnalysis("../../templates", a, &out, false)
			case "full page":
				err = RenderAnalysis("../../templates", a, &out, true)
			case "standalone":
				err = RenderStandaloneAnalysis("../../templates", a, &out)
			}
			if err != nil {
				t.Fatal(err)
			}
			rendered := out.String()
			if strings.Count(rendered, `role="meter"`) != 13 {
				t.Fatalf("got %d score rows, want 9 measures and 4 summaries", strings.Count(rendered, `role="meter"`))
			}
			if strings.Count(rendered, "Text-based measures") != 1 || strings.Contains(rendered, "Big Five text signals") || strings.Contains(rendered, "Additional text measures") {
				t.Fatal("score sections were not combined under the single heading")
			}
			if !strings.Contains(rendered, `aria-label="Big Five bands"`) || !strings.Contains(rendered, "Measures beyond the Big Five are project-defined language proxies") {
				t.Fatal("combined section is missing its scoped legend or proxy note")
			}
			for _, row := range v.Traits {
				tooltip := `id="band-` + row.Key + `" role="tooltip"`
				if !strings.Contains(rendered, tooltip) || !strings.Contains(rendered, html.EscapeString(row.SignalDescription)) {
					t.Errorf("measure %q is missing its own band explanation", row.Key)
				}
			}
			lastPosition := -1
			for _, name := range wantNames {
				needle := `aria-label="` + name + ` estimated text score"`
				position := strings.Index(rendered, needle)
				if position < 0 || strings.Count(rendered, needle) != 1 || position <= lastPosition {
					t.Fatalf("measure %q missing, repeated, or out of order", name)
				}
				lastPosition = position
			}
		})
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
