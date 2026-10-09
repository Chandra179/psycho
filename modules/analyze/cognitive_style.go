package analyze

// Cognitive style follows the categorical-versus-dynamic language index of
// Pennebaker et al. (2014), built from eight function-word categories: articles
// and prepositions mark categorical (formal, hierarchical) writing, while
// personal pronouns, impersonal pronouns, auxiliary verbs, adverbs,
// conjunctions and negations mark dynamic (narrative, here-and-now) writing.
// The paper's index is unit weighted (30 + article + preposition minus the
// other six, in percent of words) and does not use long words.
//
// Source:
//
//	Pennebaker, J.W., Chung, C.K., Frazee, J., Lavergne, G.M., & Beaver, D.I.
//	(2014). When small words foretell academic success: The case of college
//	admissions essays. PLoS ONE, 9(12), e115844.
//
// Higher = categorical (labelled "systematic"); lower = dynamic ("intuitive").
// Every category carries the same weight, mirroring the paper's unit weights.
// The 0.01 per percentage point scale is a project assumption chosen so the
// score spreads like the other proxies; the paper validates the index against
// college grades, not personality or any other outcome here. The index sums to
// about -19.7 points on the 3,992-post reference corpus (SD 10.6), the way the
// paper adds 30 to stay positive, so the baseline is 0.50 + 0.01 * 19.7 = 0.70
// and typical text starts near the middle instead of being clamped at 0.
const (
	cognitiveStyleUnitWeight = 0.01
	cognitiveStyleBaseline   = 0.70
)

var cognitiveStyleCoefficients = map[string]float64{
	"article":            cognitiveStyleUnitWeight,
	"preposition":        cognitiveStyleUnitWeight,
	"personal_pronoun":   -cognitiveStyleUnitWeight,
	"impersonal_pronoun": -cognitiveStyleUnitWeight,
	"auxiliary_verb":     -cognitiveStyleUnitWeight,
	"adverb":             -cognitiveStyleUnitWeight,
	"conjunction":        -cognitiveStyleUnitWeight,
	"negation":           -cognitiveStyleUnitWeight,
}

// ComputeCognitiveStyle computes the categorical-versus-dynamic score from
// function-word category percentages.
func ComputeCognitiveStyle(fv FeatureVector) float64 {
	return ComputeCognitiveStyleCalculation(fv).FinalScore
}

func ComputeCognitiveStyleCalculation(fv FeatureVector) *ScoreCalculation {
	return computeWeightedScoreFrom(fv, cognitiveStyleCoefficients, cognitiveStyleBaseline)
}

// ComputeCognitiveStyleLabel returns a human-readable label for the score.
func ComputeCognitiveStyleLabel(score float64) string {
	switch HighModerateLow(score) {
	case "high":
		return "systematic"
	case "low":
		return "intuitive"
	}
	return "mixed"
}
