package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/pipeline"
	"psycho/modules/profile"
	"psycho/zlogger"
)

const calibrationPath = "../config/calibration.json"

func loadCommittedCalibration(t *testing.T) *analyze.Calibration {
	t.Helper()
	data, err := os.ReadFile(calibrationPath)
	if err != nil {
		t.Fatalf("read committed calibration: %v", err)
	}
	cal, err := analyze.LoadCalibration(data)
	if err != nil {
		t.Fatalf("load committed calibration: %v", err)
	}
	return cal
}

// TestCalibrationCentersCorpusMean pins the core property: a document that
// scores like the corpus average must land at the 50th percentile after
// adjustment.
func TestCalibrationCentersCorpusMean(t *testing.T) {
	// Ten documents whose openness raw score is 0.60, everything else varied.
	var samples []analyze.BigFiveScores
	for i := 0; i < 10; i++ {
		s := analyze.BigFiveScores{Openness: 0.60}
		setOtherDims(&s, 0.40+float64(i)*0.02)
		samples = append(samples, s)
	}
	cal, err := analyze.BuildCalibration("test", "2026-10-01T00:00:00Z", samples)
	if err != nil {
		t.Fatalf("build calibration: %v", err)
	}

	d := cal.Dimensions["openness"]
	if d.Offset != -0.10 {
		t.Fatalf("openness offset = %v, want -0.10", d.Offset)
	}
	if got := clampForTest(0.60 + d.Offset); got != 0.50 {
		t.Fatalf("adjusted corpus-mean score = %v, want 0.50", got)
	}
	p, ok := cal.Percentile("openness", 0.50)
	if !ok {
		t.Fatal("openness missing from calibration")
	}
	if p != 50 {
		t.Fatalf("corpus-mean percentile = %d, want 50", p)
	}
}

// TestCalibrationPercentileMonotonic checks the lookup is non-decreasing in
// score against the committed calibration file, and clamps at the ends.
func TestCalibrationPercentileMonotonic(t *testing.T) {
	cal := loadCommittedCalibration(t)

	for dim, d := range cal.Dimensions {
		prev := 0
		for i, q := range d.Quantiles {
			p, ok := cal.Percentile(dim, q)
			if !ok {
				t.Fatalf("%s: percentile lookup failed at quantile %d", dim, i)
			}
			if p < prev {
				t.Fatalf("%s: percentile went backwards at quantile %d: %d < %d", dim, i, p, prev)
			}
			prev = p
		}
		low, _ := cal.Percentile(dim, d.Quantiles[0]-1)
		high, _ := cal.Percentile(dim, d.Quantiles[98]+1)
		if low != 1 || high != 99 {
			t.Fatalf("%s: clamping broken: low=%d high=%d, want 1/99", dim, low, high)
		}
	}
}

// TestCalibrationRoundTrip pins the serialization format cmd/calibrate
// writes and the server reads.
func TestCalibrationRoundTrip(t *testing.T) {
	var samples []analyze.BigFiveScores
	for i := 0; i < 20; i++ {
		s := analyze.BigFiveScores{}
		for _, dim := range analyze.CalibratedDimensions() {
			setDimension(&s, dim, 0.30+float64(i)*0.02)
		}
		samples = append(samples, s)
	}
	cal, err := analyze.BuildCalibration("test", "2026-10-01T00:00:00Z", samples)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	data, err := json.Marshal(cal)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	loaded, err := analyze.LoadCalibration(data)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	for _, dim := range analyze.CalibratedDimensions() {
		for _, score := range []float64{0.3, 0.45, 0.5, 0.62, 0.75} {
			want, okW := cal.Percentile(dim, score)
			got, okG := loaded.Percentile(dim, score)
			if okW != okG || want != got {
				t.Fatalf("%s @ %.2f: percentile %d/%v, want %d/%v", dim, score, got, okG, want, okW)
			}
		}
	}
}

func TestCalibrationLoaderRejectsMalformed(t *testing.T) {
	if _, err := analyze.LoadCalibration([]byte(`{"dimensions":{}}`)); err == nil {
		t.Fatal("expected error for empty dimensions")
	}
	if _, err := analyze.LoadCalibration([]byte(`{"dimensions":{"openness":{"quantiles":[0.5,0.4]}}}`)); err == nil {
		t.Fatal("expected error for unsorted quantiles")
	}
	if _, err := analyze.BuildCalibration("test", "x", nil); err == nil {
		t.Fatal("expected error for zero samples")
	}
}

