package analyze

import "math"

// Regulatory focus coefficients: promotion words increase the score,
// prevention words decrease it. Score ranges [0,1] with 0.50 neutral.
// Source: Higgins, E.T. (1997). Beyond pleasure and pain.
//   American Psychologist, 52(12), 1280-1300.
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
