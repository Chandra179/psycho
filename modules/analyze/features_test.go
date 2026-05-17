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
	// Empty doc: map is nil or has zero values — both are acceptable
	if fv.CategoryPercents != nil {
		if v, ok := fv.CategoryPercents["positive_emotion"]; ok && v != 0 {
			t.Errorf("positive_emotion = %f; want 0", v)
		}
	}
}
