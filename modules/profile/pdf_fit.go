package profile

import "psycho/modules/report"

// pdfFitNotes returns the same fit notes the HTML report shows: top holds
// sentences for the opening notice, byKey the one-line reason per measure.
// Older saved profiles without recorded calculation details have no word
// count, so they get no notes rather than guessed ones.
func pdfFitNotes(p Profile) (top []string, byKey map[string]string) {
	if p.CalculationDetails == nil {
		return nil, nil
	}
	a := &report.Analysis{
		WordCount:          p.CalculationDetails.WordCount,
		Traits:             map[string]report.Trait{},
		Summary:            report.SummaryVariables(p.Summary),
		CalculationDetails: p.CalculationDetails,
	}
	for k, t := range p.Traits {
		a.Traits[k] = report.Trait{Score: t.Score}
	}
	return report.FitNotes(a)
}
