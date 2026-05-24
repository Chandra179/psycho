package profile

import (
	"testing"

	"psycho/modules/analyze"
)

func TestMarotoPDFGenerate(t *testing.T) {
	gen := NewMarotoPDFGenerator()

	prof := Profile{
		AnalysisID:     "test-123",
		ConfidenceFlag: "high",
		Traits: map[string]TraitResult{
			"openness":          {Score: 0.65, Percentile: 75, ConfidenceInterval: []float64{0.55, 0.75}},
			"conscientiousness": {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.40, 0.60}},
			"extraversion":      {Score: 0.40, Percentile: 30, ConfidenceInterval: []float64{0.30, 0.50}},
			"agreeableness":     {Score: 0.55, Percentile: 60, ConfidenceInterval: []float64{0.45, 0.65}},
			"neuroticism":       {Score: 0.35, Percentile: 20, ConfidenceInterval: []float64{0.25, 0.45}},
			"regulatory_focus":   {Score: 0.60, Percentile: 70, ConfidenceInterval: []float64{0.50, 0.70}},
			"need_for_cognition": {Score: 0.45, Percentile: 40, ConfidenceInterval: []float64{0.35, 0.55}},
			"cognitive_style":    {Score: 0.55, Percentile: 60, ConfidenceInterval: []float64{0.45, 0.65}},
			"need_for_closure":   {Score: 0.50, Percentile: 50, ConfidenceInterval: []float64{0.40, 0.60}},
		},
		Values: map[string]float64{
			"value_self_direction": 3.4,
			"value_universalism":   2.8,
			"value_benevolence":    2.1,
		},
		Summary: analyze.SummaryVariables{
			AnalyticalThinking: 0.65,
			Clout:              0.55,
			Authenticity:       0.70,
			EmotionalTone:      0.60,
		},
		Narrative: "This is a test narrative.",
	}

	pdf, err := gen.Generate(prof)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(pdf) < 100 {
		t.Errorf("PDF too small: %d bytes", len(pdf))
	}

	pdfHeader := "%PDF-"
	if string(pdf[:len(pdfHeader)]) != pdfHeader {
		t.Errorf("PDF missing %%PDF- header, got: %s", string(pdf[:min(20, len(pdf))]))
	}
}

func TestMarotoPDFGenerateEmptyTraits(t *testing.T) {
	gen := NewMarotoPDFGenerator()

	prof := Profile{
		AnalysisID:     "test-empty",
		ConfidenceFlag: "low",
		Traits:         map[string]TraitResult{},
		Values:         map[string]float64{},
		Summary:        analyze.SummaryVariables{},
	}

	pdf, err := gen.Generate(prof)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(pdf) < 50 {
		t.Errorf("PDF too small: %d bytes", len(pdf))
	}
}
