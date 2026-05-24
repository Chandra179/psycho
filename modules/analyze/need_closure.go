package analyze

import "math"

// Need for closure coefficients: certainty words increase the score,
// tentative words decrease it. Score ranges [0,1] with 0.50 neutral.
// Higher = strong need for closure; lower = high tolerance for ambiguity.
// Source: Webster, D.M., & Kruglanski, A.W. (1994). Individual differences
//
//	in need for cognitive closure. Journal of Personality and Social Psychology.
var needClosureCoefficients = map[string]float64{
	"certainty":  0.020,
	"tentative": -0.025,
}

// ComputeNeedForClosure computes a need for cognitive closure score from categories.
func ComputeNeedForClosure(fv FeatureVector) float64 {
	score := 0.50
	for cat, weight := range needClosureCoefficients {
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

// ComputeNeedForClosureLabel returns a human-readable label for the score.
func ComputeNeedForClosureLabel(score float64) string {
	if score > 0.65 {
		return "high"
	}
	if score < 0.35 {
		return "low"
	}
	return "moderate"
}
