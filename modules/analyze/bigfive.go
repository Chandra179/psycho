package analyze

import (
	"maps"
	"math"
	"slices"
)

// bigFiveModel implements a correlation-weighted heuristic with fixed weights.
type bigFiveModel struct{}

func NewBigFiveModel() TraitModel {
	return &bigFiveModel{}
}

func (m *bigFiveModel) Infer(fv FeatureVector) BigFiveScores {
	s := intercepts
	s.Calculations = make(map[string]*ScoreCalculation, 5)
	for _, dim := range dimensionKeys[:5] {
		s.Calculations[dim] = newScoreCalculation(dimensionValue(dim, &s))
	}

	for _, catName := range slices.Sorted(maps.Keys(coefficients)) {
		weights := coefficients[catName]
		cat := Category(catName)
		pct := fv.CategoryPercents[cat]
		s.Openness += weights.Openness * pct
		s.Conscientiousness += weights.Conscientiousness * pct
		s.Extraversion += weights.Extraversion * pct
		s.Agreeableness += weights.Agreeableness * pct
		s.Neuroticism += weights.Neuroticism * pct
		for _, dim := range dimensionKeys[:5] {
			w := weights.weightFor(dim)
			s.Calculations[dim].addTerm(fv, catName, pct, w, w*pct)
		}
	}

	for _, dim := range dimensionKeys[:5] {
		setDimensionValue(dim, &s, s.Calculations[dim].finish(dimensionValue(dim, &s)))
	}

	return s
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return math.Round(v*100) / 100
}
