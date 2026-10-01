package profile

import (
	"math"

	"github.com/google/uuid"

	"psycho/modules/analyze"
)

// Profile holds the aggregated output for a single analysis.
type Profile struct {
	AnalysisID     string                   `json:"analysis_id"`
	ConfidenceFlag string                   `json:"confidence_flag"`
	Traits         map[string]TraitResult   `json:"traits"`
	Values         map[string]float64       `json:"values"`
	ValueEvidence  map[string][]string      `json:"value_evidence,omitempty"`
	Summary        analyze.SummaryVariables `json:"summary"`
	Narrative      string                   `json:"narrative"`
}

// TraitResult holds one Big Five trait output. Evidence lists the category
// contributions behind the score, strongest first — the audit trail.
type TraitResult struct {
	Score              float64                `json:"score"`
	Percentile         int                    `json:"percentile"`
	ConfidenceInterval []float64              `json:"confidence_interval"`
	Evidence           []analyze.Contribution `json:"evidence,omitempty"`
}

// ScoreAggregator merges raw scores into a user-facing profile. An optional
// calibration replaces the normal-approximation percentile with a lookup
// against an empirically measured reference distribution.
type ScoreAggregator struct {
	calibration *analyze.Calibration
}

func NewScoreAggregator() *ScoreAggregator {
	return &ScoreAggregator{}
}

// UseCalibration attaches an empirical reference distribution used for
// percentile lookup. Call before Aggregate; nil restores the default.
func (sa *ScoreAggregator) UseCalibration(cal *analyze.Calibration) {
	sa.calibration = cal
}

// Aggregate converts raw BigFiveScores into a Profile with confidence intervals.
func (sa *ScoreAggregator) Aggregate(scores analyze.BigFiveScores, fv analyze.FeatureVector, wordCount int, coverage float64) Profile {
	confidence := computeConfidenceFlag(wordCount, coverage)
	ciWidth := computeCIWidth(wordCount, coverage)

	traits := map[string]TraitResult{
		"openness":           makeTraitResult(scores.Openness, ciWidth, sa.percentileFor("openness", scores.Openness), analyze.BigFiveEvidence("openness", fv)),
		"conscientiousness":  makeTraitResult(scores.Conscientiousness, ciWidth, sa.percentileFor("conscientiousness", scores.Conscientiousness), analyze.BigFiveEvidence("conscientiousness", fv)),
		"extraversion":       makeTraitResult(scores.Extraversion, ciWidth, sa.percentileFor("extraversion", scores.Extraversion), analyze.BigFiveEvidence("extraversion", fv)),
		"agreeableness":      makeTraitResult(scores.Agreeableness, ciWidth, sa.percentileFor("agreeableness", scores.Agreeableness), analyze.BigFiveEvidence("agreeableness", fv)),
		"neuroticism":        makeTraitResult(scores.Neuroticism, ciWidth, sa.percentileFor("neuroticism", scores.Neuroticism), analyze.BigFiveEvidence("neuroticism", fv)),
		"regulatory_focus":   makeTraitResult(scores.RegulatoryFocus, ciWidth, sa.percentileFor("regulatory_focus", scores.RegulatoryFocus), analyze.RegulatoryFocusEvidence(fv)),
		"need_for_cognition": makeTraitResult(scores.NeedForCognition, ciWidth, sa.percentileFor("need_for_cognition", scores.NeedForCognition), analyze.NeedForCognitionEvidence(fv)),
		"cognitive_style":    makeTraitResult(scores.CognitiveStyle, ciWidth, sa.percentileFor("cognitive_style", scores.CognitiveStyle), analyze.CognitiveStyleEvidence(fv)),
		"need_for_closure":   makeTraitResult(scores.NeedForClosure, ciWidth, sa.percentileFor("need_for_closure", scores.NeedForClosure), analyze.NeedForClosureEvidence(fv)),
	}

	// Matched words per Schwartz value — the audit trail for the values
	// section (the extractor already samples words for every category).
	valueEvidence := make(map[string][]string)
	for _, cat := range analyze.SchwartzValueKeys() {
		if words := fv.Evidence[cat]; len(words) > 0 {
			valueEvidence[string(cat)] = words
		}
	}

	return Profile{
		AnalysisID:     uuid.New().String(),
		ConfidenceFlag: confidence,
		Traits:         traits,
		Values:         scores.Values,
		ValueEvidence:  valueEvidence,
		Summary:        analyze.ComputeSummaryVariables(fv),
	}
}

func makeTraitResult(score, ciWidth float64, percentile int, evidence []analyze.Contribution) TraitResult {
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
		Percentile:         percentile,
		ConfidenceInterval: []float64{math.Round(low*100) / 100, math.Round(high*100) / 100},
		Evidence:           evidence,
	}
}

// percentileFor resolves a score's population percentile: from the
// calibration reference distribution when one is attached, otherwise from
// the normal approximation below.
func (sa *ScoreAggregator) percentileFor(dim string, score float64) int {
	if sa.calibration != nil {
		if p, ok := sa.calibration.Percentile(dim, score); ok {
			return p
		}
	}
	return scoreToPercentile(score)
}

// scoreToPercentile converts a [0,1] trait score to a population percentile
// assuming a normal distribution with mean 0.50 and SD 0.15 — the same
// trait-scale SD the regression coefficients were derived under
// (see modules/analyze/coefficients.go).
func scoreToPercentile(score float64) int {
	mean := 0.50
	sd := 0.15
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
