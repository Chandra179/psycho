package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"psycho/middleware"
	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

type analyzeResponse struct {
	AnalysisID         string                        `json:"analysis_id"`
	WordCount          int                           `json:"word_count"`
	DictionaryCoverage float64                       `json:"dictionary_coverage"`
	ConfidenceFlag     string                        `json:"confidence_flag"`
	Traits             map[string]profile.TraitResult `json:"traits"`
}

func TestFullPipeline(t *testing.T) {
	logger := zlogger.New("dev")

	profileDeps, err := profile.NewDependencies(profile.Config{DBPath: ":memory:"}, logger)
	if err != nil {
		t.Fatalf("init profile: %v", err)
	}

	analyzeDeps, err := analyze.NewDependencies(analyze.Config{DictionaryPath: "../modules/analyze/dictionary.json"}, logger)
	if err != nil {
		t.Fatalf("init analyze: %v", err)
	}

	ingestCfg := ingest.Config{MaxTextSize: 1_000_000}

	handler := makeHandleAnalyze(ingestCfg, logger, analyzeDeps, profileDeps)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /analyze", handler)
	chain := middleware.Chain(mux, middleware.RequestID)
	server := httptest.NewServer(chain)
	defer server.Close()

	words := make([]string, 1000)
	for i := range words {
		switch i % 10 {
		case 0:
			words[i] = "happy"
		case 1:
			words[i] = "think"
		case 2:
			words[i] = "achieve"
		case 3:
			words[i] = "friend"
		case 4:
			words[i] = "sad"
		case 5:
			words[i] = "always"
		case 6:
			words[i] = "I"
		case 7:
			words[i] = "accommodate"
		default:
			words[i] = "the"
		}
	}
	text := ""
	for i := 0; i < len(words); i += 20 {
		end := min(i+20, len(words))
		for j := i; j < end; j++ {
			text += words[j] + " "
		}
	}

	payload := map[string]string{
		"text":        text,
		"source_type": "blog",
		"source_date": "2024-03-15",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(fmt.Sprintf("%s/analyze", server.URL), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /analyze: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result analyzeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.AnalysisID == "" {
		t.Error("AnalysisID is empty")
	}
	if result.WordCount < 900 {
		t.Errorf("WordCount = %d; expected >= 900", result.WordCount)
	}
	if result.DictionaryCoverage <= 0 {
		t.Errorf("DictionaryCoverage = %f; expected > 0", result.DictionaryCoverage)
	}
	if result.ConfidenceFlag == "" {
		t.Error("ConfidenceFlag is empty")
	}
	if len(result.Traits) != 5 {
		t.Errorf("len(Traits) = %d; want 5", len(result.Traits))
	}

	for traitName, trait := range result.Traits {
		if trait.Score < 0 || trait.Score > 1 {
			t.Errorf("%s score = %f; out of range", traitName, trait.Score)
		}
		if trait.Percentile < 0 || trait.Percentile > 100 {
			t.Errorf("%s percentile = %d; out of range", traitName, trait.Percentile)
		}
		if len(trait.ConfidenceInterval) != 2 {
			t.Errorf("%s CI length = %d; want 2", traitName, len(trait.ConfidenceInterval))
		}
	}

	saved, err := profileDeps.Storage.GetAnalysis(result.AnalysisID)
	if err != nil {
		t.Fatalf("GetAnalysis: %v", err)
	}
	if saved.WordCount != result.WordCount {
		t.Errorf("saved WordCount = %d; want %d", saved.WordCount, result.WordCount)
	}
}

func makeHandleAnalyze(
	cfg ingest.Config,
	logger *zlogger.Logger,
	analyzer *analyze.Dependencies,
	profiler *profile.Dependencies,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type reqStruct struct {
			Text       string `json:"text" validate:"required,min=10"`
			SourceType string `json:"source_type" validate:"required,oneof=blog chat email paste file url"`
			SourceDate string `json:"source_date" validate:"omitempty,datetime=2006-01-02"`
		}
		req, err := middleware.DecodeAndValidate[reqStruct](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if cfg.MaxTextSize > 0 && len(req.Text) > cfg.MaxTextSize {
			http.Error(w, "text exceeds max size", http.StatusBadRequest)
			return
		}

		normalizer := ingest.NewNormalizer()
		doc := normalizer.Normalize(req.Text)

		features, coverage := analyzer.Extractor.Extract(doc)
		scores := analyzer.Model.Infer(features)
		prof := profiler.Aggregator.Aggregate(scores, doc.WordCount, coverage)

		analysisID, err := profiler.Storage.SaveAnalysis(req.SourceType, doc.WordCount, coverage, features, prof)
		if err != nil {
			logger.Error(r.Context(), "failed to save analysis", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := analyzeResponse{
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
