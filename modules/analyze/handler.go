package analyze

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"psycho/middleware"
	"psycho/modules/ingest"
	"psycho/zlogger"
)

type AnalyzeRequest struct {
	Text      string `json:"text" validate:"omitempty"`
	SourceURL string `json:"source_url" validate:"omitempty,url"`
}

type AnalyzeResponse struct {
	AnalysisID          string                      `json:"analysis_id"`
	WordCount           int                         `json:"word_count"`
	DictionaryCoverage  float64                     `json:"dictionary_coverage"`
	ConfidenceFlag      string                      `json:"confidence_flag"`
	Traits              map[string]any              `json:"traits"`
	Values              map[string]float64          `json:"values"`
	ValueEvidence       map[string][]string         `json:"value_evidence,omitempty"`
	PercentileReference *ingest.PercentileReference `json:"percentile_reference,omitempty"`
	CalculationDetails  *CalculationDetails         `json:"calculation_details,omitempty"`
	Summary             SummaryVariables            `json:"summary"`
	Narrative           string                      `json:"narrative"`
}

// MakeHandleAnalyze builds the POST /analyze handler. The analysis itself is
// delegated to analyzeFn (the composed pipeline); this handler only handles
// transport: decode, URL fetch, size limits, response shaping.
func MakeHandleAnalyze(
	maxTextSize int,
	logger *zlogger.Logger,
	analyzeFn ingest.AnalyzeFunc,
) http.HandlerFunc {
	return makeHandleAnalyze(maxTextSize, logger, analyzeFn, ingest.FetchURLText)
}

func makeHandleAnalyze(maxTextSize int, logger *zlogger.Logger, analyzeFn ingest.AnalyzeFunc, fetchURL func(string, int) (string, error)) http.HandlerFunc {
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

		if req.SourceURL != "" {
			fetched, err := fetchURL(req.SourceURL, maxTextSize)
			if err != nil {
				logger.Error(r.Context(), "url fetch failed", zlogger.Field{Key: "error", Value: err.Error()})
				http.Error(w, fmt.Sprintf("failed to fetch URL: %v", err), http.StatusBadGateway)
				return
			}
			text = fetched
		}

		if maxTextSize > 0 && len(text) > maxTextSize {
			http.Error(w, "text exceeds max size", http.StatusBadRequest)
			return
		}

		out, err := analyzeFn(r.Context(), text)
		if err != nil {
			if errors.Is(err, ingest.ErrInvalidText) {
				http.Error(w, ingest.ErrInvalidText.Error(), http.StatusBadRequest)
				return
			}
			logger.Error(r.Context(), "analysis failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		summary, _ := out.Summary.(SummaryVariables)
		calculations, _ := out.CalculationDetails.(*CalculationDetails)

		resp := AnalyzeResponse{
			AnalysisID:          out.AnalysisID,
			WordCount:           out.WordCount,
			DictionaryCoverage:  out.DictionaryCoverage,
			ConfidenceFlag:      out.ConfidenceFlag,
			Traits:              out.Traits,
			Values:              out.Values,
			ValueEvidence:       out.ValueEvidence,
			PercentileReference: out.PercentileReference,
			CalculationDetails:  calculations,
			Summary:             summary,
			Narrative:           out.Narrative,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
