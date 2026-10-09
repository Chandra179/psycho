package analyze

// Need for cognition coefficients: analytic words increase the score,
// intuitive words decrease it. Score ranges [0,1] with 0.50 neutral.
// Source: Cacioppo, J.T. & Petty, R.E. (1982). The need for cognition.
//
//	Journal of Personality and Social Psychology, 42(1), 116-131.
//
// The paper introduces the Need for Cognition construct and scale, not word lists.
// The analytic/intuitive categories and weights are this project's own
// operationalization; no published word-list mapping exists.
var needCogCoefficients = map[string]float64{
	"analytic_thinking":  0.025,
	"intuitive_thinking": -0.015,
}

func ComputeNeedForCognition(fv FeatureVector) float64 {
	return ComputeNeedForCognitionCalculation(fv).FinalScore
}

func ComputeNeedForCognitionCalculation(fv FeatureVector) *ScoreCalculation {
	return computeWeightedScore(fv, needCogCoefficients)
}

func ComputeNeedForCognitionLabel(score float64) string {
	return HighModerateLow(score)
}
