package analyze

import (
	"math"
	"sync"
)

// Every calibrated dimension is baseline + sum(weight * category percent), so
// the score is the mean over words of a per-word contribution:
//
//	x_i = 100 * (sum of weights of the categories word i belongs to)
//	score = baseline + mean(x_i)
//
// Treating the words as independent draws, the standard error of the score is
// sd(x_i) / sqrt(n). It differs per dimension because each one weights
// different categories, which is why one shared width for all nine measures was
// wrong. Real text is bursty (topics repeat words), so the independent-word
// figure is too small; seInflation scales it to what was measured.

// seInflation is the measured ratio of the real half-to-half score difference
// to the independent-word standard error, per dimension. Measured on 2026-10-09
// by scoring the first and second halves of the 375 reference posts with at
// least 800 words: the empirical standard error (sd of the difference / sqrt 2)
// divided by the predicted one gave 1.50 openness, 1.46 conscientiousness, 1.47
// extraversion, 1.23 agreeableness, 1.74 neuroticism, 1.15 regulatory focus, 1.05
// need for cognition and 1.29 need for closure; cognitive style was re-measured
// at 1.68 after it moved to the function-word index. Re-measured on 2026-10-10
// (389 posts) after the non-significant pronoun weights were dropped: 1.52
// openness, 1.49 conscientiousness, 1.51 extraversion, 1.25 agreeableness, 1.53
// neuroticism (was 1.74, so the value was lowered), 1.13 regulatory focus, 1.13
// need for cognition, 1.69 cognitive style, 1.26 need for closure; the others
// moved by under 10% and were kept. The values
// below round up. They describe repeatability within one text, not accuracy
// against a person's true trait.
var seInflation = map[string]float64{
	"openness":           1.5,
	"conscientiousness":  1.5,
	"extraversion":       1.5,
	"agreeableness":      1.25,
	"neuroticism":        1.55,
	"regulatory_focus":   1.15,
	"need_for_cognition": 1.05,
	"cognitive_style":    1.7,
	"need_for_closure":   1.3,
}

var (
	dimWeightsOnce sync.Once
	dimWeights     map[string]map[Category]float64
)

// dimensionWeights returns per-category weights (per percentage point) for each
// calibrated dimension, taken from the same tables the scorers use.
func dimensionWeights() map[string]map[Category]float64 {
	dimWeightsOnce.Do(func() {
		dimWeights = make(map[string]map[Category]float64, len(dimensionKeys))
		for _, dim := range dimensionKeys[:5] {
			m := make(map[Category]float64)
			for cat, w := range coefficients {
				if v := w.weightFor(dim); v != 0 {
					m[Category(cat)] = v
				}
			}
			dimWeights[dim] = m
		}
		add := func(dim string, src map[string]float64) {
			m := make(map[Category]float64, len(src))
			for cat, w := range src {
				m[Category(cat)] = w
			}
			dimWeights[dim] = m
		}
		add("regulatory_focus", regFocusCoefficients)
		add("need_for_cognition", needCogCoefficients)
		add("cognitive_style", cognitiveStyleCoefficients)
		add("need_for_closure", needClosureCoefficients)
	})
	return dimWeights
}

// seAccumulator gathers sum and sum of squares of x_i per dimension.
type seAccumulator struct {
	n     int
	sum   map[string]float64
	sumSq map[string]float64
}

func newSEAccumulator() *seAccumulator {
	return &seAccumulator{sum: make(map[string]float64, len(dimensionKeys)), sumSq: make(map[string]float64, len(dimensionKeys))}
}

// add records one word: the categories it counts toward.
func (a *seAccumulator) add(cats []Category) {
	a.n++
	weights := dimensionWeights()
	for _, dim := range dimensionKeys {
		w := weights[dim]
		x := 0.0
		for _, c := range cats {
			x += w[c]
		}
		if x == 0 {
			continue
		}
		x *= 100
		a.sum[dim] += x
		a.sumSq[dim] += x * x
	}
}

func (a *seAccumulator) standardErrors() map[string]float64 {
	out := make(map[string]float64, len(dimensionKeys))
	if a.n < 2 {
		return out
	}
	n := float64(a.n)
	for _, dim := range dimensionKeys {
		mean := a.sum[dim] / n
		variance := a.sumSq[dim]/n - mean*mean
		if variance < 0 {
			variance = 0
		}
		out[dim] = math.Sqrt(variance/n) * seInflation[dim]
	}
	return out
}
