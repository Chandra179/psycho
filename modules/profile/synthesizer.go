package profile

import (
	"math"

	"github.com/google/uuid"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
)

// Profile holds the aggregated output for a single analysis.
type Profile struct {
	AnalysisID          string                          `json:"analysis_id"`
	ConfidenceFlag      string                          `json:"confidence_flag"`
	Traits              map[string]TraitResult          `json:"traits"`
	Values              map[string]float64              `json:"values"`
	ValueEvidence       map[string][]string             `json:"value_evidence,omitempty"`
	ValueExcerpts       map[string][]ingest.TextExcerpt `json:"value_excerpts,omitempty"`
	PercentileReference *ingest.PercentileReference     `json:"percentile_reference,omitempty"`
	CalculationDetails  *analyze.CalculationDetails     `json:"calculation_details,omitempty"`
	Summary             analyze.SummaryVariables        `json:"summary"`
	Narrative           string                          `json:"narrative"`
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

// Aggregate records the actual score, summary, percentile and rough-range operations.
func (sa *ScoreAggregator) Aggregate(scores analyze.BigFiveScores, fv analyze.FeatureVector, wordCount int, coverage float64) Profile {
	confidence := computeConfidenceFlag(wordCount, coverage)
	sharedWidth, rangeDetails := computeRangeWithDetails(wordCount, coverage)
	// Each measure gets its own half-width from the sampling error of its own
	// category weights. The shared length rule is the fallback for features
	// built without per-word spread.
	widthFor := func(dim string) float64 {
		if se, ok := fv.ScoreSE[dim]; ok && se > 0 {
			return perMeasureHalfWidth(se)
		}
		return sharedWidth
	}
	summary, summaryDetails := analyze.ComputeSummaryVariablesWithDetails(fv)
	details := &analyze.CalculationDetails{
		ModelFingerprint: analyze.ModelFingerprint(), WordCount: fv.WordCount, DictionaryMatches: fv.DictionaryMatches, BigWordCount: fv.BigWordCount,
		CategoryCounts: fv.CategoryCounts, Traits: scores.Calculations, Summary: summaryDetails,
		Values: make(map[string]analyze.ValueCalculation), RangeBounds: make(map[string]analyze.RangeBoundsCalculation), Range: rangeDetails, Percentiles: make(map[string]analyze.PercentileCalculation),
	}

	recordTrait := func(dim string, score float64, evidence []analyze.Contribution) TraitResult {
		// Rank on the unrounded calibrated score so scores that display the
		// same two decimals can still land on different percentiles.
		lookup := analyze.UnroundedCalibratedScore(dim, &scores)
		result, bounds := makeTraitResultWithDetails(score, widthFor(dim), sa.percentileForRecorded(dim, lookup, details), evidence)
		bounds.StandardError = fv.ScoreSE[dim]
		if bounds.StandardError > 0 {
			bounds.Method = "per_measure_sampling_error"
		} else {
			bounds.Method = "shared_length_rule"
		}
		details.RangeBounds[dim] = bounds
		return result
	}

	traits := map[string]TraitResult{
		"openness":           recordTrait("openness", scores.Openness, analyze.BigFiveEvidence("openness", fv)),
		"conscientiousness":  recordTrait("conscientiousness", scores.Conscientiousness, analyze.BigFiveEvidence("conscientiousness", fv)),
		"extraversion":       recordTrait("extraversion", scores.Extraversion, analyze.BigFiveEvidence("extraversion", fv)),
		"agreeableness":      recordTrait("agreeableness", scores.Agreeableness, analyze.BigFiveEvidence("agreeableness", fv)),
		"neuroticism":        recordTrait("neuroticism", scores.Neuroticism, analyze.BigFiveEvidence("neuroticism", fv)),
		"regulatory_focus":   recordTrait("regulatory_focus", scores.RegulatoryFocus, analyze.RegulatoryFocusEvidence(fv)),
		"need_for_cognition": recordTrait("need_for_cognition", scores.NeedForCognition, analyze.NeedForCognitionEvidence(fv)),
		"cognitive_style":    recordTrait("cognitive_style", scores.CognitiveStyle, analyze.CognitiveStyleEvidence(fv)),
		"need_for_closure":   recordTrait("need_for_closure", scores.NeedForClosure, analyze.NeedForClosureEvidence(fv)),
	}

	// Matched words per Schwartz value — the audit trail for the values
	// section (the extractor already samples words for every category).
	valueEvidence := make(map[string][]string)
	for _, cat := range analyze.SchwartzValueKeys() {
		details.Values[string(cat)] = analyze.ValueCalculation{Formula: "percent = matched_count / total_words * 100", MatchedCount: fv.CategoryCounts[cat], TotalWords: fv.WordCount, Percent: scores.Values[string(cat)]}
		if words := fv.Evidence[cat]; len(words) > 0 {
			valueEvidence[string(cat)] = words
		}
	}

	return Profile{
		AnalysisID:         uuid.New().String(),
		ConfidenceFlag:     confidence,
		Traits:             traits,
		Values:             scores.Values,
		ValueEvidence:      valueEvidence,
		ValueExcerpts:      fv.ValueExcerpts,
		Summary:            summary,
		CalculationDetails: details,
	}
}

func makeTraitResult(score, ciWidth float64, percentile int, evidence []analyze.Contribution) TraitResult {
	result, _ := makeTraitResultWithDetails(score, ciWidth, percentile, evidence)
	return result
}

func makeTraitResultWithDetails(score, ciWidth float64, percentile int, evidence []analyze.Contribution) (TraitResult, analyze.RangeBoundsCalculation) {
	low := score - ciWidth
	high := score + ciWidth
	bounds := analyze.RangeBoundsCalculation{Score: score, HalfWidth: ciWidth, UnclampedLow: low, UnclampedHigh: high}
	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}
	bounds.ClampedLow, bounds.ClampedHigh = low, high
	bounds.Low, bounds.High = math.Round(low*100)/100, math.Round(high*100)/100
	return TraitResult{
		Score:              math.Round(score*100) / 100,
		Percentile:         percentile,
		ConfidenceInterval: []float64{bounds.Low, bounds.High},
		Evidence:           evidence,
	}, bounds
}

