package analyze

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// DimensionCalibration holds the empirical reference distribution for one
// trait dimension, measured over a calibration corpus. Offset is added to
// the raw heuristic score so the corpus average lands at 0.50; Quantiles
// holds the 1st–99th percentile scores of the offset-adjusted corpus, which
// is what Percentile lookups interpolate over. Mean/SD/N are informational.
type DimensionCalibration struct {
	Offset    float64   `json:"offset"`
	Mean      float64   `json:"mean"`
	SD        float64   `json:"sd"`
	Quantiles []float64 `json:"quantiles"`
	N         int       `json:"n"`
}

// Calibration is the empirical reference that turns raw heuristic scores
// into interpretable absolute numbers: corpus-mean-centered scores and
// distribution-free percentiles. A nil *Calibration means "uncalibrated" —
// callers fall back to the fixed 0.50 intercepts and the normal
// approximation in profile.scoreToPercentile.
//
// DictionarySHA256 pins the calibration to the dictionary it was built
// from; a test fails if the dictionary changes without recalibration.
type Calibration struct {
	Corpus           string                          `json:"corpus"`
	GeneratedAt      string                          `json:"generated_at"`
	ModelFingerprint string                          `json:"model_fingerprint"`
	DictionarySHA256 string                          `json:"dictionary_sha256,omitempty"`
	Dimensions       map[string]DimensionCalibration `json:"dimensions"`
}

// SampleSize returns the number of reference texts used by this calibration.
// Calibration dimensions are built from the same corpus sample.
func (c *Calibration) SampleSize() int {
	if c == nil {
		return 0
	}
	for _, dim := range dimensionKeys {
		if d, ok := c.Dimensions[dim]; ok {
			return d.N
		}
	}
	return 0
}

// LoadCalibration parses calibration JSON produced by cmd/calibrate.
func LoadCalibration(data []byte) (*Calibration, error) {
	var c Calibration
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse calibration: %w", err)
	}
	if c.ModelFingerprint == "" {
		return nil, fmt.Errorf("calibration has no model_fingerprint; rerun cmd/calibrate")
	}
	if len(c.Dimensions) != len(dimensionKeys) {
		return nil, fmt.Errorf("calibration must cover all %d dimensions", len(dimensionKeys))
	}
	sampleSize := 0
	for _, name := range dimensionKeys {
		d, ok := c.Dimensions[name]
		if !ok {
			return nil, fmt.Errorf("calibration is missing dimension %q", name)
		}
		if d.N < 2 || (sampleSize != 0 && d.N != sampleSize) {
			return nil, fmt.Errorf("calibration dimension %q has an invalid or inconsistent sample size", name)
		}
		sampleSize = d.N
		if len(d.Quantiles) != 99 {
			return nil, fmt.Errorf("calibration dimension %q must have 99 quantiles", name)
		}
		if !finite(d.Offset) || !finite(d.Mean) || d.Mean < 0 || d.Mean > 1 || !finite(d.SD) || d.SD < 0 {
			return nil, fmt.Errorf("calibration dimension %q has invalid statistics", name)
		}
		for i, q := range d.Quantiles {
			if !finite(q) || q < 0 || q > 1 {
				return nil, fmt.Errorf("calibration dimension %q has invalid quantile %d", name, i+1)
			}
			if i > 0 && q < d.Quantiles[i-1] {
				return nil, fmt.Errorf("calibration dimension %q quantiles not sorted", name)
			}
		}
	}

	return &c, nil
}

