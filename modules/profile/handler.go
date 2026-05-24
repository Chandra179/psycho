package profile

import (
	"fmt"
	"net/http"

	"psycho/zlogger"
)

func MakeHandleExportPDF(storage *Storage, pdfGen ProfilePDFGenerator, logger *zlogger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "analysis ID required", http.StatusBadRequest)
			return
		}

		prof, err := storage.GetProfile(id)
		if err != nil {
			logger.Error(r.Context(), "profile not found", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "analysis not found", http.StatusNotFound)
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