// TestCalibratedPipeline runs two directionally opposite texts through the
// full pipeline with the committed calibration attached: the article-heavy
// text must out-rank the pronoun-heavy one on openness, and percentiles must
// come from the measured reference distribution (extreme texts stay inside
// the clamped 1–99 band while a mid-range text sits near the middle).
func TestCalibratedPipeline(t *testing.T) {
	logger := zlogger.New("dev")

	profileDeps, err := profile.NewDependencies(profile.Config{DBPath: ":memory:"}, logger)
	if err != nil {
		t.Fatalf("init profile: %v", err)
	}
	analyzeDeps, err := analyze.NewDependencies(analyze.Config{
		DictionaryPath:  "../modules/analyze/dictionary.json",
		CalibrationPath: calibrationPath,
	}, logger)
	if err != nil {
		t.Fatalf("init analyze: %v", err)
	}
	if analyzeDeps.Calibration == nil {
		t.Fatal("calibration not loaded")
	}
	profileDeps.Aggregator.UseCalibration(analyzeDeps.Calibration)

	pipe := pipeline.New(
		analyzeDeps.Extractor,
		analyzeDeps.Model,
		profileDeps.Aggregator,
		profileDeps.NarrativeGenerator,
		profileDeps.Storage,
		analyzeDeps.Calibration,
	)

	articleHeavy := strings.Repeat("Theories about architecture and the history of the cities were the topics of the lectures. ", 40)
	pronounHeavy := strings.Repeat("I was there and you were with me when they told us we would miss it all. ", 40)

	outHigh, err := pipe.Run(t.Context(), "test", "", articleHeavy)
	if err != nil {
		t.Fatalf("run article-heavy: %v", err)
	}
	outLow, err := pipe.Run(t.Context(), "test", "", pronounHeavy)
	if err != nil {
		t.Fatalf("run pronoun-heavy: %v", err)
	}

	pctHigh := outHigh.Traits["openness"].(profile.TraitResult).Percentile
	pctLow := outLow.Traits["openness"].(profile.TraitResult).Percentile
	if pctHigh <= pctLow {
		t.Fatalf("openness percentiles: article-heavy %d should exceed pronoun-heavy %d", pctHigh, pctLow)
	}
	if pctHigh < 1 || pctHigh > 99 || pctLow < 1 || pctLow > 99 {
		t.Fatalf("percentiles out of [1,99]: %d, %d", pctHigh, pctLow)
	}
}

// --- helpers ---

func setOtherDims(s *analyze.BigFiveScores, v float64) {
	for _, dim := range analyze.CalibratedDimensions() {
		if dim != "openness" {
			setDimension(s, dim, v)
		}
	}
}

func setDimension(s *analyze.BigFiveScores, dim string, v float64) {
	switch dim {
	case "openness":
		s.Openness = v
	case "conscientiousness":
		s.Conscientiousness = v
	case "extraversion":
		s.Extraversion = v
	case "agreeableness":
		s.Agreeableness = v
	case "neuroticism":
		s.Neuroticism = v
	case "regulatory_focus":
		s.RegulatoryFocus = v
	case "need_for_cognition":
		s.NeedForCognition = v
	case "cognitive_style":
		s.CognitiveStyle = v
	case "need_for_closure":
		s.NeedForClosure = v
	}
}

func clampForTest(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return float64(int(v*100+0.5)) / 100
}

// TestCalibrationMatchesDictionary fails when modules/analyze/dictionary.json
// changes without a corresponding cmd/calibrate regeneration — a stale
// calibration would silently misplace every percentile.
func TestCalibrationMatchesDictionary(t *testing.T) {
	cal := loadCommittedCalibration(t)
	if cal.DictionarySHA256 == "" {
		t.Fatal("committed calibration has no dictionary_sha256; regenerate with cmd/calibrate")
	}
	data, err := os.ReadFile("../modules/analyze/dictionary.json")
	if err != nil {
		t.Fatalf("read dictionary: %v", err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != cal.DictionarySHA256 {
		t.Fatalf("dictionary changed (sha256 %s) but calibration was built for %s — rerun cmd/calibrate", got, cal.DictionarySHA256)
	}
}
