package analyze

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"psycho/modules/ingest"
)

func extractText(t *testing.T, dictJSON, text string) FeatureVector {
	t.Helper()
	dict, err := LoadDictionaryFromJSON([]byte(dictJSON))
	if err != nil {
		t.Fatal(err)
	}
	fv, _ := NewFeatureExtractor(dict).Extract(ingest.Document{RawText: text})
	return fv
}

func TestLongWordsAreCountedInLettersNotBytes(t *testing.T) {
	// "résumés" is 7 letters but 9 bytes; "café" is 4 letters but 5 bytes;
	// "don't" is 4 letters (apostrophe excluded) but 5 bytes.
	fv := extractText(t, `{"certainty":["x"]}`, "résumés café don't extraordinary")
	if fv.BigWordCount != 2 {
		t.Fatalf("BigWordCount = %d; want 2 (résumés, extraordinary)", fv.BigWordCount)
	}
	if got, want := fv.AvgWordLength, (7.0+4+4+13)/4; math.Abs(got-want) > 1e-9 {
		t.Fatalf("AvgWordLength = %f; want %f", got, want)
	}
}

func TestEmotionalToneFlipsNegatedEmotionWords(t *testing.T) {
	dict := `{"positive_emotion":["happy","good"],"negative_emotion":["sad","bad"]}`
	plain := extractText(t, dict, "i am happy and it was good")
	flipped := extractText(t, dict, "i am not happy and it was not good")
	if flipped.NegatedPositive != 2 || plain.NegatedPositive != 0 {
		t.Fatalf("negated positives: plain=%d flipped=%d", plain.NegatedPositive, flipped.NegatedPositive)
	}
	a, _ := ComputeSummaryVariablesWithDetails(plain)
	b, _ := ComputeSummaryVariablesWithDetails(flipped)
	if a.EmotionalTone <= 0.5 || b.EmotionalTone >= 0.5 {
		t.Fatalf("tone must flip with negation: plain=%.2f negated=%.2f", a.EmotionalTone, b.EmotionalTone)
	}
	notBad := extractText(t, dict, "it was not bad and not sad")
	if c, _ := ComputeSummaryVariablesWithDetails(notBad); c.EmotionalTone <= 0.5 {
		t.Fatalf("'not bad' should read positive, got %.2f", c.EmotionalTone)
	}
}

func TestSummaryToneBandsAreTighterThanTheGenericBands(t *testing.T) {
	for score, want := range map[float64]string{0.44: "negative", 0.5: "neutral", 0.56: "positive", 0.45: "negative", 0.55: "positive"} {
		if got := SummaryTone(score); got != want {
			t.Errorf("SummaryTone(%.2f) = %q, want %q", score, got, want)
		}
	}
}

func TestScoreStandardErrorIsPerMeasureAndShrinksWithLength(t *testing.T) {
	dict, err := os.ReadFile("dictionary.json")
	if err != nil {
		t.Fatal(err)
	}
	short := extractText(t, string(dict), strings.Repeat("I was not sure but we built the first plan with care. ", 20))
	long := extractText(t, string(dict), strings.Repeat("I was not sure but we built the first plan with care. ", 80))
	for _, dim := range dimensionKeys {
		if short.ScoreSE[dim] == 0 {
			continue // no words in this measure's categories: no spread to report
		}
		if ratio := short.ScoreSE[dim] / long.ScoreSE[dim]; math.Abs(ratio-2) > 0.05 {
			t.Errorf("%s: SE should halve when length quadruples, ratio = %.3f", dim, ratio)
		}
	}
	if short.ScoreSE["openness"] == short.ScoreSE["need_for_cognition"] {
		t.Error("measures with different weights must not share one standard error")
	}
}

func TestStandardErrorMatchesDirectScoreVariance(t *testing.T) {
	// For a one-category dimension the score is baseline + w*100*share, where
	// share is a proportion: SE = w*100*sqrt(p(1-p)/n).
	fv := extractText(t, `{"certainty":["sure"],"tentative":["maybe"]}`, strings.Repeat("sure maybe other other ", 50))
	n, p := float64(fv.WordCount), 0.25
	want := 0.020 * 100 * math.Sqrt(p*(1-p)/n) // need_for_closure certainty weight
	// tentative (-0.025) shares the denominator; account for both categories.
	x := []float64{2.0, -2.5, 0, 0}
	mean := 0.0
	for _, v := range x {
		mean += v / 4
	}
	variance := 0.0
	for _, v := range x {
		variance += (v - mean) * (v - mean) / 4
	}
	want = math.Sqrt(variance/n) * seInflation["need_for_closure"]
	if got := fv.ScoreSE["need_for_closure"]; math.Abs(got-want) > 1e-9 {
		t.Fatalf("need_for_closure SE = %.6f; want %.6f", got, want)
	}
}

