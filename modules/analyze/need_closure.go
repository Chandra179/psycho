package analyze

// Need for closure coefficients: certainty words increase the score,
// tentative words decrease it. Score ranges [0,1] with 0.50 neutral.
// Higher = strong need for closure; lower = high tolerance for ambiguity.
// Source: Webster, D.M., & Kruglanski, A.W. (1994). Individual differences
//
//	in need for cognitive closure. Journal of Personality and Social Psychology.
//
// The paper provides the Need for Closure Scale, not word lists. The
// certainty/tentative markers and weights are this project's own
// operationalization; no published word-list mapping exists.
var needClosureCoefficients = map[string]float64{
	"certainty": 0.020,
	"tentative": -0.025,
}

// ComputeNeedForClosure computes a need for cognitive closure score from categories.
func ComputeNeedForClosure(fv FeatureVector) float64 {
	return ComputeNeedForClosureCalculation(fv).FinalScore
}

func ComputeNeedForClosureCalculation(fv FeatureVector) *ScoreCalculation {
	return computeWeightedScore(fv, needClosureCoefficients)
}

// ComputeNeedForClosureLabel returns a human-readable label for the score.
func ComputeNeedForClosureLabel(score float64) string {
	return HighModerateLow(score)
}
