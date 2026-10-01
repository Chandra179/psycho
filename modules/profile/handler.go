package profile

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"psycho/zlogger"
)

// MakeHandleGetAnalysis returns a stored analysis as JSON.
func MakeHandleGetAnalysis(storage *Storage, logger *zlogger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "analysis ID required", http.StatusBadRequest)
			return
		}

		analysis, err := storage.GetAnalysis(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "analysis not found", http.StatusNotFound)
				return
			}
			logger.Error(r.Context(), "failed to load analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(analysis)
	}
}

func MakeHandleExportPDF(storage *Storage, pdfGen ProfilePDFGenerator, logger *zlogger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "analysis ID required", http.StatusBadRequest)
			return
		}

		prof, err := storage.GetProfile(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "analysis not found", http.StatusNotFound)
				return
			}
			logger.Error(r.Context(), "failed to load profile", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		pdf, err := pdfGen.Generate(prof)
		if err != nil {
			logger.Error(r.Context(), "pdf generation failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "pdf generation failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"profile-%s.pdf\"", prof.AnalysisID))
		_, _ = w.Write(pdf)
	}
}
