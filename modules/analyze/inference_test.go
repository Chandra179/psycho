package analyze

import (
	"testing"
)

func TestBigFiveModelInfer(t *testing.T) {
	model := NewBigFiveModel()

	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"cognitive_process": 15.0,
			"tentative":         10.0,
			"negative_emotion":  12.0,
			"pronoun":           8.0,
			"article":           5.0,
			"inclusive":         3.0,
			"achievement":       4.0,
		},
		TypeTokenRatio: 0.7,
		BigWordRatio:   0.15,
		AvgWordLength:  4.5,
	}

	scores := model.Infer(fv)

	if scores.Openness <= 0 || scores.Openness > 1 {
		t.Errorf("Openness out of range: %f", scores.Openness)
	}
	if scores.Conscientiousness <= 0 || scores.Conscientiousness > 1 {
		t.Errorf("Conscientiousness out of range: %f", scores.Conscientiousness)
	}
	if scores.Extraversion <= 0 || scores.Extraversion > 1 {
		t.Errorf("Extraversion out of range: %f", scores.Extraversion)
	}
	if scores.Agreeableness <= 0 || scores.Agreeableness > 1 {
		t.Errorf("Agreeableness out of range: %f", scores.Agreeableness)
	}
	if scores.Neuroticism <= 0 || scores.Neuroticism > 1 {
		t.Errorf("Neuroticism out of range: %f", scores.Neuroticism)
	}
}

func TestBigFiveModelHighOpenness(t *testing.T) {
	model := NewBigFiveModel()
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"article":   20.0,
			"inclusive": 15.0,
		},
	}
	scores := model.Infer(fv)
	if scores.Openness < 0.58 {
		t.Errorf("expected elevated Openness, got %f", scores.Openness)
	}
}

func TestBigFiveModelHighConscientiousness(t *testing.T) {
	model := NewBigFiveModel()
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"achievement": 30.0,
		},
	}
	scores := model.Infer(fv)
	if scores.Conscientiousness < 0.58 {
		t.Errorf("expected elevated Conscientiousness, got %f", scores.Conscientiousness)
	}
}

func TestBigFiveModelLowWordCount(t *testing.T) {
	model := NewBigFiveModel()
	fv := FeatureVector{CategoryPercents: map[Category]float64{}}
	scores := model.Infer(fv)
	if scores.Openness != 0.50 {
		t.Errorf("baseline Openness = %f; expected 0.50", scores.Openness)
	}
}

func TestBigFiveModelHighNeuroticism(t *testing.T) {
	model := NewBigFiveModel()
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"negative_emotion": 25.0,
			"pronoun":          20.0,
			"tentative":        15.0,
			"cognitive_process": 10.0,
		},
	}
	scores := model.Infer(fv)
	if scores.Neuroticism < 0.5 {
		t.Errorf("expected elevated Neuroticism, got %f", scores.Neuroticism)
	}
}

func TestBigFiveModelHighExtraversion(t *testing.T) {
	model := NewBigFiveModel()
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"positive_emotion": 20.0,
			"social":           20.0,
			"pronoun":          15.0,
		},
	}
	scores := model.Infer(fv)
	if scores.Extraversion < 0.53 {
		t.Errorf("expected elevated Extraversion, got %f", scores.Extraversion)
	}
}

func TestComputeRegulatoryFocusPromotion(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"promotion_focus":  15.0,
			"prevention_focus": 2.0,
		},
	}
	score := ComputeRegulatoryFocus(fv)
	if score < 0.55 {
		t.Errorf("expected elevated promotion focus, got %f", score)
	}
	label := ComputeRegulatoryFocusLabel(score)
	if label != "promotion_focus" {
		t.Errorf("expected promotion_focus label, got %s", label)
	}
}

func TestComputeRegulatoryFocusPrevention(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"promotion_focus":  1.0,
			"prevention_focus": 12.0,
		},
	}
	score := ComputeRegulatoryFocus(fv)
	if score > 0.45 {
		t.Errorf("expected low promotion focus, got %f", score)
	}
	label := ComputeRegulatoryFocusLabel(score)
	if label != "prevention_focus" {
		t.Errorf("expected prevention_focus label, got %s", label)
	}
}