func TestCalibrationQuantilesKeepSubRoundingResolution(t *testing.T) {
	var samples []BigFiveScores
	for i := 0; i < 400; i++ {
		s := BigFiveScores{Calculations: map[string]*ScoreCalculation{}}
		v := 0.50 + float64(i)*0.00005 // spans 0.02: only ~3 distinct 2-decimal scores
		for _, dim := range dimensionKeys {
			s.Calculations[dim] = &ScoreCalculation{ClampedScore: v, ModelScore: clamp(v)}
			setDimensionValue(dim, &s, clamp(v))
		}
		samples = append(samples, s)
	}
	cal, err := BuildCalibration("test", "now", samples)
	if err != nil {
		t.Fatal(err)
	}
	distinct := map[float64]bool{}
	for _, q := range cal.Dimensions["openness"].Quantiles {
		distinct[q] = true
	}
	if len(distinct) < 90 {
		t.Fatalf("quantiles collapse to %d distinct values; unrounded scores should keep ~99", len(distinct))
	}
	// Two scores that display identically still get different ranks.
	lo, _ := cal.Percentile("openness", 0.4951)
	hi, _ := cal.Percentile("openness", 0.5049)
	if lo >= hi {
		t.Fatalf("percentiles of 0.4951 and 0.5049 should differ: %d, %d", lo, hi)
	}
}

func TestAdjustScoresKeepsUnroundedValueForRanking(t *testing.T) {
	s := BigFiveScores{Calculations: map[string]*ScoreCalculation{}}
	for _, dim := range dimensionKeys {
		s.Calculations[dim] = &ScoreCalculation{ClampedScore: 0.5123, ModelScore: 0.51, FinalScore: 0.51}
		setDimensionValue(dim, &s, 0.51)
	}
	cal := &Calibration{Dimensions: map[string]DimensionCalibration{"openness": {Offset: 0.01}}}
	cal.AdjustScores(&s)
	if s.Openness != 0.52 {
		t.Fatalf("displayed score = %v; want 0.52", s.Openness)
	}
	if got := UnroundedCalibratedScore("openness", &s); math.Abs(got-0.5223) > 1e-9 {
		t.Fatalf("unrounded calibrated score = %v; want 0.5223", got)
	}
}

func TestValueListsExcludePolysemousWords(t *testing.T) {
	raw, err := os.ReadFile("dictionary.json")
	if err != nil {
		t.Fatal(err)
	}
	var d map[string][]string
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	banned := []string{"just", "kind", "content", "natural", "care", "support", "rule", "rules", "independent", "certain", "order", "world", "control", "play", "party", "follow", "values"}
	for cat, words := range d {
		if !strings.HasPrefix(cat, "value_") {
			continue
		}
		for _, w := range words {
			for _, b := range banned {
				if w == b {
					t.Errorf("%s still lists the everyday word %q", cat, w)
				}
			}
		}
	}
	for _, w := range []string{"cried", "scam", "garbage", "disaster", "scream"} {
		found := false
		for _, v := range d["negative_emotion"] {
			found = found || v == w
		}
		if !found {
			t.Errorf("negative_emotion should list %q", w)
		}
	}
}

func TestNegatedValueWordsAreSkippedButContractionsCount(t *testing.T) {
	dict := `{"value_benevolence":["generous"]}`
	for text, want := range map[string]int{
		"he was generous":                1,
		"he wasn't generous":             0,
		"he wasn’t generous":             0,
		"nothing about him was generous": 1, // "nothing" is four tokens back: outside the window
		"he was never really generous":   0,
	} {
		if got := extractText(t, dict, text).CategoryCounts["value_benevolence"]; got != want {
			t.Errorf("%q: count = %d, want %d", text, got, want)
		}
	}
}

func TestDictionaryHonoursExclusions(t *testing.T) {
	var dict map[string][]string
	raw, err := os.ReadFile("dictionary.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatal(err)
	}
	var rec struct {
		Excluded    map[string][]struct{ Word, Rule string } `json:"excluded"`
		AuditedKept map[string][]string                      `json:"audited_kept"`
	}
	raw, err = os.ReadFile("dictionary_exclusions.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Excluded) == 0 {
		t.Fatal("no exclusions recorded")
	}
	has := func(cat, w string) bool {
		for _, v := range dict[cat] {
			if v == w {
				return true
			}
		}
		return false
	}
	for cat, list := range rec.Excluded {
		for _, e := range list {
			if has(cat, e.Word) {
				t.Errorf("%s still lists %q, which %s excluded", cat, e.Word, e.Rule)
			}
		}
	}
	for cat, words := range rec.AuditedKept {
		for _, w := range words {
			if !has(cat, w) {
				t.Errorf("%s lost %q, which the audit kept", cat, w)
			}
		}
	}
}
