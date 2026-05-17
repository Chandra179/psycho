package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"psycho/middleware"
	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

// AnalyzeRequest is the payload for POST /analyze.
type AnalyzeRequest struct {
	Text       string `json:"text" validate:"omitempty"`
	SourceType string `json:"source_type" validate:"required,oneof=blog chat email paste file url"`
	SourceDate string `json:"source_date" validate:"omitempty,datetime=2006-01-02"`
	SourceURL  string `json:"source_url" validate:"omitempty,url"`
}

// AnalyzeResponse is the JSON output.
type AnalyzeResponse struct {
	AnalysisID         string                        `json:"analysis_id"`
	WordCount          int                           `json:"word_count"`
	DictionaryCoverage float64                       `json:"dictionary_coverage"`
	ConfidenceFlag     string                        `json:"confidence_flag"`
	Traits             map[string]profile.TraitResult `json:"traits"`
}

func makeHandleAnalyze(
	cfg ingest.Config,
	logger *zlogger.Logger,
	analyzer *analyze.Dependencies,
	profiler *profile.Dependencies,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			fetched, err := ingest.FetchURLText(req.SourceURL)
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

		if cfg.MaxTextSize > 0 && len(text) > cfg.MaxTextSize {
			http.Error(w, "text exceeds max size", http.StatusBadRequest)
			return
		}

		normalizer := ingest.NewNormalizer()
		doc := normalizer.Normalize(text)

		features, coverage := analyzer.Extractor.Extract(doc)
		scores := analyzer.Model.Infer(features)
		prof := profiler.Aggregator.Aggregate(scores, doc.WordCount, coverage)

		analysisID, err := profiler.Storage.SaveAnalysis(req.SourceType, doc.WordCount, coverage, features, prof)
		if err != nil {
			logger.Error(r.Context(), "failed to save analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := AnalyzeResponse{
			AnalysisID:         analysisID,
			WordCount:          doc.WordCount,
			DictionaryCoverage: coverage,
			ConfidenceFlag:     prof.ConfidenceFlag,
			Traits:             prof.Traits,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
