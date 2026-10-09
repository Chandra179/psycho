package analyze

import (
	"math"
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
			"negative_emotion":  25.0,
			"pronoun":           20.0,
			"tentative":         15.0,
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

func TestComputeCognitiveStyleCategorical(t *testing.T) {
	// Formal prose: many articles and prepositions, few pronouns and auxiliaries.
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"article": 8.0, "preposition": 16.0, "personal_pronoun": 2.0, "impersonal_pronoun": 4.0,
			"auxiliary_verb": 5.0, "adverb": 2.0, "conjunction": 3.0, "negation": 0.5,
		},
	}
	score := ComputeCognitiveStyle(fv)
	if score < 0.65 {
		t.Errorf("expected categorical style (score >= 0.65), got %f", score)
	}
	if label := ComputeCognitiveStyleLabel(score); label != "systematic" {
		t.Errorf("expected systematic label, got %s", label)
	}
}

func TestComputeCognitiveStyleDynamic(t *testing.T) {
	// Narrative writing: many pronouns, auxiliaries, adverbs and conjunctions.
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"article": 4.0, "preposition": 9.0, "personal_pronoun": 15.0, "impersonal_pronoun": 6.0,
			"auxiliary_verb": 12.0, "adverb": 7.0, "conjunction": 8.0, "negation": 2.0,
		},
	}
	score := ComputeCognitiveStyle(fv)
	if score > 0.35 {
		t.Errorf("expected dynamic style (score <= 0.35), got %f", score)
	}
	if label := ComputeCognitiveStyleLabel(score); label != "intuitive" {
		t.Errorf("expected intuitive label, got %s", label)
	}
}

func TestComputeCognitiveStyleIgnoresLongWordsAndContentCategories(t *testing.T) {
	base := FeatureVector{CategoryPercents: map[Category]float64{"article": 5.0, "personal_pronoun": 8.0}}
	other := FeatureVector{
		CategoryPercents: map[Category]float64{"article": 5.0, "personal_pronoun": 8.0, "cognitive_process": 20.0, "cause": 10.0, "sensation": 9.0},
		BigWordRatio:     0.5,
	}
	if a, b := ComputeCognitiveStyle(base), ComputeCognitiveStyle(other); a != b {
		t.Errorf("only the eight function-word categories may move this score: %f vs %f", a, b)
	}
}

func TestComputeCognitiveStyleTypicalTextIsMixed(t *testing.T) {
	// Reference-corpus averages (CDI about -19.7 points) must land mid-scale.
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"article": 4.0, "preposition": 9.0, "personal_pronoun": 11.0, "impersonal_pronoun": 5.0,
			"auxiliary_verb": 9.0, "adverb": 5.0, "conjunction": 3.65, "negation": 0.0,
		},
	}
	score := ComputeCognitiveStyle(fv)
	if math.Abs(score-0.50) > 0.02 {
		t.Errorf("typical text should score near 0.50, got %f", score)
	}
	if label := ComputeCognitiveStyleLabel(score); label != "mixed" {
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
