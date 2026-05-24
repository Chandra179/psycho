package profile

import (
	"strings"
	"testing"

	"psycho/modules/analyze"
)

func TestNewTemplateNarrativeGenerator(t *testing.T) {
	g := NewTemplateNarrativeGenerator()
	if g == nil {
		t.Fatal("NewTemplateNarrativeGenerator() returned nil")
	}
}

func TestTemplateNarrativeGenerator_GeneratesAllSections(t *testing.T) {
	g := NewTemplateNarrativeGenerator()
	sv := analyze.SummaryVariables{
		AnalyticalThinking: 0.72,
		Clout:              0.45,
		Authenticity:       0.88,
		EmotionalTone:      0.31,
	}
	prof := Profile{
		AnalysisID:     "test-1",
		ConfidenceFlag: "high",
		Traits: map[string]TraitResult{
			"openness":           {Score: 0.75, Percentile: 98, ConfidenceInterval: []float64{0.65, 0.85}},
			"conscientiousness":  {Score: 0.60, Percentile: 80, ConfidenceInterval: []float64{0.50, 0.70}},
			"extraversion":       {Score: 0.45, Percentile: 34, ConfidenceInterval: []float64{0.35, 0.55}},
			"agreeableness":      {Score: 0.55, Percentile: 66, ConfidenceInterval: []float64{0.45, 0.65}},
			"neuroticism":        {Score: 0.30, Percentile: 5, ConfidenceInterval: []float64{0.20, 0.40}},
			"regulatory_focus":   {Score: 0.70, Percentile: 95, ConfidenceInterval: []float64{0.60, 0.80}},
			"need_for_cognition": {Score: 0.82, Percentile: 99, ConfidenceInterval: []float64{0.72, 0.92}},
		},
		Summary: sv,
	}

	narrative := g.GenerateSynthesis(prof)

	checks := []string{
		"Psychological Profile",
		"Confidence level:** high",
		"Openness",
		"Conscientiousness",
		"Extraversion",
		"Agreeableness",
		"Neuroticism",
		"Regulatory Focus",
		"Need for Cognition",
		"Analytical Thinking",
		"Clout",
		"Authenticity",
		"Emotional Tone",
		"promotion_focus",
		"high",
		"98th percentile",
		"95% CI",
		"Not a clinical assessment",
	}
	for _, want := range checks {
		if !strings.Contains(narrative, want) {
			t.Errorf("narrative missing expected text %q", want)
		}
	}
}

func TestTemplateNarrativeGenerator_LowConfidence(t *testing.T) {
	g := NewTemplateNarrativeGenerator()
	prof := Profile{
		AnalysisID:     "test-2",
		ConfidenceFlag: "low",
		Traits: map[string]TraitResult{
			"openness":           {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
			"conscientiousness":  {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
			"extraversion":       {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
			"agreeableness":      {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
			"neuroticism":        {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
			"regulatory_focus":   {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
			"need_for_cognition": {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
		},
	}

	narrative := g.GenerateSynthesis(prof)
	if !strings.Contains(narrative, "Confidence level:** low") {
		t.Error("low confidence narrative should report low confidence")
	}
}

func TestTemplateNarrativeGenerator_EdgeScores(t *testing.T) {
	g := NewTemplateNarrativeGenerator()
	prof := Profile{
		AnalysisID:     "test-3",
		ConfidenceFlag: "high",
		Traits: map[string]TraitResult{
			"openness":           {Score: 0.10, Percentile: 1, ConfidenceInterval: []float64{0.00, 0.20}},
			"conscientiousness":  {Score: 0.90, Percentile: 99, ConfidenceInterval: []float64{0.80, 1.00}},
			"extraversion":       {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.40, 0.60}},
			"agreeableness":      {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.40, 0.60}},
			"neuroticism":        {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.40, 0.60}},
			"regulatory_focus":   {Score: 0.20, Percentile: 1, ConfidenceInterval: []float64{0.10, 0.30}},
			"need_for_cognition": {Score: 0.30, Percentile: 5, ConfidenceInterval: []float64{0.20, 0.40}},
		},
	}

	narrative := g.GenerateSynthesis(prof)

	if !strings.Contains(narrative, "low") {
		t.Error("low-scored traits should produce 'low' labels")
	}
	if !strings.Contains(narrative, "prevention_focus") {
		t.Error("regulatory focus below 0.35 should label prevention_focus")
	}
	if strings.Contains(narrative, "promotion_focus") {
		t.Error("low regulatory focus should not be promotion_focus")
	}
}

func TestTraitDisplayName(t *testing.T) {
	cases := []struct{ key, want string }{
		{"openness", "Openness"},
		{"conscientiousness", "Conscientiousness"},
		{"extraversion", "Extraversion"},
		{"agreeableness", "Agreeableness"},
		{"neuroticism", "Neuroticism"},
		{"regulatory_focus", "Regulatory Focus"},
		{"need_for_cognition", "Need for Cognition"},
		{"unknown", "unknown"},
	}
	for _, c := range cases {
		got := traitDisplayName(c.key)
		if got != c.want {
			t.Errorf("traitDisplayName(%q) = %q; want %q", c.key, got, c.want)
		}
	}
}

func TestTraitLabel(t *testing.T) {
	if label := traitLabel("openness", 0.70); label != "high" {
		t.Errorf("traitLabel(openness, 0.70) = %q; want high", label)
	}
	if label := traitLabel("openness", 0.30); label != "low" {
		t.Errorf("traitLabel(openness, 0.30) = %q; want low", label)
	}
	if label := traitLabel("openness", 0.50); label != "moderate" {
		t.Errorf("traitLabel(openness, 0.50) = %q; want moderate", label)
	}
}

func TestSummaryLabel(t *testing.T) {
	if got := summaryLabel(0.70, "high", "low"); got != "high" {
		t.Errorf("summaryLabel(0.70) = %q; want high", got)
	}
	if got := summaryLabel(0.30, "high", "low"); got != "low" {
		t.Errorf("summaryLabel(0.30) = %q; want low", got)
	}
	if got := summaryLabel(0.50, "high", "low"); got != "moderate" {
		t.Errorf("summaryLabel(0.50) = %q; want moderate", got)
	}
}

func TestToneLabel(t *testing.T) {
	if got := toneLabel(0.70); got != "positive" {
		t.Errorf("toneLabel(0.70) = %q; want positive", got)
	}
	if got := toneLabel(0.30); got != "negative" {
		t.Errorf("toneLabel(0.30) = %q; want negative", got)
	}
	if got := toneLabel(0.50); got != "neutral" {
		t.Errorf("toneLabel(0.50) = %q; want neutral", got)
	}
}
