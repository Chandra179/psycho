package profile

import (
	"math"

	"github.com/google/uuid"

	"psycho/modules/analyze"
)

// Profile holds the aggregated output for a single analysis.
type Profile struct {
	AnalysisID     string
	ConfidenceFlag string
	Traits         map[string]TraitResult
	Values         map[string]float64
	Summary        analyze.SummaryVariables
}

// TraitResult holds one Big Five trait output.
type TraitResult struct {
	Score              float64   `json:"score"`
	Percentile         int       `json:"percentile"`
	ConfidenceInterval []float64 `json:"confidence_interval"`
}

// ScoreAggregator merges raw scores into a user-facing profile.
type ScoreAggregator struct{}

func NewScoreAggregator() *ScoreAggregator {
	return &ScoreAggregator{}
}

// Aggregate converts raw BigFiveScores into a Profile with confidence intervals.
func (sa *ScoreAggregator) Aggregate(scores analyze.BigFiveScores, fv analyze.FeatureVector, wordCount int, coverage float64) Profile {
	confidence := computeConfidenceFlag(wordCount, coverage)
	ciWidth := computeCIWidth(wordCount, coverage)

	traits := map[string]TraitResult{
		"openness":           makeTraitResult(scores.Openness, ciWidth),
		"conscientiousness":  makeTraitResult(scores.Conscientiousness, ciWidth),
		"extraversion":       makeTraitResult(scores.Extraversion, ciWidth),
		"agreeableness":      makeTraitResult(scores.Agreeableness, ciWidth),
		"neuroticism":        makeTraitResult(scores.Neuroticism, ciWidth),
		"regulatory_focus":   makeTraitResult(scores.RegulatoryFocus, ciWidth),
		"need_for_cognition": makeTraitResult(scores.NeedForCognition, ciWidth),
		"cognitive_style":    makeTraitResult(scores.CognitiveStyle, ciWidth),
		"need_for_closure":   makeTraitResult(scores.NeedForClosure, ciWidth),
	}

	return Profile{
		AnalysisID:     uuid.New().String(),
		ConfidenceFlag: confidence,
		Traits:         traits,
		Values:         scores.Values,
		Summary:        analyze.ComputeSummaryVariables(fv),
	}
}

func makeTraitResult(score, ciWidth float64) TraitResult {
	low := score - ciWidth
	high := score + ciWidth
	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}
	return TraitResult{
		Score:              math.Round(score*100) / 100,
		Percentile:         scoreToPercentile(score),
		ConfidenceInterval: []float64{math.Round(low*100) / 100, math.Round(high*100) / 100},
	}
}

// scoreToPercentile converts a [0,1] trait score to a population percentile
// assuming a normal distribution with mean 0.50 and SD 0.12.
func scoreToPercentile(score float64) int {
	mean := 0.50
	sd := 0.12
	z := (score - mean) / sd
	p := normalCDF(z)
	pct := int(math.Round(p * 100))
	if pct < 1 {
		return 1
	}
	if pct > 99 {
		return 99
	}
	return pct
}

// normalCDF computes the standard normal CDF using the error function.
func normalCDF(z float64) float64 {
	return 0.5 * (1 + math.Erf(z/math.Sqrt2))
}

func computeConfidenceFlag(wordCount int, coverage float64) string {
	if wordCount < 500 {
		return "low"
	}
	if coverage < 0.6 {
		return "medium"
	}
	if wordCount < 1000 {
		return "medium"
	}
	return "high"
}

func computeCIWidth(wordCount int, coverage float64) float64 {
	// 95% CI width based on standard error of measurement.
	// SE = baseSE / sqrt(n/1000) where baseSE ≈ 0.08 for a 1000-word text
	// at typical LIWC-based prediction accuracy. CI = 1.96 × SE.
	baseSE := 0.08
	se := baseSE / math.Sqrt(float64(wordCount)/1000)
	width := 1.96 * se
	if coverage < 0.5 {
		width *= 1.5
	} else if coverage < 0.7 {
		width *= 1.2
	}
	if width > 0.25 {
		width = 0.25
	}
	if width < 0.02 {
		width = 0.02
	}
	return math.Round(width*100) / 100
}
