package profile

import (
	"reflect"
	"testing"

	"psycho/modules/analyze"
	"psycho/zlogger"
)

func TestCalculationDetailsAndSQLInjectionBoundAsData(t *testing.T) {
	db := openTestDB(t)
	s := NewStorage(db, zlogger.New("prod"))
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}
	attack := "x'); DROP TABLE analyses; --"
	details := &analyze.CalculationDetails{ModelFingerprint: "recorded", Traits: map[string]*analyze.ScoreCalculation{"openness": {Baseline: .5, FinalScore: .52}}, CategoryCounts: map[analyze.Category]int{"article": 3}}
	p := Profile{AnalysisID: attack, CalculationDetails: details, Narrative: "<script>alert(1)</script>"}
	if _, err := s.SaveAnalysis(10, .5, analyze.FeatureVector{}, p); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetAnalysis(attack)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(details, got.CalculationDetails) {
		t.Fatal("JSON or SQL input changed")
	}
	if _, err := s.GetAnalysis("' OR 1=1 --"); err == nil {
		t.Fatal("injection selected a row")
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM analyses").Scan(&count); err != nil || count != 1 {
		t.Fatal("injection damaged database")
	}
}

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

	id, err := storage.SaveAnalysis(1000, 0.7, features, profile)
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

// TestStorageMigratesLegacySchema builds a pre-rename database (scores_json,
// no source_date, no profile_version), runs Migrate, and verifies the
// schema is brought forward without losing the stored row or retaining the
// retired writing metadata columns.
func TestStorageMigratesLegacySchema(t *testing.T) {
	db := openTestDB(t)
	legacy := `
CREATE TABLE analyses (
	id TEXT PRIMARY KEY,
	source_type TEXT NOT NULL,
	word_count INTEGER NOT NULL,
	dictionary_coverage REAL NOT NULL,
	features_json TEXT NOT NULL,
	scores_json TEXT NOT NULL,
	confidence_flag TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`
	if _, err := db.Exec(legacy); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO analyses (id, source_type, word_count, dictionary_coverage, features_json, scores_json, confidence_flag)
		 VALUES ('legacy-1', 'blog', 500, 0.6, '{}', '{"analysis_id":"legacy-1","confidence_flag":"high"}', 'high')`,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	storage := NewStorage(db, zlogger.New("dev"))
	if err := storage.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, col := range []string{"profile_json", "profile_version"} {
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info('analyses') WHERE name = ?`, col,
		).Scan(&count); err != nil {
			t.Fatalf("inspect schema: %v", err)
		}
		if count != 1 {
			t.Errorf("column %q missing after migration", col)
		}
	}
	for _, col := range []string{"source_type", "source_date"} {
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info('analyses') WHERE name = ?`, col,
		).Scan(&count); err != nil {
			t.Fatalf("inspect retired metadata column: %v", err)
		}
		if count != 0 {
			t.Errorf("retired metadata column %q should be removed after migration", col)
		}
	}
	var scoresJSON int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('analyses') WHERE name = 'scores_json'`,
	).Scan(&scoresJSON); err != nil || scoresJSON != 0 {
		t.Errorf("scores_json should be gone after migration (count=%d, err=%v)", scoresJSON, err)
	}

	saved, err := storage.GetAnalysis("legacy-1")
	if err != nil {
		t.Fatalf("GetAnalysis on migrated row: %v", err)
	}
	if saved.ConfidenceFlag != "high" || saved.ProfileVersion != 1 {
		t.Errorf("migrated row: flag=%q version=%d; want high/1", saved.ConfidenceFlag, saved.ProfileVersion)
	}
}
