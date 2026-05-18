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
