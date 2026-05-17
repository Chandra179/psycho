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
func (sa *ScoreAggregator) Aggregate(scores analyze.BigFiveScores, wordCount int, coverage float64) Profile {
	confidence := computeConfidenceFlag(wordCount, coverage)
	ciWidth := computeCIWidth(wordCount, coverage)

	traits := map[string]TraitResult{
		"openness":          makeTraitResult(scores.Openness, ciWidth),
		"conscientiousness": makeTraitResult(scores.Conscientiousness, ciWidth),
		"extraversion":      makeTraitResult(scores.Extraversion, ciWidth),
		"agreeableness":     makeTraitResult(scores.Agreeableness, ciWidth),
		"neuroticism":       makeTraitResult(scores.Neuroticism, ciWidth),
	}

	return Profile{
		AnalysisID:     uuid.New().String(),
		ConfidenceFlag: confidence,
		Traits:         traits,
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
		Percentile:         int(math.Round(score * 100)),
		ConfidenceInterval: []float64{math.Round(low*100) / 100, math.Round(high*100) / 100},
	}
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
	// Base width decreases with word count and increases with poor coverage.
	base := 0.15
	if wordCount > 0 {
		base = base / math.Sqrt(float64(wordCount)/100)
	}
	if coverage < 0.5 {
		base *= 1.5
	} else if coverage < 0.7 {
		base *= 1.2
	}
	if base > 0.25 {
		base = 0.25
	}
	return math.Round(base*100) / 100
}
