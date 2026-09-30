package analyze

import "math"

// Regulatory focus coefficients: promotion words increase the score,
// prevention words decrease it. Score ranges [0,1] with 0.50 neutral.
// Source: Higgins, E.T. (1997). Beyond pleasure and pain.
//   American Psychologist, 52(12), 1280-1300.
//
// The paper provides the theoretical construct only — no word lists or
// weights. The promotion/prevention categories in dictionary.json and the
// weights below are this project's own operationalization.
//
// The score is deliberately bipolar (prevention subtracts from promotion).
// Higgins' framework treats promotion and prevention strength as largely
// independent dimensions, so a prevention-heavy text is not necessarily
// low in promotion; split into two independent scores if that distinction
// matters for a use case.
var regFocusCoefficients = map[string]float64{
	"promotion_focus":  0.020,
	"prevention_focus": -0.020,
}

func ComputeRegulatoryFocus(fv FeatureVector) float64 {
	score := 0.50
	for cat, weight := range regFocusCoefficients {
		pct := fv.CategoryPercents[Category(cat)]
		score += weight * pct
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return math.Round(score*100) / 100
}

func ComputeRegulatoryFocusLabel(score float64) string {
	if score > 0.65 {
		return "promotion_focus"
	}
	if score < 0.35 {
		return "prevention_focus"
	}
	return "balanced"
}
