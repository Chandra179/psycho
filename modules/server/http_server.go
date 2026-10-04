package server

import (
	"net/http"
	"path/filepath"
	"time"

	"psycho/config"
	"psycho/middleware"
	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/pipeline"
	"psycho/modules/profile"
	"psycho/modules/report"
	"psycho/zlogger"
)

func NewHandler(cfg *config.Config, logger *zlogger.Logger) (http.Handler, error) {
	profileDeps, err := profile.NewDependencies(profile.Config{DBPath: cfg.Profile.DBPath, PDFBackend: cfg.Profile.PDFBackend}, logger)
	if err != nil {
		return nil, err
	}

	analyzeDeps, err := analyze.NewDependencies(analyze.Config{
		DictionaryPath:  cfg.Analyze.DictionaryPath,
		CalibrationPath: cfg.Analyze.CalibrationPath,
	}, logger)
	if err != nil {
		return nil, err
	}

	ingestDeps := ingest.NewDependencies(ingest.Config{MaxTextSize: cfg.Ingest.MaxTextSize, DirPath: cfg.Ingest.DirPath}, logger)

	mwDeps := middleware.NewDependencies(logger)

	// Percentiles come from the calibration corpus when one is configured.
	profileDeps.Aggregator.UseCalibration(analyzeDeps.Calibration)

	// One pipeline wires every stage once; both handlers call into it.
	pipe := pipeline.New(
		analyzeDeps.Extractor,
		analyzeDeps.Model,
		profileDeps.Aggregator,
		profileDeps.NarrativeGenerator,
		profileDeps.Storage,
		analyzeDeps.Calibration,
	)

	mux := http.NewServeMux()

	// Browser surface: paste text, get a report. The upload page and report
	// templates both live in templates/, resolved like config relative to
	// the working directory.
	const templatesDir = "templates"

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(templatesDir, "index.html"))
	})

	// Serve a fixed asset allowlist, never a directory or arbitrary path.
	for _, file := range []string{"app.css", "app.js", "htmx.min.js"} {
		mux.HandleFunc("GET /assets/"+file, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join("assets", file))
		})
	}

	mux.HandleFunc("POST /report", report.MakeHandleReportForm(cfg.Ingest.MaxTextSize, templatesDir, logger, pipe.Run))

	mux.HandleFunc("GET /analysis/{id}/pdf", profile.MakeHandleExportPDF(profileDeps.Storage, profileDeps.PDFGenerator, logger))

	mux.HandleFunc("GET /analysis/{id}", profile.MakeHandleGetAnalysis(profileDeps.Storage, logger))

	mux.HandleFunc("POST /analyze", analyze.MakeHandleAnalyze(cfg.Ingest.MaxTextSize, logger, pipe.Run))

	mux.HandleFunc("POST /analyze-dir", ingest.MakeHandleAnalyzeDir(ingestDeps.Config, logger, pipe.Run))

	chain := middleware.Chain(
		mux,
		mwDeps.Recovery(),
		middleware.RequestID,
		middleware.Timeout(middleware.TimeoutConfig{Duration: time.Duration(cfg.Middleware.TimeoutInSec) * time.Second}),
	)

	return middleware.BrowserSecurity(chain), nil
}
