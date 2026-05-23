package analyze

import (
	"math"
	"testing"

	"psycho/modules/ingest"
)

func TestFeatureExtractor(t *testing.T) {
	jsonData := []byte(`{"positive_emotion": ["happy"], "negative_emotion": ["sad"]}`)
	dict, _ := LoadDictionaryFromJSON(jsonData)
	extractor := NewFeatureExtractor(dict)

	doc := ingest.Document{RawText: "happy sad happy", WordCount: 3, TypeTokenRatio: 1.0}
	fv, coverage := extractor.Extract(doc)

	if coverage != 1.0 {
		t.Errorf("coverage = %f; want 1.0", coverage)
	}
	wantPos := 200.0 / 3.0
	if math.Abs(fv.CategoryPercents["positive_emotion"]-wantPos) > 0.01 {
		t.Errorf("positive_emotion %% = %f; want %f", fv.CategoryPercents["positive_emotion"], wantPos)
	}
	wantNeg := 100.0 / 3.0
	if math.Abs(fv.CategoryPercents["negative_emotion"]-wantNeg) > 0.01 {
		t.Errorf("negative_emotion %% = %f; want %f", fv.CategoryPercents["negative_emotion"], wantNeg)
	}
}

func TestFeatureExtractorEmpty(t *testing.T) {
	jsonData := []byte(`{"positive_emotion": ["happy"]}`)
	dict, _ := LoadDictionaryFromJSON(jsonData)
	extractor := NewFeatureExtractor(dict)

	doc := ingest.Document{RawText: "", WordCount: 0}
	fv, coverage := extractor.Extract(doc)
	if coverage != 0 {
		t.Errorf("coverage = %f; want 0", coverage)
	}
	if fv.CategoryPercents != nil {
		if v, ok := fv.CategoryPercents["positive_emotion"]; ok && v != 0 {
			t.Errorf("positive_emotion = %f; want 0", v)
		}
	}
}

func TestComputeSummaryVariables(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"positive_emotion":  10.0,
			"negative_emotion":  2.0,
			"article":           8.0,
			"pronoun":           12.0,
			"cognitive_process": 6.0,
			"tentative":         4.0,
			"certainty":         3.0,
			"cause":             2.0,
			"social":            5.0,
			"achievement":       3.0,
			"quantitative":      4.0,
		},
	}
	sv := ComputeSummaryVariables(fv)

	if sv.EmotionalTone <= 0.5 {
		t.Errorf("EmotionalTone = %f; expected > 0.5 (more positive than negative)", sv.EmotionalTone)
	}
	if sv.AnalyticalThinking <= 0.3 || sv.AnalyticalThinking >= 0.9 {
		t.Errorf("AnalyticalThinking = %f; want reasonable value in (0.3, 0.9)", sv.AnalyticalThinking)
	}
	if sv.Clout < 0.19 || sv.Clout >= 0.9 {
		t.Errorf("Clout = %f; want reasonable value in (0.2, 0.9)", sv.Clout)
	}
	if sv.Authenticity <= 0.1 || sv.Authenticity >= 0.9 {
		t.Errorf("Authenticity = %f; want reasonable value in (0.1, 0.9)", sv.Authenticity)
	}
}

func TestComputeSummaryVariablesEmpty(t *testing.T) {
	sv := ComputeSummaryVariables(FeatureVector{})
	if sv.EmotionalTone != 0.5 {
		t.Errorf("EmotionalTone = %f; want 0.5 for empty vector", sv.EmotionalTone)
	}
	if sv.AnalyticalThinking != 0.5 {
		t.Errorf("AnalyticalThinking = %f; want 0.5 for empty vector", sv.AnalyticalThinking)
	}
}
