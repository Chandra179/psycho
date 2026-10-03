package report

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"psycho/modules/ingest"
	"psycho/zlogger"
)

// validSourceTypes mirrors the JSON API's source_type constraint.
var validSourceTypes = map[string]bool{
	"blog": true, "chat": true, "email": true, "paste": true, "file": true, "url": true,
}

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
			http.Error(w, "Please confirm the writing is your own — the consent box is required.", http.StatusBadRequest)
			return
		}

		text := strings.TrimSpace(r.PostFormValue("text"))
		if len(text) < 10 {
			http.Error(w, "Text must be at least 10 characters.", http.StatusBadRequest)
			return
		}
		if maxTextSize > 0 && len(text) > maxTextSize {
			http.Error(w, "Text exceeds the maximum size.", http.StatusBadRequest)
			return
		}

		sourceType := r.PostFormValue("source_type")
		if sourceType == "" {
			sourceType = "paste"
		}
		if !validSourceTypes[sourceType] {
			http.Error(w, "Invalid kind of writing.", http.StatusBadRequest)
			return
		}

		// Match the JSON API's validation (datetime=2006-01-02) so the DB
		// never stores an unparseable date.
		sourceDate := r.PostFormValue("source_date")
		if sourceDate != "" {
			if _, err := time.Parse("2006-01-02", sourceDate); err != nil {
				http.Error(w, "Invalid written-on date — use YYYY-MM-DD.", http.StatusBadRequest)
				return
			}
		}

		out, err := analyzeFn(r.Context(), sourceType, sourceDate, text)
		if err != nil {
			logger.Error(r.Context(), "analysis failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "Something went wrong while analyzing — please try again.", http.StatusInternalServerError)
			return
		}

		// AnalysisOutput is the JSON contract at this seam; round-trip it
		// into the typed report input.
		blob, err := json.Marshal(out)
		if err != nil {
			logger.Error(r.Context(), "failed to encode analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "Something went wrong while analyzing — please try again.", http.StatusInternalServerError)
			return
		}
		var a Analysis
		if err := json.Unmarshal(blob, &a); err != nil {
			logger.Error(r.Context(), "failed to decode analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "Something went wrong while analyzing — please try again.", http.StatusInternalServerError)
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
