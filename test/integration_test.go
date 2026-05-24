package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"psycho/middleware"
	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

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

	handler := analyze.MakeHandleAnalyze(ingestCfg, logger, analyzeDeps,
		func(sourceType string, wordCount int, coverage float64, features analyze.FeatureVector, scores analyze.BigFiveScores) (string, map[string]any, string, string, error) {
			prof := profileDeps.Aggregator.Aggregate(scores, features, wordCount, coverage)
			analysisID, err := profileDeps.Storage.SaveAnalysis(sourceType, wordCount, coverage, features, prof)
			if err != nil {
				return "", nil, "", "", err
			}
			traits := make(map[string]any, len(prof.Traits))
			for k, v := range prof.Traits {
				traits[k] = v
			}
			return analysisID, traits, prof.ConfidenceFlag, profileDeps.NarrativeGenerator.GenerateSynthesis(prof), nil
		},
	)
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

	var result analyze.AnalyzeResponse
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
	if len(result.Traits) != 7 {
		t.Errorf("len(Traits) = %d; want 7", len(result.Traits))
	}

	if result.Summary == (analyze.SummaryVariables{}) {
		t.Error("Summary is zero; expected computed summary variables")
	}

	if result.Narrative == "" {
		t.Error("Narrative is empty; expected generated narrative")
	}

	for traitName, traitAny := range result.Traits {
		trait := traitAny.(map[string]any)
		score := trait["score"].(float64)
		percentile := int(trait["percentile"].(float64))
		ci := trait["confidence_interval"].([]any)
		if score < 0 || score > 1 {
			t.Errorf("%s score = %f; out of range", traitName, score)
		}
		if percentile < 0 || percentile > 100 {
			t.Errorf("%s percentile = %d; out of range", traitName, percentile)
		}
		if len(ci) != 2 {
			t.Errorf("%s CI length = %d; want 2", traitName, len(ci))
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

func TestFullPipelineAnalyzeDir(t *testing.T) {
	logger := zlogger.New("dev")

	profileDeps, err := profile.NewDependencies(profile.Config{DBPath: ":memory:"}, logger)
	if err != nil {
		t.Fatalf("init profile: %v", err)
	}

	analyzeDeps, err := analyze.NewDependencies(analyze.Config{DictionaryPath: "../modules/analyze/dictionary.json"}, logger)
	if err != nil {
		t.Fatalf("init analyze: %v", err)
	}

	// Create temp dir with .txt files
	tmpDir := t.TempDir()
	writeFile(t, filepath.Join(tmpDir, "sample1.txt"), "happy think achieve friend sad always I accommodate the")
	writeFile(t, filepath.Join(tmpDir, "sample2.txt"), "happy think achieve friend sad always I accommodate the")
	writeFile(t, filepath.Join(tmpDir, "notes.doc"), "should be ignored")
	writeFile(t, filepath.Join(tmpDir, "README.md"), "should be ignored")

	ingestCfg := ingest.Config{MaxTextSize: 1_000_000, DirPath: tmpDir}

	handler := ingest.MakeHandleAnalyzeDir(ingestCfg, logger,
		func(text string, sourceType string) (string, int, float64, string, map[string]any, any, string, error) {
			normalizer := ingest.NewNormalizer()
			doc := normalizer.Normalize(text)
			features, coverage := analyzeDeps.Extractor.Extract(doc)
			scores := analyzeDeps.Model.Infer(features)
			scores.RegulatoryFocus = analyze.ComputeRegulatoryFocus(features)
			scores.NeedForCognition = analyze.ComputeNeedForCognition(features)
			prof := profileDeps.Aggregator.Aggregate(scores, features, doc.WordCount, coverage)
			analysisID, err := profileDeps.Storage.SaveAnalysis(sourceType, doc.WordCount, coverage, features, prof)
			if err != nil {
				return "", 0, 0, "", nil, nil, "", err
			}
			traits := make(map[string]any, len(prof.Traits))
			for k, v := range prof.Traits {
				traits[k] = v
			}
			return analysisID, doc.WordCount, coverage, prof.ConfidenceFlag, traits, prof.Summary, profileDeps.NarrativeGenerator.GenerateSynthesis(prof), nil
		},
	)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /analyze-dir", handler)
	chain := middleware.Chain(mux, middleware.RequestID)
	server := httptest.NewServer(chain)
	defer server.Close()

	payload := map[string]string{
		"source_type": "file",
		"source_date": "2024-03-15",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(fmt.Sprintf("%s/analyze-dir", server.URL), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /analyze-dir: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result ingest.AnalyzeDirResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.AnalysisID == "" {
		t.Error("AnalysisID is empty")
	}
	if result.WordCount < 15 {
		t.Errorf("WordCount = %d; expected >= 15 (9 words * 2 files)", result.WordCount)
	}
	if result.DictionaryCoverage <= 0 {
		t.Errorf("DictionaryCoverage = %f; expected > 0", result.DictionaryCoverage)
	}
	if result.FilesRead != 2 {
		t.Errorf("FilesRead = %d; want 2", result.FilesRead)
	}
	if len(result.Traits) != 7 {
		t.Errorf("len(Traits) = %d; want 7", len(result.Traits))
	}
}

func TestAnalyzeDirWithDataSamples(t *testing.T) {
	logger := zlogger.New("dev")

	profileDeps, err := profile.NewDependencies(profile.Config{DBPath: ":memory:"}, logger)
	if err != nil {
		t.Fatalf("init profile: %v", err)
	}

	analyzeDeps, err := analyze.NewDependencies(analyze.Config{DictionaryPath: "../modules/analyze/dictionary.json"}, logger)
	if err != nil {
		t.Fatalf("init analyze: %v", err)
	}

	samplesDir := "../samples"
	ingestCfg := ingest.Config{MaxTextSize: 1_000_000, DirPath: samplesDir}

	handler := ingest.MakeHandleAnalyzeDir(ingestCfg, logger,
		func(text string, sourceType string) (string, int, float64, string, map[string]any, any, string, error) {
			normalizer := ingest.NewNormalizer()
			doc := normalizer.Normalize(text)
			features, coverage := analyzeDeps.Extractor.Extract(doc)
			scores := analyzeDeps.Model.Infer(features)
			scores.RegulatoryFocus = analyze.ComputeRegulatoryFocus(features)
			scores.NeedForCognition = analyze.ComputeNeedForCognition(features)
			prof := profileDeps.Aggregator.Aggregate(scores, features, doc.WordCount, coverage)
			analysisID, err := profileDeps.Storage.SaveAnalysis(sourceType, doc.WordCount, coverage, features, prof)
			if err != nil {
				return "", 0, 0, "", nil, nil, "", err
			}
			traits := make(map[string]any, len(prof.Traits))
			for k, v := range prof.Traits {
				traits[k] = v
			}
			return analysisID, doc.WordCount, coverage, prof.ConfidenceFlag, traits, prof.Summary, profileDeps.NarrativeGenerator.GenerateSynthesis(prof), nil
		},
	)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /analyze-dir", handler)
	chain := middleware.Chain(mux, middleware.RequestID)
	server := httptest.NewServer(chain)
	defer server.Close()

	payload := map[string]string{
		"source_type": "file",
		"source_date": "2025-05-17",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(fmt.Sprintf("%s/analyze-dir", server.URL), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /analyze-dir: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result ingest.AnalyzeDirResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.AnalysisID == "" {
		t.Error("AnalysisID is empty")
	}
	if result.WordCount < 3000 {
		t.Errorf("WordCount = %d; expected >= 3000 for 4 articles", result.WordCount)
	}
	if result.DictionaryCoverage <= 0 {
		t.Errorf("DictionaryCoverage = %f; expected > 0", result.DictionaryCoverage)
	}
	if result.FilesRead != 8 {
		t.Errorf("FilesRead = %d; want 8", result.FilesRead)
	}
	if len(result.Traits) != 7 {
		t.Errorf("len(Traits) = %d; want 7", len(result.Traits))
	}

	outPath := "../testresults/genz-job-struggles/integration-output.json"
	outDir := filepath.Dir(outPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(outPath, out, 0644); err != nil {
		t.Fatalf("write output: %v", err)
	}
	t.Logf("output written to %s", outPath)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}

func TestIndividualSamples(t *testing.T) {
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

	handler := analyze.MakeHandleAnalyze(ingestCfg, logger, analyzeDeps,
		func(sourceType string, wordCount int, coverage float64, features analyze.FeatureVector, scores analyze.BigFiveScores) (string, map[string]any, string, string, error) {
			prof := profileDeps.Aggregator.Aggregate(scores, features, wordCount, coverage)
			analysisID, err := profileDeps.Storage.SaveAnalysis(sourceType, wordCount, coverage, features, prof)
			if err != nil {
				return "", nil, "", "", err
			}
			traits := make(map[string]any, len(prof.Traits))
			for k, v := range prof.Traits {
				traits[k] = v
			}
			return analysisID, traits, prof.ConfidenceFlag, profileDeps.NarrativeGenerator.GenerateSynthesis(prof), nil
		},
	)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /analyze", handler)
	chain := middleware.Chain(mux, middleware.RequestID)
	server := httptest.NewServer(chain)
	defer server.Close()

	samples := map[string]struct {
		desc           string
		minAuthenticity float64
		maxClout       float64
		minAnalytic    float64
		emotionalLow   bool
		lowConfidence  bool
	}{
		"tweet-thread.txt": {
			desc:          "casual social media — low confidence (<500 words), low analytical thinking, low clout, high authenticity",
			minAuthenticity: 0.70,
			maxClout:       0.50,
			lowConfidence:  true,
		},
		"angry-review.txt": {
			desc:          "consumer rant — low confidence, low emotional tone (high negative emotion), low clout, high authenticity",
			minAuthenticity: 0.50,
			maxClout:       0.45,
			emotionalLow:   true,
			lowConfidence:  true,
		},
		"diary-entry.txt": {
			desc:          "personal confessional — near-zero analytical thinking, near-zero clout, max authenticity, negative emotional tone",
			minAuthenticity: 0.75,
			maxClout:       0.15,
			emotionalLow:   true,
		},
		"research-abstract.txt": {
			desc:          "formal academic paper — highest analytical thinking among samples, moderate authenticity, moderate clout",
			minAnalytic:    0.48,
			minAuthenticity: 0.20,
			maxClout:       0.70,
		},
	}

	entries, err := os.ReadDir("../samples")
	if err != nil {
		t.Fatalf("read samples dir: %v", err)
	}

	var failures []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".txt") {
			continue
		}
		text, err := os.ReadFile(filepath.Join("../samples", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		payload := map[string]string{
			"text":        string(text),
			"source_type": "file",
		}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(fmt.Sprintf("%s/analyze", server.URL), "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST /analyze %s: %v", name, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", name, resp.StatusCode)
		}

		var result analyze.AnalyzeResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}

		s := result.Summary
		expect, hasExpect := samples[name]

		t.Logf("%-28s w=%4d cov=%.2f conf=%s | AT=%.2f CL=%.2f AU=%.2f ET=%.2f | %s",
			name, result.WordCount, result.DictionaryCoverage, result.ConfidenceFlag,
			s.AnalyticalThinking, s.Clout, s.Authenticity, s.EmotionalTone, expect.desc)

		if !hasExpect {
			continue
		}

		if expect.lowConfidence && result.ConfidenceFlag != "low" {
			failures = append(failures, fmt.Sprintf("%s: confidence=%s; want low", name, result.ConfidenceFlag))
		}
		if expect.minAuthenticity > 0 && s.Authenticity < expect.minAuthenticity {
			failures = append(failures, fmt.Sprintf("%s: authenticity=%.2f; want >=%.2f", name, s.Authenticity, expect.minAuthenticity))
		}
		if expect.maxClout > 0 && s.Clout > expect.maxClout {
			failures = append(failures, fmt.Sprintf("%s: clout=%.2f; want <=%.2f", name, s.Clout, expect.maxClout))
		}
		if expect.minAnalytic > 0 && s.AnalyticalThinking < expect.minAnalytic {
			failures = append(failures, fmt.Sprintf("%s: analytical_thinking=%.2f; want >=%.2f", name, s.AnalyticalThinking, expect.minAnalytic))
		}
		if expect.emotionalLow && s.EmotionalTone >= 0.5 {
			failures = append(failures, fmt.Sprintf("%s: emotional_tone=%.2f; want <0.5", name, s.EmotionalTone))
		}
	}

	if len(failures) > 0 {
		t.Errorf("%d assertion failures:\n%s", len(failures), strings.Join(failures, "\n"))
	}
}
