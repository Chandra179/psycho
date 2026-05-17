package analyze

// BigFiveScores holds the raw regression output for OCEAN.
type BigFiveScores struct {
	Openness          float64
	Conscientiousness float64
	Extraversion      float64
	Agreeableness     float64
	Neuroticism       float64
}

// TraitModel is the interface for personality inference.
type TraitModel interface {
	Infer(features FeatureVector) BigFiveScores
}
