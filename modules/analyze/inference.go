package analyze

// BigFiveScores holds the raw heuristic output for all dimensions.
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
	Calculations      map[string]*ScoreCalculation
}

// TraitModel is the interface for personality inference.
type TraitModel interface {
	Infer(features FeatureVector) BigFiveScores
}

// ScoreFeatures scores every calibrated dimension with its calculation trace.
// The pipeline and cmd/calibrate both call it, so the calibration corpus is
// scored exactly as production text is.
func ScoreFeatures(model TraitModel, fv FeatureVector) BigFiveScores {
	scores := model.Infer(fv)
	if scores.Calculations == nil {
		scores.Calculations = make(map[string]*ScoreCalculation)
	}
	scores.Calculations["regulatory_focus"] = ComputeRegulatoryFocusCalculation(fv)
	scores.Calculations["need_for_cognition"] = ComputeNeedForCognitionCalculation(fv)
	scores.Calculations["cognitive_style"] = ComputeCognitiveStyleCalculation(fv)
	scores.Calculations["need_for_closure"] = ComputeNeedForClosureCalculation(fv)
	scores.RegulatoryFocus = scores.Calculations["regulatory_focus"].FinalScore
	scores.NeedForCognition = scores.Calculations["need_for_cognition"].FinalScore
	scores.CognitiveStyle = scores.Calculations["cognitive_style"].FinalScore
	scores.NeedForClosure = scores.Calculations["need_for_closure"].FinalScore
	scores.Values = ComputeSchwartzValues(fv)
	return scores
}
