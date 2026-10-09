package analyze

import (
	"maps"
	"math"
	"slices"
)

// CalculationDetails records the operations used for this analysis. It is
// persisted with the profile so future renderers never need to rerun a model.
type CalculationDetails struct {
	ModelFingerprint  string `json:"model_fingerprint"`
	WordCount         int    `json:"word_count"`
	DictionaryMatches int    `json:"dictionary_matches"`
	// BigWordCount is the number of words longer than six letters. It is
	// recorded for the formal-prose note only; no score uses it.
	BigWordCount   int                               `json:"big_word_count,omitempty"`
	CategoryCounts map[Category]int                  `json:"category_counts"`
	Traits         map[string]*ScoreCalculation      `json:"traits"`
	Summary        map[string]SummaryCalculation     `json:"summary"`
	Values         map[string]ValueCalculation       `json:"values"`
	RangeBounds    map[string]RangeBoundsCalculation `json:"range_bounds"`
	Range          RangeCalculation                  `json:"range"`
	Percentiles    map[string]PercentileCalculation  `json:"percentiles"`
}

// ScoreTerm retains full floating-point precision; sampled words are separate
// evidence and do not determine the category counts.
type ScoreTerm struct {
	Category     string  `json:"category"`
	MatchedCount int     `json:"matched_count"`
	TotalWords   int     `json:"total_words"`
	WordPercent  float64 `json:"word_percent"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
}

type ScoreCalculation struct {
	Formula                  string      `json:"formula"`
	Baseline                 float64     `json:"baseline"`
	Terms                    []ScoreTerm `json:"terms"`
	ContributionTotal        float64     `json:"contribution_total"`
	UnroundedScore           float64     `json:"unrounded_score"`
	ClampedScore             float64     `json:"clamped_score"`
	ModelScore               float64     `json:"model_score"`
	CalibrationApplied       bool        `json:"calibration_applied"`
	CalibrationOffset        float64     `json:"calibration_offset"`
	CalibratedUnroundedScore float64     `json:"calibrated_unrounded_score"`
	CalibratedClampedScore   float64     `json:"calibrated_clamped_score"`
	FinalScore               float64     `json:"final_score"`
}

type SummaryCalculation struct {
	Formula      string             `json:"formula"`
	Inputs       map[string]float64 `json:"inputs"`
	Numerator    float64            `json:"numerator"`
	Divisor      float64            `json:"divisor"`
	ScaledInput  float64            `json:"scaled_input"`
	SigmoidValue float64            `json:"sigmoid_value"`
	Score        float64            `json:"score"`
}

type ValueCalculation struct {
	Formula      string  `json:"formula"`
	MatchedCount int     `json:"matched_count"`
	TotalWords   int     `json:"total_words"`
	Percent      float64 `json:"percent"`
}

type RangeCalculation struct {
	Formula            string  `json:"formula"`
	WordCount          int     `json:"word_count"`
	Coverage           float64 `json:"coverage"`
	BaseStandardError  float64 `json:"base_standard_error"`
	LengthScale        float64 `json:"length_scale"`
	StandardError      float64 `json:"standard_error"`
	ErrorMultiplier    float64 `json:"error_multiplier"`
	CoverageMultiplier float64 `json:"coverage_multiplier"`
	UnboundedHalfWidth float64 `json:"unbounded_half_width"`
	MinimumHalfWidth   float64 `json:"minimum_half_width"`
	MaximumHalfWidth   float64 `json:"maximum_half_width"`
	BoundedHalfWidth   float64 `json:"bounded_half_width"`
	HalfWidth          float64 `json:"half_width"`
}

type RangeBoundsCalculation struct {
	// Method is "per_measure_sampling_error" (half-width = 1.96 * StandardError,
	// limited to 0.01-0.25) or "shared_length_rule" for older records.
	Method        string  `json:"method,omitempty"`
	StandardError float64 `json:"standard_error,omitempty"`
	Score         float64 `json:"score"`
	HalfWidth     float64 `json:"half_width"`
	UnclampedLow  float64 `json:"unclamped_low"`
	UnclampedHigh float64 `json:"unclamped_high"`
	ClampedLow    float64 `json:"clamped_low"`
	ClampedHigh   float64 `json:"clamped_high"`
	Low           float64 `json:"low"`
	High          float64 `json:"high"`
}

// PercentileCalculation separates method-specific operands so unused values
// are absent rather than presenting artificial zero measurements.
type PercentileCalculation struct {
	Method              string                          `json:"method"`
	Formula             string                          `json:"formula"`
	Score               float64                         `json:"score"`
	NormalApproximation *NormalPercentileCalculation    `json:"normal_approximation,omitempty"`
	EmpiricalLookup     *EmpiricalPercentileCalculation `json:"empirical_lookup,omitempty"`
	UnclampedPercentile int                             `json:"unclamped_percentile"`
	Percentile          int                             `json:"percentile"`
}

type NormalPercentileCalculation struct {
	Mean float64 `json:"mean"`
	SD   float64 `json:"sd"`
	Z    float64 `json:"z"`
	CDF  float64 `json:"cdf"`
}

type EmpiricalPercentileCalculation struct {
	SampleSize      int     `json:"sample_size"`
	LowerScore      float64 `json:"lower_score"`
	UpperScore      float64 `json:"upper_score"`
	LowerPercentile int     `json:"lower_percentile"`
	UpperPercentile int     `json:"upper_percentile"`
	QuantilesBelow  int     `json:"quantiles_below"`
	EqualQuantiles  int     `json:"equal_quantiles"`
	Fraction        float64 `json:"fraction"`
}

const additionalScoreBaseline = 0.50

const weightedScoreFormula = "category_percent = matched_count / total_words * 100; contribution = weight * category_percent; the accumulator starts at baseline and adds terms in recorded (sorted category) order; contribution_total accumulates separately from zero; unrounded_score is the actual accumulator; model_score = round2(clamp(baseline + sum(contributions), 0, 1)); final_score = round2(clamp(model_score + calibration_offset, 0, 1)) when calibration is applied. round2 rounds half away from zero to 2 decimals."

func newScoreCalculation(baseline float64) *ScoreCalculation {
	return &ScoreCalculation{Formula: weightedScoreFormula, Baseline: baseline, Terms: []ScoreTerm{}}
}

func (c *ScoreCalculation) addTerm(fv FeatureVector, category string, percent, weight, contribution float64) {
	if weight == 0 {
		return
	}
	count := fv.CategoryCounts[Category(category)]
	c.Terms = append(c.Terms, ScoreTerm{category, count, fv.WordCount, percent, weight, contribution})
	c.ContributionTotal += contribution
}

// finish records the actual accumulator rather than recomputing it from
// display-rounded evidence. The original clamp/round operation is reused.
func (c *ScoreCalculation) finish(accumulator float64) float64 {
	c.UnroundedScore = accumulator
	c.ClampedScore = math.Max(0, math.Min(1, accumulator))
	c.ModelScore = clamp(accumulator)
	c.CalibratedUnroundedScore = c.ModelScore
	c.CalibratedClampedScore = c.ModelScore
	c.FinalScore = c.ModelScore
	return c.FinalScore
}

func computeWeightedScore(fv FeatureVector, weights map[string]float64) *ScoreCalculation {
	return computeWeightedScoreFrom(fv, weights, additionalScoreBaseline)
}

func computeWeightedScoreFrom(fv FeatureVector, weights map[string]float64, baseline float64) *ScoreCalculation {
	c := newScoreCalculation(baseline)
	score := c.Baseline
	for _, category := range slices.Sorted(maps.Keys(weights)) {
		weight := weights[category]
		percent := fv.CategoryPercents[Category(category)]
		contribution := weight * percent
		score += contribution
		c.addTerm(fv, category, percent, weight, contribution)
	}
	c.finish(score)
	return c
}
