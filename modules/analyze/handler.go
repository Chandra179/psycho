package analyze

import (
	"encoding/json"
	"fmt"
	"net/http"

	"psycho/middleware"
	"psycho/modules/ingest"
	"psycho/zlogger"
)

type AnalyzeRequest struct {
	Text       string `json:"text" validate:"omitempty"`
	SourceType string `json:"source_type" validate:"required,oneof=blog chat email paste file url"`
	SourceDate string `json:"source_date" validate:"omitempty,datetime=2006-01-02"`
	SourceURL  string `json:"source_url" validate:"omitempty,url"`
}

type AnalyzeResponse struct {
	AnalysisID         string              `json:"analysis_id"`
	WordCount          int                 `json:"word_count"`
	DictionaryCoverage float64             `json:"dictionary_coverage"`
	ConfidenceFlag     string              `json:"confidence_flag"`
	Traits             map[string]any      `json:"traits"`
	Values             map[string]float64  `json:"values"`
	ValueEvidence      map[string][]string `json:"value_evidence,omitempty"`
	Summary            SummaryVariables    `json:"summary"`
	Narrative          string              `json:"narrative"`
}

// MakeHandleAnalyze builds the POST /analyze handler. The analysis itself is
// delegated to analyzeFn (the composed pipeline); this handler only handles
// transport: decode, URL fetch, size limits, response shaping.
func MakeHandleAnalyze(
	maxTextSize int,
	logger *zlogger.Logger,
	analyzeFn ingest.AnalyzeFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Bound the decoded body before parsing: MaxTextSize applies to the
		// extracted text, and the JSON envelope adds little — without this,
		// an oversized body is read into memory in full before the size
		// check can reject it.
		r.Body = http.MaxBytesReader(w, r.Body, int64(maxTextSize)+4096)
		req, err := middleware.DecodeAndValidate[AnalyzeRequest](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		text := req.Text

		if req.SourceType == "url" {
			if req.SourceURL == "" {
				http.Error(w, "source_url is required when source_type=url", http.StatusBadRequest)
				return
			}
			fetched, err := ingest.FetchURLText(req.SourceURL, maxTextSize)
			if err != nil {
				logger.Error(r.Context(), "url fetch failed", zlogger.Field{Key: "error", Value: err.Error()})
				http.Error(w, fmt.Sprintf("failed to fetch URL: %v", err), http.StatusBadGateway)
				return
			}
			text = fetched
		}

		if len(text) < 10 {
			http.Error(w, "text must be at least 10 characters", http.StatusBadRequest)
			return
		}

		if maxTextSize > 0 && len(text) > maxTextSize {
			http.Error(w, "text exceeds max size", http.StatusBadRequest)
			return
		}

		out, err := analyzeFn(r.Context(), req.SourceType, req.SourceDate, text)
		if err != nil {
			logger.Error(r.Context(), "analysis failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		summary, _ := out.Summary.(SummaryVariables)

		resp := AnalyzeResponse{
			AnalysisID:         out.AnalysisID,
			WordCount:          out.WordCount,
			DictionaryCoverage: out.DictionaryCoverage,
			ConfidenceFlag:     out.ConfidenceFlag,
			Traits:             out.Traits,
			Values:             out.Values,
			ValueEvidence:      out.ValueEvidence,
			Summary:            summary,
			Narrative:          out.Narrative,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
