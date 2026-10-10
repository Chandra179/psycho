package profile

import (
	"strings"
	"testing"

	"psycho/modules/analyze"
)

func TestPDFFitNotesMatchFormalProse(t *testing.T) {
	p := Profile{
		Traits:  map[string]TraitResult{"cognitive_style": {Score: 0.75}, "openness": {Score: 0.6}},
		Summary: analyze.SummaryVariables{Authenticity: 0.1},
		CalculationDetails: &analyze.CalculationDetails{
			WordCount:    1000,
			BigWordCount: 400,
		},
	}
	top, byKey := pdfFitNotes(p)
	if len(top) == 0 || !strings.Contains(top[0], "formal writing") {
		t.Fatalf("top = %v", top)
	}
	if byKey["openness"] == "" || byKey["extraversion"] == "" {
		t.Fatalf("byKey = %v", byKey)
	}
	if _, err := NewMarotoPDFGenerator().Generate(p); err != nil {
		t.Fatal(err)
	}
}

func TestPDFFitNotesSkipOldProfiles(t *testing.T) {
	top, byKey := pdfFitNotes(Profile{Summary: analyze.SummaryVariables{Authenticity: 0.1}})
	if top != nil || byKey != nil {
		t.Fatalf("no recorded details means no notes: %v %v", top, byKey)
	}
}