// BuildCalibration derives reference statistics from corpus score samples
// (raw scores as produced by Infer + the Compute* functions): per-dimension
// offset centering the corpus mean at 0.50, and the 1st–99th percentile
// quantiles of the offset-adjusted scores.
func BuildCalibration(corpus, generatedAt string, samples []BigFiveScores) (*Calibration, error) {
	if len(samples) < 2 {
		return nil, fmt.Errorf("need at least 2 score samples, got %d", len(samples))
	}

	raw := make(map[string][]float64, len(dimensionKeys))
	for _, s := range samples {
		for _, dim := range dimensionKeys {
			raw[dim] = append(raw[dim], preRoundValue(dim, &s))
		}
	}

	dimensions := make(map[string]DimensionCalibration, len(dimensionKeys))
	for _, dim := range dimensionKeys {
		vals := raw[dim]
		mean := 0.0
		for _, v := range vals {
			mean += v
		}
		mean /= float64(len(vals))

		offset := round4(0.50 - mean)

		// Quantiles are taken over the unrounded adjusted scores, the same
		// values AdjustScores keeps as CalibratedClampedScore, so percentile
		// lookups can tell apart scores that round to the same two decimals.
		adjusted := make([]float64, len(vals))
		for i, v := range vals {
			adjusted[i] = math.Max(0, math.Min(1, v+offset))
		}
		sort.Float64s(adjusted)

		adjMean := 0.0
		for _, v := range adjusted {
			adjMean += v
		}
		adjMean /= float64(len(adjusted))
		variance := 0.0
		for _, v := range adjusted {
			variance += (v - adjMean) * (v - adjMean)
		}
		sd := math.Sqrt(variance / float64(len(adjusted)))

		quantiles := make([]float64, 99)
		for p := 1; p <= 99; p++ {
			quantiles[p-1] = round4(quantile(adjusted, float64(p)))
		}

		dimensions[dim] = DimensionCalibration{
			Offset:    offset,
			Mean:      round4(adjMean),
			SD:        round4(sd),
			Quantiles: quantiles,
			N:         len(samples),
		}
	}

	return &Calibration{
		Corpus:           corpus,
		ModelFingerprint: ModelFingerprint(),
		GeneratedAt:      generatedAt,
		Dimensions:       dimensions,
	}, nil
}

// AdjustScores recenters raw heuristic scores so the calibration corpus
// averages 0.50 per dimension. Input scores must come from the production
// path (clamped and rounded by Infer / the Compute* functions); the output
// is clamped and rounded the same way.
func (c *Calibration) AdjustScores(scores *BigFiveScores) {
	for _, dim := range dimensionKeys {
		d, ok := c.Dimensions[dim]
		if !ok {
			continue
		}
		unrounded := preRoundValue(dim, scores) + d.Offset
		final := clamp(unrounded)
		setDimensionValue(dim, scores, final)
		if trace := scores.Calculations[dim]; trace != nil {
			trace.CalibrationApplied = true
			trace.CalibrationOffset = d.Offset
			trace.CalibratedUnroundedScore = unrounded
			trace.CalibratedClampedScore = math.Max(0, math.Min(1, unrounded))
			trace.FinalScore = final
		}
	}
}

// Percentile maps an offset-adjusted score to its population percentile,
// clamped to [1, 99]. ok is false when the dimension is not covered by this
// calibration; callers fall back to their default.
func (c *Calibration) Percentile(dim string, score float64) (int, bool) {
	p, _, ok := c.PercentileWithDetails(dim, score)
	return p, ok
}

