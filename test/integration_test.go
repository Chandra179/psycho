package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/pipeline"
	"psycho/modules/profile"
)

// newTestPipeline wires the real modules against the real dictionary, the
// same composition the browser build performs, with no calibration.
func newTestPipeline(t *testing.T) *pipeline.Pipeline {
	t.Helper()
	analyzeDeps, err := analyze.NewDependencies(analyze.Config{DictionaryPath: "../modules/analyze/dictionary.json"})
	if err != nil {
		t.Fatalf("init analyze: %v", err)
	}
	return pipeline.New(
		analyzeDeps.Extractor,
		analyzeDeps.Model,
		profile.NewScoreAggregator(),
		profile.NewTemplateNarrativeGenerator(),
		analyzeDeps.Calibration,
	)
}

func hasMatchedWordSamples(traits map[string]any) bool {
	for _, traitValue := range traits {
		trait, ok := traitValue.(profile.TraitResult)
		if !ok {
			continue
		}
		for _, row := range trait.Evidence {
			if len(row.MatchedWords) > 0 {
				return true
			}
		}
	}
	return false
}

func TestFullPipeline(t *testing.T) {
	pipe := newTestPipeline(t)

	pool := []string{"happy", "think", "achieve", "friend", "sad", "always", "I", "accommodate", "the", "the"}
	var text strings.Builder
	for i := 0; i < 1000; i++ {
		text.WriteString(pool[i%len(pool)])
		text.WriteString(" ")
	}

	result, err := pipe.Run(t.Context(), text.String())
	if err != nil {
		t.Fatalf("Run: %v", err)
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
	if len(result.Traits) != 9 {
		t.Errorf("len(Traits) = %d; want 9", len(result.Traits))
	}
	if result.PercentileReference == nil || result.PercentileReference.Method != ingest.PercentileMethodNormalApproximation {
		t.Errorf("uncalibrated response should describe normal-approximation percentiles: %+v", result.PercentileReference)
	}
	if !hasMatchedWordSamples(result.Traits) {
		t.Error("response should include matched-word evidence samples")
	}

	for traitName, traitAny := range result.Traits {
		trait := traitAny.(profile.TraitResult)
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
}

// TestAllSamplesAnalyze counts the files in samples/ so adding or removing a
// sample cannot silently skip analysis.
func TestAllSamplesAnalyze(t *testing.T) {
	pipe := newTestPipeline(t)
	samples, err := filepath.Glob("../samples/*.txt")
	if err != nil || len(samples) == 0 {
		t.Fatalf("no sample files found: %v", err)
	}
	var all strings.Builder
	for _, path := range samples {
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		all.Write(text)
		all.WriteString("\n\n")
	}
	result, err := pipe.Run(t.Context(), all.String())
	if err != nil {
		t.Fatal(err)
	}
	if result.WordCount < 3000 {
		t.Errorf("WordCount = %d; expected >= 3000 for the combined samples", result.WordCount)
	}
	if len(result.Traits) != 9 {
		t.Errorf("len(Traits) = %d; want 9", len(result.Traits))
	}
}

func TestIndividualSamples(t *testing.T) {
	pipe := newTestPipeline(t)

	samples := map[string]struct {
		desc            string
		minAuthenticity float64
		maxAuthenticity float64
		maxClout        float64
		minAnalytic     float64
		emotionalLow    bool
		lowConfidence   bool
	}{
		"tweet-thread.txt": {
			desc:            "casual social media — low confidence (<500 words), low analytical thinking, low clout, relatively high authenticity",
			minAuthenticity: 0.25,
			maxClout:        0.50,
			lowConfidence:   true,
		},
		"angry-review.txt": {
			desc:            "consumer rant — low confidence, low emotional tone (high negative emotion), low clout, relatively high authenticity",
			minAuthenticity: 0.28,
			maxClout:        0.45,
			emotionalLow:    true,
			lowConfidence:   true,
		},
		"diary-entry.txt": {
			desc:            "personal confessional — near-zero analytical thinking, near-zero clout, max authenticity, negative emotional tone",
			minAuthenticity: 0.75,
			maxClout:        0.15,
			emotionalLow:    true,
		},
		"research-abstract.txt": {
			desc:            "formal academic paper — highest analytical thinking among samples, lowest authenticity (polished, complex vocabulary), moderate clout",
			minAnalytic:     0.48,
			maxAuthenticity: 0.15,
			maxClout:        0.70,
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

		out, err := pipe.Run(t.Context(), string(text))
		if err != nil {
			t.Fatalf("analyze %s: %v", name, err)
		}
		result := struct {
			WordCount          int
			DictionaryCoverage float64
			ConfidenceFlag     string
			Summary            analyze.SummaryVariables
		}{out.WordCount, out.DictionaryCoverage, out.ConfidenceFlag, out.Summary.(analyze.SummaryVariables)}

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
		if expect.maxAuthenticity > 0 && s.Authenticity > expect.maxAuthenticity {
			failures = append(failures, fmt.Sprintf("%s: authenticity=%.2f; want <=%.2f", name, s.Authenticity, expect.maxAuthenticity))
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