// percentileForRecorded records the active lookup once, alongside the result.
func (sa *ScoreAggregator) percentileForRecorded(dim string, score float64, details *analyze.CalculationDetails) int {
	p, trace := sa.percentileWithDetails(dim, score)
	details.Percentiles[dim] = trace
	return p
}

func (sa *ScoreAggregator) percentileFor(dim string, score float64) int {
	p, _ := sa.percentileWithDetails(dim, score)
	return p
}

func (sa *ScoreAggregator) percentileWithDetails(dim string, score float64) (int, analyze.PercentileCalculation) {
	if sa.calibration != nil {
		if p, trace, ok := sa.calibration.PercentileWithDetails(dim, score); ok {
			return p, trace
		}
	}
	return normalPercentileWithDetails(score)
}

// The fallback assumes a mean of 0.50 and SD of 0.15; these are project
// assumptions, not a measured population distribution.
func scoreToPercentile(score float64) int { p, _ := normalPercentileWithDetails(score); return p }

func normalPercentileWithDetails(score float64) (int, analyze.PercentileCalculation) {
	mean, sd := 0.50, 0.15
	z := (score - mean) / sd
	cdf := normalCDF(z)
	raw := int(math.Round(cdf * 100))
	pct := max(1, min(99, raw))
	return pct, analyze.PercentileCalculation{Method: "normal_approximation", Formula: "z = (score - mean) / sd; cdf = 0.5 * (1 + erf(z / sqrt(2))); percentile = clamp(round(cdf * 100), 1, 99)", Score: score, NormalApproximation: &analyze.NormalPercentileCalculation{Mean: mean, SD: sd, Z: z, CDF: cdf}, UnclampedPercentile: raw, Percentile: pct}
}

// normalCDF computes the standard normal CDF using the error function.
func normalCDF(z float64) float64 {
	return 0.5 * (1 + math.Erf(z/math.Sqrt2))
}

func computeConfidenceFlag(wordCount int, coverage float64) string {
	return analyze.QualityFlag(wordCount, coverage)
}

// computeCIWidth retains the legacy API name. This is a project-defined
// rough half-width, not a validated confidence interval.
func computeCIWidth(wordCount int, coverage float64) float64 {
	w, _ := computeRangeWithDetails(wordCount, coverage)
	return w
}

// Half-width limits for per-measure bounds. The floor keeps a very long text
// from claiming near-exact scores; the cap matches the shared rule.
const (
	minMeasureHalfWidth = 0.01
	maxMeasureHalfWidth = 0.25
)

// perMeasureHalfWidth is 1.96 standard errors of the measure's own sampling
// error, limited to [0.01, 0.25] and rounded to two decimals. It is not a
// validated confidence interval: it assumes words are independent draws.
func perMeasureHalfWidth(se float64) float64 {
	w := 1.96 * se
	w = math.Max(minMeasureHalfWidth, math.Min(maxMeasureHalfWidth, w))
	return math.Round(w*100) / 100
}

func computeRangeWithDetails(wordCount int, coverage float64) (float64, analyze.RangeCalculation) {
	baseSE := 0.08
	lengthScale := math.Sqrt(float64(wordCount) / 1000)
	se := baseSE / lengthScale
	width := 1.96 * se
	multiplier := 1.0
	if coverage < 0.5 {
		multiplier = 1.5
		width *= 1.5
	} else if coverage < 0.7 {
		multiplier = 1.2
		width *= 1.2
	}
	unbounded := width
	if width > 0.25 {
		width = 0.25
	}
	if width < 0.02 {
		width = 0.02
	}
	halfWidth := math.Round(width*100) / 100
	return halfWidth, analyze.RangeCalculation{Formula: "length_scale = sqrt(word_count / 1000); standard_error = 0.08 / length_scale; unbounded_half_width = 1.96 * standard_error * coverage_multiplier (1.5 if coverage < 0.5, 1.2 if < 0.7, otherwise 1); half_width = round2(clamp(unbounded_half_width, 0.02, 0.25)); bounds = round2(clamp(score +/- half_width, 0, 1))", WordCount: wordCount, Coverage: coverage, BaseStandardError: baseSE, LengthScale: lengthScale, StandardError: se, ErrorMultiplier: 1.96, CoverageMultiplier: multiplier, UnboundedHalfWidth: unbounded, MinimumHalfWidth: 0.02, MaximumHalfWidth: 0.25, BoundedHalfWidth: width, HalfWidth: halfWidth}
}