// PercentileWithDetails records the branch and operands of the actual lookup.
func (c *Calibration) PercentileWithDetails(dim string, score float64) (int, PercentileCalculation, bool) {
	d, ok := c.Dimensions[dim]
	lookup := &EmpiricalPercentileCalculation{SampleSize: d.N}
	trace := PercentileCalculation{Method: "empirical", Score: score, EmpiricalLookup: lookup}
	if !ok || len(d.Quantiles) == 0 {
		return 0, trace, false
	}
	q := d.Quantiles
	finish := func(p int) (int, PercentileCalculation, bool) {
		trace.UnclampedPercentile = p
		trace.Percentile = clampPercentile(p)
		return trace.Percentile, trace, true
	}
	nLess, nEq := 0, 0
	for _, v := range q {
		if v < score {
			nLess++
		} else if v == score {
			nEq++
		}
	}
	lookup.QuantilesBelow = nLess
	lookup.EqualQuantiles = nEq
	if score < q[0] {
		trace.Formula = "score < first quantile: percentile = 1"
		lookup.LowerScore = q[0]
		lookup.LowerPercentile = 1
		lookup.UpperScore = q[0]
		lookup.UpperPercentile = 1
		return finish(1)
	}
	if score > q[len(q)-1] {
		trace.Formula = "score > last quantile: percentile = 99"
		lookup.UpperScore = q[len(q)-1]
		lookup.UpperPercentile = 99
		lookup.LowerScore = q[len(q)-1]
		lookup.LowerPercentile = 99
		return finish(99)
	}
	if nEq > 0 {
		trace.Formula = "percentile = clamp(quantiles_below + integer_floor((equal_quantiles + 1) / 2), 1, 99)"
		lookup.LowerPercentile = nLess + 1
		lookup.UpperPercentile = nLess + nEq
		lookup.LowerScore = score
		lookup.UpperScore = score
		return finish(nLess + (nEq+1)/2)
	}
	prev, next := q[nLess-1], q[nLess]
	frac := (score - prev) / (next - prev)
	trace.Formula = "fraction = (score - lower_score) / (upper_score - lower_score); percentile = clamp(lower_percentile + round(fraction), 1, 99)"
	lookup.LowerScore = prev
	lookup.UpperScore = next
	lookup.LowerPercentile = nLess
	lookup.UpperPercentile = nLess + 1
	lookup.Fraction = frac
	return finish(nLess + int(math.Round(frac)))
}

// preRoundValue is the dimension's model score before the two-decimal
// rounding (clamped to [0, 1]), falling back to the rounded score when no
// calculation trace is attached.
func preRoundValue(dim string, s *BigFiveScores) float64 {
	if t := s.Calculations[dim]; t != nil {
		return t.ClampedScore
	}
	return dimensionValue(dim, s)
}

// UnroundedCalibratedScore returns the calibrated, unrounded score used for
// percentile lookups, or the displayed score when no calibration was applied.
func UnroundedCalibratedScore(dim string, scores *BigFiveScores) float64 {
	if t := scores.Calculations[dim]; t != nil && t.CalibrationApplied {
		return t.CalibratedClampedScore
	}
	return dimensionValue(dim, scores)
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func clampPercentile(p int) int {
	if p < 1 {
		return 1
	}
	if p > 99 {
		return 99
	}
	return p
}

// CalibratedDimensions returns the dimension keys a calibration covers, in
// reporting order.
func CalibratedDimensions() []string {
	return append([]string(nil), dimensionKeys...)
}

// dimensionKeys are the calibrated score dimensions, keyed the same way the
// profile aggregator keys its trait map.
var dimensionKeys = []string{
	"openness",
	"conscientiousness",
	"extraversion",
	"agreeableness",
	"neuroticism",
	"regulatory_focus",
	"need_for_cognition",
	"cognitive_style",
	"need_for_closure",
}

func dimensionValue(dim string, s *BigFiveScores) float64 {
	switch dim {
	case "openness":
		return s.Openness
	case "conscientiousness":
		return s.Conscientiousness
	case "extraversion":
		return s.Extraversion
	case "agreeableness":
		return s.Agreeableness
	case "neuroticism":
		return s.Neuroticism
	case "regulatory_focus":
		return s.RegulatoryFocus
	case "need_for_cognition":
		return s.NeedForCognition
	case "cognitive_style":
		return s.CognitiveStyle
	case "need_for_closure":
		return s.NeedForClosure
	}
	return 0
}

func setDimensionValue(dim string, s *BigFiveScores, v float64) {
	switch dim {
	case "openness":
		s.Openness = v
	case "conscientiousness":
		s.Conscientiousness = v
	case "extraversion":
		s.Extraversion = v
	case "agreeableness":
		s.Agreeableness = v
	case "neuroticism":
		s.Neuroticism = v
	case "regulatory_focus":
		s.RegulatoryFocus = v
	case "need_for_cognition":
		s.NeedForCognition = v
	case "cognitive_style":
		s.CognitiveStyle = v
	case "need_for_closure":
		s.NeedForClosure = v
	}
}

// quantile returns the p-th percentile (p in [0, 100]) of a sorted slice by
// linear interpolation between adjacent order statistics.
func quantile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := (p / 100) * float64(len(sorted)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	frac := idx - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}
