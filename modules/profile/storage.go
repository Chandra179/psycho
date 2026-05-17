package profile

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"psycho/modules/analyze"
	"psycho/zlogger"
)

// Storage persists analysis results to SQLite.
type Storage struct {
	db     *sql.DB
	logger *zlogger.Logger
}

func NewStorage(db *sql.DB, logger *zlogger.Logger) *Storage {
	return &Storage{db: db, logger: logger}
}

// Migrate creates the analyses table.
func (s *Storage) Migrate() error {
	q := `
CREATE TABLE IF NOT EXISTS analyses (
	id TEXT PRIMARY KEY,
	source_type TEXT NOT NULL,
	word_count INTEGER NOT NULL,
	dictionary_coverage REAL NOT NULL,
	features_json TEXT NOT NULL,
	scores_json TEXT NOT NULL,
	confidence_flag TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := s.db.Exec(q)
	return err
}

// SaveAnalysis persists a profile and returns the analysis ID.
func (s *Storage) SaveAnalysis(sourceType string, wordCount int, coverage float64, features analyze.FeatureVector, profile Profile) (string, error) {
	featuresJSON, err := json.Marshal(features.CategoryPercents)
	if err != nil {
		return "", fmt.Errorf("marshal features: %w", err)
	}
	scoresJSON, err := json.Marshal(profile.Traits)
	if err != nil {
		return "", fmt.Errorf("marshal scores: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO analyses (id, source_type, word_count, dictionary_coverage, features_json, scores_json, confidence_flag)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		profile.AnalysisID, sourceType, wordCount, coverage, string(featuresJSON), string(scoresJSON), profile.ConfidenceFlag,
	)
	if err != nil {
		return "", fmt.Errorf("insert analysis: %w", err)
	}
	return profile.AnalysisID, nil
}

// GetAnalysis retrieves a saved analysis by ID.
func (s *Storage) GetAnalysis(id string) (*SavedAnalysis, error) {
	var a SavedAnalysis
	var featuresJSON, scoresJSON string
	err := s.db.QueryRow(
		`SELECT id, source_type, word_count, dictionary_coverage, features_json, scores_json, confidence_flag, created_at
		 FROM analyses WHERE id = ?`, id,
	).Scan(&a.ID, &a.SourceType, &a.WordCount, &a.Coverage, &featuresJSON, &scoresJSON, &a.ConfidenceFlag, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(featuresJSON), &a.Features); err != nil {
		return nil, fmt.Errorf("unmarshal features: %w", err)
	}
	if err := json.Unmarshal([]byte(scoresJSON), &a.Scores); err != nil {
		return nil, fmt.Errorf("unmarshal scores: %w", err)
	}
	return &a, nil
}

// SavedAnalysis is the database row representation.
type SavedAnalysis struct {
	ID              string
	SourceType      string
	WordCount       int
	Coverage        float64
	Features        map[string]float64
	Scores          map[string]TraitResult
	ConfidenceFlag  string
	CreatedAt       string
}
