package analyze

// BigFiveScores holds the raw regression output for all dimensions.
type BigFiveScores struct {
	Openness          float64
	Conscientiousness float64
	Extraversion      float64
	Agreeableness     float64
	Neuroticism       float64
	RegulatoryFocus   float64
	NeedForCognition  float64
	CognitiveStyle    float64
	NeedForClosure    float64
	Values            map[string]float64
}

// TraitModel is the interface for personality inference.
type TraitModel interface {
	Infer(features FeatureVector) BigFiveScores
}
