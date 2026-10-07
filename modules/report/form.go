package report

import (
	"encoding/json"
	"errors"
	"net/http"

	"psycho/modules/ingest"
	"psycho/zlogger"
)

// MakeHandleReportForm handles the browser upload form (POST /report): it
// runs the analysis and returns the rendered report directly in the
// response — as an HTML fragment for HTMX (HX-Request header) or as a full
// standalone page otherwise. No separate report URL exists.
func MakeHandleReportForm(
	maxTextSize int,
	templatesDir string,
	logger *zlogger.Logger,
	analyzeFn ingest.AnalyzeFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Same bound as the JSON API: reject oversized bodies before
		// ParseForm reads them into memory.
		r.Body = http.MaxBytesReader(w, r.Body, int64(maxTextSize)+4096)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		// The MVP accepts only the writer's own text; the checkbox is the
		// consent record (see discovery issue #22 for the third-party stance).
		if r.PostFormValue("consent") != "on" {
			http.Error(w, "Please confirm the writing is your own; the consent box is required.", http.StatusBadRequest)
			return
		}

		text := r.PostFormValue("text")

		if maxTextSize > 0 && len(text) > maxTextSize {
			http.Error(w, "Text exceeds the maximum size.", http.StatusBadRequest)
			return
		}

		out, err := analyzeFn(r.Context(), text)
		if err != nil {
			if errors.Is(err, ingest.ErrInvalidText) {
				http.Error(w, ingest.ErrInvalidText.Error(), http.StatusBadRequest)
				return
			}
			logger.Error(r.Context(), "analysis failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "Something went wrong while analyzing; please try again.", http.StatusInternalServerError)
			return
		}

		// AnalysisOutput is the JSON contract at this seam; round-trip it
		// into the typed report input.
		blob, err := json.Marshal(out)
		if err != nil {
			logger.Error(r.Context(), "failed to encode analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "Something went wrong while analyzing; please try again.", http.StatusInternalServerError)
			return
		}
		var a Analysis
		if err := json.Unmarshal(blob, &a); err != nil {
			logger.Error(r.Context(), "failed to decode analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "Something went wrong while analyzing; please try again.", http.StatusInternalServerError)
			return
		}

		fullPage := r.Header.Get("HX-Request") != "true"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := RenderAnalysis(templatesDir, &a, w, fullPage); err != nil {
			// Headers are already written; log for diagnosis.
			logger.Error(r.Context(), "failed to render report", zlogger.Field{Key: "error", Value: err.Error()})
		}
	}
}