func TestComputeRegulatoryFocusBalanced(t *testing.T) {
	fv := FeatureVector{CategoryPercents: map[Category]float64{}}
	score := ComputeRegulatoryFocus(fv)
	if score != 0.50 {
		t.Errorf("expected balanced 0.50, got %f", score)
	}
	label := ComputeRegulatoryFocusLabel(score)
	if label != "balanced" {
		t.Errorf("expected balanced label, got %s", label)
	}
}

func TestComputeNeedForCognitionHigh(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"analytic_thinking":  20.0,
			"intuitive_thinking": 2.0,
		},
	}
	score := ComputeNeedForCognition(fv)
	if score < 0.55 {
		t.Errorf("expected high need for cognition, got %f", score)
	}
	label := ComputeNeedForCognitionLabel(score)
	if label != "high" {
		t.Errorf("expected high label, got %s", label)
	}
}

func TestComputeNeedForCognitionLow(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"analytic_thinking":  1.0,
			"intuitive_thinking": 15.0,
		},
	}
	score := ComputeNeedForCognition(fv)
	if score > 0.45 {
		t.Errorf("expected low need for cognition, got %f", score)
	}
	label := ComputeNeedForCognitionLabel(score)
	if label != "low" {
		t.Errorf("expected low label, got %s", label)
	}
}

func TestComputeCognitiveStyleSystematic(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"cognitive_process":  15.0,
			"cause":              12.0,
			"certainty":          10.0,
			"big_words":           8.0,
			"analytic_thinking":  12.0,
			"sensation":           3.0,
			"pronoun":             8.0,
			"present_focus":       5.0,
			"intuitive_thinking":  2.0,
			"tentative":           4.0,
		},
	}
	score := ComputeCognitiveStyle(fv)
	if score < 0.55 {
		t.Errorf("expected systematic style (score >= 0.55), got %f", score)
	}
	label := ComputeCognitiveStyleLabel(score)
	if label != "systematic" {
		t.Errorf("expected systematic label, got %s", label)
	}
}

func TestComputeCognitiveStyleIntuitive(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"cognitive_process":  3.0,
			"cause":              4.0,
			"certainty":          3.0,
			"big_words":          1.0,
			"analytic_thinking":  2.0,
			"sensation":          12.0,
			"pronoun":            18.0,
			"present_focus":      15.0,
			"intuitive_thinking": 12.0,
			"tentative":          10.0,
		},
	}
	score := ComputeCognitiveStyle(fv)
	if score > 0.45 {
		t.Errorf("expected intuitive style (score <= 0.45), got %f", score)
	}
	label := ComputeCognitiveStyleLabel(score)
	if label != "intuitive" {
		t.Errorf("expected intuitive label, got %s", label)
	}
}

func TestComputeCognitiveStyleMixed(t *testing.T) {
	fv := FeatureVector{CategoryPercents: map[Category]float64{}}
	score := ComputeCognitiveStyle(fv)
	if score != 0.50 {
		t.Errorf("expected mixed 0.50, got %f", score)
	}
	label := ComputeCognitiveStyleLabel(score)
	if label != "mixed" {
		t.Errorf("expected mixed label, got %s", label)
	}
}

func TestComputeNeedForClosureHigh(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"certainty": 20.0,
			"tentative": 2.0,
		},
	}
	score := ComputeNeedForClosure(fv)
	if score < 0.55 {
		t.Errorf("expected high need for closure, got %f", score)
	}
	label := ComputeNeedForClosureLabel(score)
	if label != "high" {
		t.Errorf("expected high label, got %s", label)
	}
}

func TestComputeNeedForClosureLow(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"certainty": 1.0,
			"tentative": 20.0,
		},
	}
	score := ComputeNeedForClosure(fv)
	if score > 0.45 {
		t.Errorf("expected low need for closure, got %f", score)
	}
	label := ComputeNeedForClosureLabel(score)
	if label != "low" {
		t.Errorf("expected low label, got %s", label)
	}
}

func TestComputeNeedForClosureModerate(t *testing.T) {
	fv := FeatureVector{CategoryPercents: map[Category]float64{}}
	score := ComputeNeedForClosure(fv)
	if score != 0.50 {
		t.Errorf("expected moderate 0.50, got %f", score)
	}
	label := ComputeNeedForClosureLabel(score)
	if label != "moderate" {
		t.Errorf("expected moderate label, got %s", label)
	}
}
