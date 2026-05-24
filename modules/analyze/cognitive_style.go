package analyze

import "math"

// Cognitive style coefficients distinguish systematic (central-route) from
// intuitive (peripheral-route) processing based on language patterns.
// Higher = systematic/analytical; lower = intuitive/heuristic.
//
// Systematic markers: cognitive process words, causal reasoning, complex
// vocabulary, definitive claims, and analytical language indicate deliberate
// elaboration.
//
// Intuitive markers: perceptual/sensory language, personal pronouns,
// present-tense focus, and tentative hedging indicate heuristic, felt-sense
// processing.
//
// Source constructs:
//   - Petty, R.E., & Cacioppo, J.T. (1986). The Elaboration Likelihood Model.
//   - Pennebaker, J.W., & King, L.A. (1999). Linguistic styles: Language use
//     as an individual difference.
//
// Coefficients are synthesised from the source construct definitions;
// no published regression table exists matching LIWC categories to ELM
// processing style. The weights reflect directional hypotheses consistent
// with the ELM framework.
var cognitiveStyleCoefficients = map[string]float64{
	// Systematic (+) — openminded depth, causality, precision, formality.
	"cognitive_process":  0.008,
	"cause":              0.008,
	"certainty":          0.008,
	"big_words":          0.012,
	"analytic_thinking":  0.010,
	// Intuitive (−) — perceptual, personal, immediate, uncertain.
	"sensation":          -0.008,
	"pronoun":            -0.008,
	"present_focus":      -0.008,
	"intuitive_thinking": -0.010,
	"tentative":          -0.006,
}

// ComputeCognitiveStyle computes a cognitive processing style score from categories.
func ComputeCognitiveStyle(fv FeatureVector) float64 {
	score := 0.50
	for cat, weight := range cognitiveStyleCoefficients {
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

// ComputeCognitiveStyleLabel returns a human-readable label for the score.
func ComputeCognitiveStyleLabel(score float64) string {
	if score > 0.65 {
		return "systematic"
	}
	if score < 0.35 {
		return "intuitive"
	}
	return "mixed"
}
