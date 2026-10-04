package supervised

import (
	"math"
	"sort"
)

// AverageRanks returns 1-based ranks with ties resolved to the average rank
// (fractional), which is what both Spearman and Mann-Whitney require.
func AverageRanks(values []float64) []float64 {
	type pair struct {
		value float64
		index int
	}
	pairs := make([]pair, len(values))
	for i, v := range values {
		pairs[i] = pair{v, i}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].value < pairs[j].value })

	ranks := make([]float64, len(values))
	for i := 0; i < len(pairs); {
		j := i
		for j < len(pairs) && pairs[j].value == pairs[i].value {
			j++
		}
		avg := float64(i+1+j) / 2 // mean of ranks i+1 .. j (1-based)
		for k := i; k < j; k++ {
			ranks[pairs[k].index] = avg
		}
		i = j
	}
	return ranks
}

// Pearson computes the Pearson correlation of two equal-length samples.
func Pearson(x, y []float64) float64 {
	n := float64(len(x))
	if len(x) < 2 || len(x) != len(y) {
		return math.NaN()
	}
	var sx, sy, sxx, syy, sxy float64
	for i := range x {
		if !finite(x[i]) || !finite(y[i]) {
			return math.NaN()
		}
		sx += x[i]
		sy += y[i]
		sxx += x[i] * x[i]
		syy += y[i] * y[i]
		sxy += x[i] * y[i]
	}
	cov := sxy - sx*sy/n
	vx := sxx - sx*sx/n
	vy := syy - sy*sy/n
	if vx <= 0 || vy <= 0 {
		return math.NaN()
	}
	return cov / math.Sqrt(vx*vy)
}

// Spearman is the Pearson correlation of the average ranks: the standard
// rank correlation, tie-correct.
func Spearman(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return math.NaN()
	}
	return Pearson(AverageRanks(x), AverageRanks(y))
}

// AUCBinary scores how well scores separate positives from negatives
// (Mann-Whitney U divided by n_pos*n_neg; 0.5 = chance, 1.0 = perfect).
// Average ranks make it tie-safe. Returns NaN unless both classes are
// present.
func AUCBinary(scores []float64, positive []bool) float64 {
	if len(scores) != len(positive) {
		return math.NaN()
	}
	for _, score := range scores {
		if !finite(score) {
			return math.NaN()
		}
	}
	var nPos, nNeg int
	for _, p := range positive {
		if p {
			nPos++
		} else {
			nNeg++
		}
	}
	if nPos == 0 || nNeg == 0 {
		return math.NaN()
	}

	ranks := AverageRanks(scores)
	var rankSumPos float64
	for i, r := range ranks {
		if positive[i] {
			rankSumPos += r
		}
	}
	u := rankSumPos - float64(nPos)*float64(nPos+1)/2
	return u / (float64(nPos) * float64(nNeg))
}

// AUCBootstrapCI resamples the sample with replacement and returns the 2.5th
// and 97.5th percentile AUCs of the bootstrap distribution.
func AUCBootstrapCI(scores []float64, positive []bool, resamples int, rng interface{ Intn(int) int }) (float64, float64) {
	if resamples < 1 || len(scores) < 2 || len(scores) != len(positive) {
		return math.NaN(), math.NaN()
	}
	aucs := make([]float64, 0, resamples)
	idx := make([]int, len(scores))
	for r := 0; r < resamples; r++ {
		for i := range idx {
			idx[i] = rng.Intn(len(scores))
		}
		sub := make([]float64, len(idx))
		subPos := make([]bool, len(idx))
		for i, j := range idx {
			sub[i] = scores[j]
			subPos[i] = positive[j]
		}
		if a := AUCBinary(sub, subPos); !math.IsNaN(a) {
			aucs = append(aucs, a)
		}
	}
	if len(aucs) < 2 {
		return math.NaN(), math.NaN()
	}
	sort.Float64s(aucs)
	lo := aucs[int(0.025*float64(len(aucs)-1)+0.5)]
	hi := aucs[int(0.975*float64(len(aucs)-1)+0.5)]
	return lo, hi
}
