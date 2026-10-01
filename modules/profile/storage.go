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

// Migrate creates the analyses table and brings older databases up to the
// current schema.
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
	if _, err := s.db.Exec(q); err != nil {
		return err
	}

	var colCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('analyses') WHERE name = 'source_date'`,
	).Scan(&colCount); err != nil {
		return fmt.Errorf("inspect analyses schema: %w", err)
	}
	if colCount == 0 {
		if _, err := s.db.Exec(`ALTER TABLE analyses ADD COLUMN source_date TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add source_date column: %w", err)
		}
	}
	return nil
}

// SaveAnalysis persists a profile and returns the analysis ID.
func (s *Storage) SaveAnalysis(sourceType, sourceDate string, wordCount int, coverage float64, features analyze.FeatureVector, profile Profile) (string, error) {
	featuresJSON, err := json.Marshal(features.CategoryPercents)
	if err != nil {
		return "", fmt.Errorf("marshal features: %w", err)
	}
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("marshal profile: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO analyses (id, source_type, source_date, word_count, dictionary_coverage, features_json, scores_json, confidence_flag)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.AnalysisID, sourceType, sourceDate, wordCount, coverage, string(featuresJSON), string(profileJSON), profile.ConfidenceFlag,
	)
	if err != nil {
		return "", fmt.Errorf("insert analysis: %w", err)
	}
	return profile.AnalysisID, nil
}

// GetProfile retrieves a full Profile from storage by analysis ID.
func (s *Storage) GetProfile(id string) (Profile, error) {
	var scoresJSON string
	err := s.db.QueryRow(
		`SELECT scores_json FROM analyses WHERE id = ?`, id,
	).Scan(&scoresJSON)
	if err != nil {
		return Profile{}, err
	}
	var prof Profile
	if err := json.Unmarshal([]byte(scoresJSON), &prof); err != nil {
		return Profile{}, fmt.Errorf("unmarshal profile: %w", err)
	}
	return prof, nil
}

// GetAnalysis retrieves a saved analysis by ID.
func (s *Storage) GetAnalysis(id string) (*SavedAnalysis, error) {
	var a SavedAnalysis
	var featuresJSON, scoresJSON string
	err := s.db.QueryRow(
		`SELECT id, source_type, source_date, word_count, dictionary_coverage, features_json, scores_json, confidence_flag, created_at
		 FROM analyses WHERE id = ?`, id,
	).Scan(&a.ID, &a.SourceType, &a.SourceDate, &a.WordCount, &a.Coverage, &featuresJSON, &scoresJSON, &a.ConfidenceFlag, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(featuresJSON), &a.Features); err != nil {
		return nil, fmt.Errorf("unmarshal features: %w", err)
	}
	var prof Profile
	if err := json.Unmarshal([]byte(scoresJSON), &prof); err != nil {
		return nil, fmt.Errorf("unmarshal profile: %w", err)
	}
	a.Scores = prof.Traits
	a.Summary = prof.Summary
	a.Values = prof.Values
	a.ValueEvidence = prof.ValueEvidence
	a.Narrative = prof.Narrative
	return &a, nil
}

// SavedAnalysis is the full stored analysis as returned by the retrieval
// endpoint.
type SavedAnalysis struct {
	ID             string                   `json:"id"`
	SourceType     string                   `json:"source_type"`
	SourceDate     string                   `json:"source_date,omitempty"`
	WordCount      int                      `json:"word_count"`
	Coverage       float64                  `json:"dictionary_coverage"`
	Features       map[string]float64       `json:"features"`
	Scores         map[string]TraitResult   `json:"scores"`
	Values         map[string]float64       `json:"values,omitempty"`
	ValueEvidence  map[string][]string      `json:"value_evidence,omitempty"`
	Summary        analyze.SummaryVariables `json:"summary"`
	Narrative      string                   `json:"narrative,omitempty"`
	ConfidenceFlag string                   `json:"confidence_flag"`
	CreatedAt      string                   `json:"created_at"`
}
