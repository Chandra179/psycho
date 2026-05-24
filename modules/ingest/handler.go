package ingest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"psycho/middleware"
	"psycho/zlogger"
)

type AnalyzeDirRequest struct {
	SourceType string `json:"source_type" validate:"omitempty,oneof=blog chat email paste file url"`
	SourceDate string `json:"source_date" validate:"omitempty,datetime=2006-01-02"`
}

type AnalyzeDirResponse struct {
	AnalysisID         string         `json:"analysis_id"`
	WordCount          int            `json:"word_count"`
	DictionaryCoverage float64        `json:"dictionary_coverage"`
	ConfidenceFlag     string         `json:"confidence_flag"`
	Traits             map[string]any `json:"traits"`
	FilesRead          int            `json:"files_read"`
	Summary            any            `json:"summary"`
	Narrative          string         `json:"narrative"`
}

type AnalyzeDirFunc func(text string, sourceType string) (analysisID string, wordCount int, coverage float64, confidenceFlag string, traits map[string]any, summary any, narrative string, err error)

func MakeHandleAnalyzeDir(
	cfg Config,
	logger *zlogger.Logger,
	analyzeFn AnalyzeDirFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := middleware.DecodeAndValidate[AnalyzeDirRequest](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if cfg.DirPath == "" {
			http.Error(w, "dir_path not configured", http.StatusServiceUnavailable)
			return
		}

		sourceType := req.SourceType
		if sourceType == "" {
			sourceType = "file"
		}

		text, filesRead, err := ReadDir(cfg.DirPath)
		if err != nil {
			logger.Error(r.Context(), "failed to read directory", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "failed to read directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(text) < 10 {
			http.Error(w, "combined text must be at least 10 characters", http.StatusBadRequest)
			return
		}

		if cfg.MaxTextSize > 0 && len(text) > cfg.MaxTextSize {
			http.Error(w, "combined text exceeds max size", http.StatusBadRequest)
			return
		}

		analysisID, wordCount, coverage, confidenceFlag, traits, summary, narrative, err := analyzeFn(text, sourceType)
		if err != nil {
			logger.Error(r.Context(), "analysis failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := AnalyzeDirResponse{
			AnalysisID:         analysisID,
			WordCount:          wordCount,
			DictionaryCoverage: coverage,
			ConfidenceFlag:     confidenceFlag,
			Traits:             traits,
			FilesRead:          filesRead,
			Summary:            summary,
			Narrative:          narrative,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func FetchURLText(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ReadDir(dirPath string) (string, int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", 0, fmt.Errorf("read dir %s: %w", dirPath, err)
	}

	var builder strings.Builder
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dirPath, e.Name()))
		if err != nil {
			return "", 0, fmt.Errorf("read file %s: %w", e.Name(), err)
		}
		if count > 0 {
			builder.WriteString("\n\n")
		}
		builder.Write(b)
		count++
	}

	if count == 0 {
		return "", 0, fmt.Errorf("no .txt files found in %s", dirPath)
	}

	return builder.String(), count, nil
}
