package integration_test

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

// validationPipeline wires the real modules against the real dictionary.
type validationPipeline struct {
	extractor *analyze.FeatureExtractor
	model     analyze.TraitModel
	pd        *profile.Dependencies
}

func newValidationPipeline(t *testing.T) *validationPipeline {
	t.Helper()
	logger := zlogger.New("prod")

	pd, err := profile.NewDependencies(profile.Config{DBPath: ":memory:"}, logger)
	if err != nil {
		t.Fatalf("init profile: %v", err)
	}
	ad, err := analyze.NewDependencies(analyze.Config{DictionaryPath: "../modules/analyze/dictionary.json"}, logger)
	if err != nil {
		t.Fatalf("init analyze: %v", err)
	}
	return &validationPipeline{extractor: ad.Extractor, model: ad.Model, pd: pd}
}

// runAnalysis runs a word pool through normalize -> extract -> infer and
// returns the raw dimension scores.
func (vp *validationPipeline) runAnalysis(t *testing.T, pool []string, total int) analyze.BigFiveScores {
	t.Helper()
	words := make([]string, 0, total)
	for len(words) < total {
		words = append(words, pool...)
	}
	words = words[:total]

	doc := ingest.NewNormalizer().Normalize(strings.Join(words, " "))
	fv, _ := vp.extractor.Extract(doc)

	scores := vp.model.Infer(fv)
	scores.RegulatoryFocus = analyze.ComputeRegulatoryFocus(fv)
	scores.NeedForCognition = analyze.ComputeNeedForCognition(fv)
	scores.CognitiveStyle = analyze.ComputeCognitiveStyle(fv)
	scores.NeedForClosure = analyze.ComputeNeedForClosure(fv)
	scores.Values = analyze.ComputeSchwartzValues(fv)
	return scores
}

func assertInRange(t *testing.T, name string, v float64) {
	t.Helper()
	if v < 0 || v > 1 {
		t.Errorf("%s out of [0,1]: %f", name, v)
	}
}

// TestExactFeatureExtraction feeds a constructed 100-word text whose expected
// category percentages are known by construction and asserts the extractor
// reproduces them exactly. This is the regression net for double-counting,
// category moves, and coverage math.
func TestExactFeatureExtraction(t *testing.T) {
	vp := newValidationPipeline(t)

	// 100 words: 25 "the" (article), 25 "happy" (positive_emotion),
	// 25 "i" (pronoun), 25 "zqx" (matches nothing).
	text := strings.TrimSpace(strings.Repeat("the happy i zqx ", 25))
	doc := ingest.NewNormalizer().Normalize(text)
	fv, coverage := vp.extractor.Extract(doc)

	if doc.WordCount != 100 {
		t.Errorf("word count = %d; want 100", doc.WordCount)
	}
	for cat, want := range map[analyze.Category]float64{
		"article":          25,
		"positive_emotion": 25,
		"pronoun":          25,
	} {
		if got := fv.CategoryPercents[cat]; math.Abs(got-want) > 0.01 {
			t.Errorf("%s = %.4f; want %.1f", cat, got, want)
		}
	}
	if math.Abs(coverage-0.75) > 0.001 {
		t.Errorf("coverage = %.4f; want 0.75", coverage)
	}
	// the(3) happy(5) i(1) zqx(3) — no word longer than six letters.
	if fv.BigWordRatio != 0 {
		t.Errorf("BigWordRatio = %f; want 0", fv.BigWordRatio)
	}

	// LIWC Sixltr: share of words with more than six letters.
	fv2, _ := vp.extractor.Extract(ingest.NewNormalizer().Normalize("accommodate the"))
	if math.Abs(fv2.BigWordRatio-0.5) > 0.001 {
		t.Errorf("BigWordRatio = %f; want 0.5", fv2.BigWordRatio)
	}
}

