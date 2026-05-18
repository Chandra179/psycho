package analyze

import "math"

// bigFiveModel implements TraitModel using hardcoded regression coefficients.
type bigFiveModel struct{}

func NewBigFiveModel() TraitModel {
	return &bigFiveModel{}
}

func (m *bigFiveModel) Infer(fv FeatureVector) BigFiveScores {
	s := intercepts

	for catName, weights := range coefficients {
		cat := Category(catName)
		pct := fv.CategoryPercents[cat]
		s.Openness += weights.Openness * pct
		s.Conscientiousness += weights.Conscientiousness * pct
		s.Extraversion += weights.Extraversion * pct
		s.Agreeableness += weights.Agreeableness * pct
		s.Neuroticism += weights.Neuroticism * pct
	}

	// Clamp to [0, 1]
	s.Openness = clamp(s.Openness)
	s.Conscientiousness = clamp(s.Conscientiousness)
	s.Extraversion = clamp(s.Extraversion)
	s.Agreeableness = clamp(s.Agreeableness)
	s.Neuroticism = clamp(s.Neuroticism)

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
