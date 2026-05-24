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
	AnalysisID         string                  `json:"analysis_id"`
	WordCount          int                     `json:"word_count"`
	DictionaryCoverage float64                 `json:"dictionary_coverage"`
	ConfidenceFlag     string                  `json:"confidence_flag"`
	Traits             map[string]any          `json:"traits"`
	Values             map[string]float64      `json:"values"`
	Summary            SummaryVariables        `json:"summary"`
	Narrative          string                  `json:"narrative"`
}

type SaveAnalyzeFunc func(sourceType string, wordCount int, coverage float64, features FeatureVector, scores BigFiveScores) (analysisID string, traits map[string]any, confidenceFlag string, narrative string, err error)

func MakeHandleAnalyze(
	cfg ingest.Config,
	logger *zlogger.Logger,
	analyzer *Dependencies,
	saveFn SaveAnalyzeFunc,
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
		scores.RegulatoryFocus = ComputeRegulatoryFocus(features)
		scores.NeedForCognition = ComputeNeedForCognition(features)
		scores.CognitiveStyle = ComputeCognitiveStyle(features)
		scores.NeedForClosure = ComputeNeedForClosure(features)
		scores.Values = ComputeSchwartzValues(features)

		summary := ComputeSummaryVariables(features)

		analysisID, traits, confidenceFlag, narrative, err := saveFn(req.SourceType, doc.WordCount, coverage, features, scores)
		if err != nil {
			logger.Error(r.Context(), "failed to save analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := AnalyzeResponse{
			AnalysisID:         analysisID,
			WordCount:          doc.WordCount,
			DictionaryCoverage: coverage,
			ConfidenceFlag:     confidenceFlag,
			Traits:             traits,
			Values:             scores.Values,
			Summary:            summary,
			Narrative:          narrative,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
