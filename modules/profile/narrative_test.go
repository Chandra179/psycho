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
			"cognitive_style":    {Score: 0.68, Percentile: 93, ConfidenceInterval: []float64{0.58, 0.78}},
			"need_for_closure":   {Score: 0.35, Percentile: 11, ConfidenceInterval: []float64{0.25, 0.45}},
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
		"Cognitive Style",
		"Need for Closure",
		"Analytical Thinking",
		"Clout",
		"Authenticity",
		"Emotional Tone",
		"promotion_focus",
		"high",
		"98th percentile",
		"95% CI",
		"Template-based synthesis, not a clinical assessment",
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
			"cognitive_style":    {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
			"need_for_closure":   {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.30, 0.70}},
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
			"cognitive_style":    {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.40, 0.60}},
			"need_for_closure":   {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.40, 0.60}},
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

func TestSummaryBandFormal(t *testing.T) {
	if got := analyze.SummaryBandFormal("clout", 0.70); got != "confident/dominant" {
		t.Errorf("SummaryBandFormal(clout, 0.70) = %q; want confident/dominant", got)
	}
	if got := analyze.SummaryBandFormal("clout", 0.30); got != "submissive/uncertain" {
		t.Errorf("SummaryBandFormal(clout, 0.30) = %q; want submissive/uncertain", got)
	}
	if got := analyze.SummaryBandFormal("clout", 0.50); got != "moderate" {
		t.Errorf("SummaryBandFormal(clout, 0.50) = %q; want moderate", got)
	}
	if got := analyze.SummaryBandCompact("clout", 0.70); got != "confident" {
		t.Errorf("SummaryBandCompact(clout, 0.70) = %q; want confident", got)
	}
	if got := analyze.SummaryTone(0.70); got != "positive" {
		t.Errorf("SummaryTone(0.70) = %q; want positive", got)
	}
	if got := analyze.SummaryTone(0.30); got != "negative" {
		t.Errorf("SummaryTone(0.30) = %q; want negative", got)
	}
	if got := analyze.SummaryTone(0.50); got != "neutral" {
		t.Errorf("SummaryTone(0.50) = %q; want neutral", got)
	}
}
