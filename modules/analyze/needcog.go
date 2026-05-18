package analyze

import "math"

// Need for cognition coefficients: analytic words increase the score,
// intuitive words decrease it. Score ranges [0,1] with 0.50 neutral.
// Source: Cacioppo, J.T. & Petty, R.E. (1982). The need for cognition.
//   Journal of Personality and Social Psychology, 42(1), 116-131.
var needCogCoefficients = map[string]float64{
	"analytic_thinking":  0.025,
	"intuitive_thinking": -0.015,
}

func ComputeNeedForCognition(fv FeatureVector) float64 {
	score := 0.50
	for cat, weight := range needCogCoefficients {
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

func ComputeNeedForCognitionLabel(score float64) string {
	if score > 0.65 {
		return "high"
	}
	if score < 0.35 {
		return "low"
	}
	return "moderate"
}
