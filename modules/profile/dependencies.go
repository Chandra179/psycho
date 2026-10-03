package profile

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"psycho/zlogger"
)

type Dependencies struct {
	Config             Config
	Logger             *zlogger.Logger
	Aggregator         *ScoreAggregator
	Storage            *Storage
	NarrativeGenerator NarrativeGenerator
	PDFGenerator       ProfilePDFGenerator
}

func NewDependencies(cfg Config, logger *zlogger.Logger) (*Dependencies, error) {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite allows a single writer; multiple connections would race on
	// concurrent analyses and surface SQLITE_BUSY as HTTP 500s. One
	// connection serializes everything at negligible cost for this workload.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	storage := NewStorage(db, logger)
	if err := storage.Migrate(); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	var pdfGen ProfilePDFGenerator
	switch cfg.PDFBackend {
	case "", "maroto":
		pdfGen = NewMarotoPDFGenerator()
	default:
		return nil, fmt.Errorf("unknown pdf_backend %q (supported: maroto)", cfg.PDFBackend)
	}

	return &Dependencies{
		Config:             cfg,
		Logger:             logger,
		Aggregator:         NewScoreAggregator(),
		Storage:            storage,
		NarrativeGenerator: NewTemplateNarrativeGenerator(),
		PDFGenerator:       pdfGen,
	}, nil
}