// TestDictionaryLoaderDeduplicates pins the load-time dedup: a word listed
// twice under one category must count once.
func TestDictionaryLoaderDeduplicates(t *testing.T) {
	dict, err := analyze.LoadDictionaryFromJSON([]byte(`{"certainty": ["indeed", "indeed"]}`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	extractor := analyze.NewFeatureExtractor(dict)
	doc := ingest.NewNormalizer().Normalize("indeed indeed")
	fv, coverage := extractor.Extract(doc)

	if got := fv.CategoryPercents["certainty"]; math.Abs(got-100) > 0.01 {
		t.Errorf("certainty = %.1f; want 100 (deduped)", got)
	}
	if math.Abs(coverage-1.0) > 0.001 {
		t.Errorf("coverage = %.4f; want 1.0", coverage)
	}
}

// TestDictionaryCategoryMembership pins the word-to-category placements the
// inference models depend on.
func TestDictionaryCategoryMembership(t *testing.T) {
	vp := newValidationPipeline(t)
	pct := func(text string) map[analyze.Category]float64 {
		doc := ingest.NewNormalizer().Normalize(text)
		fv, coverage := vp.extractor.Extract(doc)
		if coverage == 0 {
			t.Fatalf("%q matched nothing in the dictionary", text)
		}
		return fv.CategoryPercents
	}

	// Cliticized pronouns must count as pronouns.
	if pct("i'm")["pronoun"] == 0 {
		t.Error("i'm did not count as pronoun")
	}
	if pct("they're")["pronoun"] == 0 {
		t.Error("they're did not count as pronoun")
	}
	// Negation words live in negation, not exclusive.
	if pct("don't")["negation"] == 0 {
		t.Error("don't did not count as negation")
	}
	if pct("not")["negation"] == 0 {
		t.Error("not did not count as negation")
	}
	if pct("not")["exclusive"] != 0 {
		t.Error("not still counted as exclusive")
	}
	// Multi-category membership survives (never: certainty + time + negation).
	n := pct("never")
	if n["certainty"] == 0 || n["time"] == 0 || n["negation"] == 0 {
		t.Errorf("never lost a membership: certainty=%f time=%f negation=%f", n["certainty"], n["time"], n["negation"])
	}
}

// TestDimensionDirectionality feeds paired synthetic texts with known
// linguistic profiles through the full pipeline: the high-profile text must
// score at least minGap higher than the low-profile text on the target
// dimension. Coefficients are all published signs (Yarkoni 2010 Table 1),
// so a sign flip or a dropped category shows up here.
func TestDimensionDirectionality(t *testing.T) {
	vp := newValidationPipeline(t)

	cases := []struct {
		name   string
		high   []string
		low    []string
		score  func(analyze.BigFiveScores) float64
		minGap float64
	}{
		{
			name:   "openness",
			high:   []string{"the", "a", "and", "with", "also"},     // articles + inclusive
			low:    []string{"i", "yesterday", "go", "was", "went"}, // pronouns + time/motion/past
			score:  func(s analyze.BigFiveScores) float64 { return s.Openness },
			minGap: 0.3,
		},
		{
			name:   "conscientiousness",
			high:   []string{"win", "success", "goal", "discipline", "focus"}, // achievement
			low:    []string{"sad", "angry", "but", "not", "don't"},           // negative emotion + exclusive + negation
			score:  func(s analyze.BigFiveScores) float64 { return s.Conscientiousness },
			minGap: 0.3,
		},
		{
			name:   "extraversion",
			high:   []string{"friend", "talk", "team", "chat", "happy"}, // social + positive emotion
			low:    []string{"zqx"},                                     // neutral filler
			score:  func(s analyze.BigFiveScores) float64 { return s.Extraversion },
			minGap: 0.3,
		},
		{
			name:   "agreeableness",
			high:   []string{"with", "and", "up", "down", "happy"},    // inclusive + space + positive emotion
			low:    []string{"sad", "angry", "bitter", "hurt", "mad"}, // negative emotion
			score:  func(s analyze.BigFiveScores) float64 { return s.Agreeableness },
			minGap: 0.3,
		},
		{
			name:   "neuroticism",
			high:   []string{"sad", "anxious", "nervous", "afraid", "worried"}, // negative emotion
			low:    []string{"the", "a", "an", "zqx"},                          // articles (weak negative)
			score:  func(s analyze.BigFiveScores) float64 { return s.Neuroticism },
			minGap: 0.2,
		},
		{
			name:   "regulatory_focus",
			high:   []string{"aspire", "thrive", "ambitious", "flourish", "maximize"}, // promotion
			low:    []string{"duty", "vigilant", "defend", "preserve", "shelter"},     // prevention
			score:  func(s analyze.BigFiveScores) float64 { return s.RegulatoryFocus },
			minGap: 0.5,
		},
		{
			name:   "need_for_cognition",
			high:   []string{"analyze", "logic", "hypothesis", "evidence", "therefore"}, // analytic
			low:    []string{"gut", "instinct", "hunch", "obvious", "simple"},           // intuitive
			score:  func(s analyze.BigFiveScores) float64 { return s.NeedForCognition },
			minGap: 0.5,
		},
		{
			name:   "need_for_closure",
			high:   []string{"always", "definitely", "certainly", "clearly", "absolutely"}, // certainty
			low:    []string{"maybe", "perhaps", "possibly", "probably", "seem"},           // tentative
			score:  func(s analyze.BigFiveScores) float64 { return s.NeedForClosure },
			minGap: 0.5,
		},
		{
			name:   "cognitive_style",
			high:   []string{"think", "because", "logic", "hypothesis", "insight"}, // systematic
			low:    []string{"feel", "look", "touch", "gut", "obvious"},            // intuitive
			score:  func(s analyze.BigFiveScores) float64 { return s.CognitiveStyle },
			minGap: 0.3,
		},
	}

	for _, tc := range cases {
		high := vp.runAnalysis(t, tc.high, 400)
		low := vp.runAnalysis(t, tc.low, 400)
		h, l := tc.score(high), tc.score(low)
		assertInRange(t, tc.name+" (high)", h)
		assertInRange(t, tc.name+" (low)", l)
		if h-l < tc.minGap {
			t.Errorf("%s: high=%.3f low=%.3f gap=%.3f; want gap >= %.2f", tc.name, h, l, h-l, tc.minGap)
		}
	}
}

type latencyReport struct {
	GeneratedAt string        `json:"generated_at"`
	Note        string        `json:"note"`
	Corpora     []latencySize `json:"corpora"`
}

type latencySize struct {
	Words int     `json:"words"`
	Runs  int     `json:"runs"`
	P50ms float64 `json:"p50_ms"`
	P95ms float64 `json:"p95_ms"`
}

func percentileOf(sorted []float64, p float64) float64 {
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// TestLatencyBenchmarks measures the full analysis pipeline (normalize,
// extract, infer, aggregate, persist to SQLite, synthesize narrative) over
// 1,000- and 5,000-word corpora and records p50/p95 per run. The PRD target
// is <5s for a 5,000-word corpus; results land in
// testresults/benchmarks/latency.json as a single-machine regression signal.
func TestLatencyBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("latency benchmarks skipped in -short mode")
	}
	vp := newValidationPipeline(t)

	pool := []string{
		"the", "happy", "i", "think", "achieve", "friend", "sad", "always",
		"accommodate", "yesterday", "because", "maybe", "team", "goal", "money",
	}

	report := latencyReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Note:        "single-machine regression signal, not production evidence",
	}

	sizes := []struct {
		words int
		runs  int
	}{{1000, 30}, {5000, 20}}

	for _, size := range sizes {
		words := make([]string, 0, size.words)
		for len(words) < size.words {
			words = append(words, pool...)
		}
		words = words[:size.words]
		text := strings.Join(words, " ")

		var durations []float64
		for i := 0; i < size.runs; i++ {
			start := time.Now()

			doc := ingest.NewNormalizer().Normalize(text)
			fv, coverage := vp.extractor.Extract(doc)
			scores := vp.model.Infer(fv)
			scores.RegulatoryFocus = analyze.ComputeRegulatoryFocus(fv)
			scores.NeedForCognition = analyze.ComputeNeedForCognition(fv)
			scores.CognitiveStyle = analyze.ComputeCognitiveStyle(fv)
			scores.NeedForClosure = analyze.ComputeNeedForClosure(fv)
			scores.Values = analyze.ComputeSchwartzValues(fv)
			prof := vp.pd.Aggregator.Aggregate(scores, fv, doc.WordCount, coverage)
			if _, err := vp.pd.Storage.SaveAnalysis("blog", "2026-09-29", doc.WordCount, coverage, fv, prof); err != nil {
				t.Fatalf("save analysis: %v", err)
			}
			_ = vp.pd.NarrativeGenerator.GenerateSynthesis(prof)

			durations = append(durations, time.Since(start).Seconds()*1000)
		}

		sort.Float64s(durations)
		p50 := percentileOf(durations, 0.50)
		p95 := percentileOf(durations, 0.95)
		t.Logf("%d words over %d runs: p50=%.0fms p95=%.0fms", size.words, size.runs, p50, p95)

		if size.words == 5000 && p95 >= 5000 {
			t.Errorf("p95 latency for 5000 words = %.0fms; PRD target is <5000ms", p95)
		}
		report.Corpora = append(report.Corpora, latencySize{
			Words: size.words, Runs: size.runs, P50ms: math.Round(p50), P95ms: math.Round(p95),
		})
	}

	outDir := "../testresults/benchmarks"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	out, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(outDir+"/latency.json", out, 0o644); err != nil {
		t.Fatalf("write latency report: %v", err)
	}
}
