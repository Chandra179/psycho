package profile

import (
	"testing"

	"psycho/modules/analyze"
	"psycho/zlogger"
)

func TestStorageMigrateAndSave(t *testing.T) {
	db := openTestDB(t)
	storage := NewStorage(db, zlogger.New("dev"))
	if err := storage.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	profile := Profile{
		AnalysisID:     "test-id-123",
		ConfidenceFlag: "high",
		Traits: map[string]TraitResult{
			"openness": {Score: 0.75, Percentile: 75, ConfidenceInterval: []float64{0.70, 0.80}},
		},
	}
	features := analyze.FeatureVector{
		CategoryPercents: map[analyze.Category]float64{"positive_emotion": 5.0},
	}

	id, err := storage.SaveAnalysis("blog", 1000, 0.7, features, profile)
	if err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}
	if id != "test-id-123" {
		t.Errorf("id = %q; want test-id-123", id)
	}

	saved, err := storage.GetAnalysis(id)
	if err != nil {
		t.Fatalf("GetAnalysis: %v", err)
	}
	if saved.ConfidenceFlag != "high" {
		t.Errorf("ConfidenceFlag = %q; want high", saved.ConfidenceFlag)
	}
	if saved.WordCount != 1000 {
		t.Errorf("WordCount = %d; want 1000", saved.WordCount)
	}
	if saved.Scores["openness"].Score != 0.75 {
		t.Errorf("Openness score = %f; want 0.75", saved.Scores["openness"].Score)
	}
}
