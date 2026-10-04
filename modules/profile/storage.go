package profile

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
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

// profileSchemaVersion versions the JSON blob stored in profile_json. Bump
// it whenever the marshaled Profile shape changes incompatibly.
const profileSchemaVersion = 1

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
	profile_json TEXT NOT NULL,
	profile_version INTEGER NOT NULL DEFAULT 1,
	confidence_flag TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	if _, err := s.db.Exec(q); err != nil {
		return err
	}

	// Pre-2026-10 databases stored the profile blob in a column misleadingly
	// named scores_json; rename it in place.
	if ok, err := s.columnExists("scores_json"); err != nil {
		return err
	} else if ok {
		if has, err := s.columnExists("profile_json"); err != nil {
			return err
		} else if !has {
			if _, err := s.db.Exec(`ALTER TABLE analyses RENAME COLUMN scores_json TO profile_json`); err != nil {
				return fmt.Errorf("rename scores_json: %w", err)
			}
		}
	}

	if err := s.ensureColumn("source_date", `ALTER TABLE analyses ADD COLUMN source_date TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add source_date column: %w", err)
	}
	if err := s.ensureColumn("profile_version", `ALTER TABLE analyses ADD COLUMN profile_version INTEGER NOT NULL DEFAULT 1`); err != nil {
		return fmt.Errorf("add profile_version column: %w", err)
	}
	return nil
}

func (s *Storage) columnExists(name string) (bool, error) {
	var count int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('analyses') WHERE name = ?`, name,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect analyses schema: %w", err)
	}
	return count > 0, nil
}

func (s *Storage) ensureColumn(name, alter string) error {
	ok, err := s.columnExists(name)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	if _, err := s.db.Exec(alter); err != nil {
		return err
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
		`INSERT INTO analyses (id, source_type, source_date, word_count, dictionary_coverage, features_json, profile_json, profile_version, confidence_flag)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.AnalysisID, sourceType, sourceDate, wordCount, coverage, string(featuresJSON), string(profileJSON), profileSchemaVersion, profile.ConfidenceFlag,
	)
	if err != nil {
		return "", fmt.Errorf("insert analysis: %w", err)
	}
	return profile.AnalysisID, nil
}

// GetProfile retrieves a full Profile from storage by analysis ID.
func (s *Storage) GetProfile(id string) (Profile, error) {
	var profileJSON string
	err := s.db.QueryRow(
		`SELECT profile_json FROM analyses WHERE id = ?`, id,
	).Scan(&profileJSON)
	if err != nil {
		return Profile{}, err
	}
	var prof Profile
	if err := json.Unmarshal([]byte(profileJSON), &prof); err != nil {
		return Profile{}, fmt.Errorf("unmarshal profile: %w", err)
	}
	return prof, nil
}

// GetAnalysis retrieves a saved analysis by ID.
func (s *Storage) GetAnalysis(id string) (*SavedAnalysis, error) {
	var a SavedAnalysis
	var featuresJSON, profileJSON string
	err := s.db.QueryRow(
		`SELECT id, source_type, source_date, word_count, dictionary_coverage, features_json, profile_json, profile_version, confidence_flag, created_at
		 FROM analyses WHERE id = ?`, id,
	).Scan(&a.ID, &a.SourceType, &a.SourceDate, &a.WordCount, &a.Coverage, &featuresJSON, &profileJSON, &a.ProfileVersion, &a.ConfidenceFlag, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(featuresJSON), &a.Features); err != nil {
		return nil, fmt.Errorf("unmarshal features: %w", err)
	}
	var prof Profile
	if err := json.Unmarshal([]byte(profileJSON), &prof); err != nil {
		return nil, fmt.Errorf("unmarshal profile: %w", err)
	}
	a.Scores = prof.Traits
	a.Summary = prof.Summary
	a.Values = prof.Values
	a.ValueEvidence = prof.ValueEvidence
	a.PercentileReference = prof.PercentileReference
	a.CalculationDetails = prof.CalculationDetails
	a.Narrative = prof.Narrative
	return &a, nil
}

// SavedAnalysis is the full stored analysis as returned by the retrieval
// endpoint.
type SavedAnalysis struct {
	ID                  string                      `json:"id"`
	SourceType          string                      `json:"source_type"`
	SourceDate          string                      `json:"source_date,omitempty"`
	WordCount           int                         `json:"word_count"`
	Coverage            float64                     `json:"dictionary_coverage"`
	Features            map[string]float64          `json:"features"`
	Scores              map[string]TraitResult      `json:"scores"`
	Values              map[string]float64          `json:"values,omitempty"`
	ValueEvidence       map[string][]string         `json:"value_evidence,omitempty"`
	PercentileReference *ingest.PercentileReference `json:"percentile_reference,omitempty"`
	CalculationDetails  *analyze.CalculationDetails `json:"calculation_details,omitempty"`
	Summary             analyze.SummaryVariables    `json:"summary"`
	Narrative           string                      `json:"narrative,omitempty"`
	ConfidenceFlag      string                      `json:"confidence_flag"`
	ProfileVersion      int                         `json:"profile_version"`
	CreatedAt           string                      `json:"created_at"`
}
