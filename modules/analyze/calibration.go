package analyze

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// DimensionCalibration holds the empirical reference distribution for one
// trait dimension, measured over a calibration corpus. Offset is added to
// the raw regression score so the corpus average lands at 0.50; Quantiles
// holds the 1st–99th percentile scores of the offset-adjusted corpus, which
// is what Percentile lookups interpolate over. Mean/SD/N are informational.
type DimensionCalibration struct {
	Offset    float64   `json:"offset"`
	Mean      float64   `json:"mean"`
	SD        float64   `json:"sd"`
	Quantiles []float64 `json:"quantiles"`
	N         int       `json:"n"`
}

// Calibration is the empirical reference that turns raw regression scores
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
	DictionarySHA256 string                          `json:"dictionary_sha256,omitempty"`
	Dimensions       map[string]DimensionCalibration `json:"dimensions"`
}

// LoadCalibration parses calibration JSON produced by cmd/calibrate.
func LoadCalibration(data []byte) (*Calibration, error) {
	var c Calibration
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse calibration: %w", err)
	}
	if len(c.Dimensions) == 0 {
		return nil, fmt.Errorf("calibration has no dimensions")
	}
	for name, d := range c.Dimensions {
		if len(d.Quantiles) == 0 {
			return nil, fmt.Errorf("calibration dimension %q has no quantiles", name)
		}
		for i := 1; i < len(d.Quantiles); i++ {
			if d.Quantiles[i] < d.Quantiles[i-1] {
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
			raw[dim] = append(raw[dim], dimensionValue(dim, &s))
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

		// Adjusted values must match what Calibration.AdjustScores produces
		// in production: clamp rounds to 2 decimals on the way through.
		adjusted := make([]float64, len(vals))
		for i, v := range vals {
			adjusted[i] = clamp(v + offset)
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
		Corpus:      corpus,
		GeneratedAt: generatedAt,
		Dimensions:  dimensions,
	}, nil
}

// AdjustScores recenters raw regression scores so the calibration corpus
// averages 0.50 per dimension. Input scores must come from the production
// path (clamped and rounded by Infer / the Compute* functions); the output
// is clamped and rounded the same way.
func (c *Calibration) AdjustScores(scores *BigFiveScores) {
	for _, dim := range dimensionKeys {
		d, ok := c.Dimensions[dim]
		if !ok {
			continue
		}
		setDimensionValue(dim, scores, clamp(dimensionValue(dim, scores)+d.Offset))
	}
}

// Percentile maps an offset-adjusted score to its population percentile,
// clamped to [1, 99]. ok is false when the dimension is not covered by this
// calibration; callers fall back to their default.
func (c *Calibration) Percentile(dim string, score float64) (int, bool) {
	d, ok := c.Dimensions[dim]
	if !ok || len(d.Quantiles) == 0 {
		return 0, false
	}
	q := d.Quantiles
	if score < q[0] {
		return 1, true
	}
	if score > q[len(q)-1] {
		return 99, true
	}

	// Quantiles[i] is the (i+1)-th percentile. Count where the score falls.
	nLess, nEq := 0, 0
	for _, v := range q {
		switch {
		case v < score:
			nLess++
		case v == score:
			nEq++
		}
	}
	if nEq > 0 {
		// Score coincides with a quantile (or a run of them — near-constant
		// dimensions produce ties): take the middle of the occupied range.
		return clampPercentile(nLess + (nEq+1)/2), true
	}
	// Strictly between q[nLess-1] (percentile nLess) and q[nLess] (the next).
	prev, next := q[nLess-1], q[nLess]
	frac := (score - prev) / (next - prev)
	return clampPercentile(nLess + int(math.Round(frac))), true
}

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
