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
		pct := fv.CategoryPercents[cat] // percentage of words in this category
		// scale pct: 1 percentage point -> weight * 0.01
		scale := pct * 0.01
		s.Openness += weights.Openness * scale
		s.Conscientiousness += weights.Conscientiousness * scale
		s.Extraversion += weights.Extraversion * scale
		s.Agreeableness += weights.Agreeableness * scale
		s.Neuroticism += weights.Neuroticism * scale
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
