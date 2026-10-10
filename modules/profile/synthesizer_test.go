package profile

import (
	"math"
	"testing"

	"psycho/modules/analyze"
)

func TestRangeAndNormalPercentileCalculationsReplay(t *testing.T) {
	for _, n := range []int{10, 600, 1200, 10000, 1000000} {
		for _, coverage := range []float64{.2, .5, .7, 1} {
			width, r := computeRangeWithDetails(n, coverage)
			raw := r.ErrorMultiplier * (r.BaseStandardError / math.Sqrt(float64(r.WordCount)/1000)) * r.CoverageMultiplier
			if raw != r.UnboundedHalfWidth || width != math.Round(max(r.MinimumHalfWidth, min(r.MaximumHalfWidth, raw))*100)/100 {
				t.Fatal("range cannot replay")
			}
			for _, score := range []float64{0, .01, .35, .50, .65, .99, 1} {
				pct, p := normalPercentileWithDetails(score)
				z := (p.Score - p.NormalApproximation.Mean) / p.NormalApproximation.SD
				expected := max(1, min(99, int(math.Round(.5*(1+math.Erf(z/math.Sqrt2))*100))))
				if pct != expected || p.NormalApproximation.Z != z {
					t.Fatal("fallback percentile cannot replay")
				}
				trait, b := makeTraitResultWithDetails(score, width, pct, nil)
				if b.Low != math.Round(max(0, score-width)*100)/100 || b.High != math.Round(min(1, score+width)*100)/100 || trait.ConfidenceInterval[0] != b.Low || trait.ConfidenceInterval[1] != b.High {
					t.Fatal("bounds cannot replay")
				}
			}
		}
	}
}

func TestComputeConfidenceFlag(t *testing.T) {
	if got := computeConfidenceFlag(400, 0.8, 0); got != "low" {
		t.Errorf("computeConfidenceFlag(400, 0.8, 0) = %q; want low", got)
	}
	if got := computeConfidenceFlag(600, 0.5, 0); got != "medium" {
		t.Errorf("computeConfidenceFlag(600, 0.5, 0) = %q; want medium", got)
	}
	if got := computeConfidenceFlag(600, 0.8, 0); got != "medium" {
		t.Errorf("computeConfidenceFlag(600, 0.8, 0) = %q; want medium", got)
	}
	if got := computeConfidenceFlag(1500, 0.8, 0); got != "high" {
		t.Errorf("computeConfidenceFlag(1500, 0.8, 0) = %q; want high", got)
	}
}

func TestComputeCIWidth(t *testing.T) {
	w, _ := computeRangeWithDetails(100, 0.8)
	if w <= 0 || w > 0.3 {
		t.Errorf("CI width = %f; want between 0 and 0.3", w)
	}
	// More words -> narrower CI
	w2, _ := computeRangeWithDetails(10000, 0.8)
	if w2 >= w {
		t.Errorf("CI width for 10000 words (%f) should be narrower than for 100 words (%f)", w2, w)
	}
}

func TestScoreAggregator(t *testing.T) {
	sa := NewScoreAggregator()
	scores := analyze.BigFiveScores{
		Openness:          0.75,
		Conscientiousness: 0.60,
		Extraversion:      0.45,
		Agreeableness:     0.55,
		Neuroticism:       0.30,
	}
	profile := sa.Aggregate(scores, analyze.FeatureVector{}, 1200, 0.75)

	if profile.AnalysisID == "" {
		t.Error("AnalysisID is empty")
	}
	if profile.ConfidenceFlag != "high" {
		t.Errorf("ConfidenceFlag = %q; want high", profile.ConfidenceFlag)
	}
	if len(profile.Traits) != 9 {
		t.Errorf("len(Traits) = %d; want 9", len(profile.Traits))
	}

	openness := profile.Traits["openness"]
	if openness.Score != 0.75 {
		t.Errorf("Openness score = %f; want 0.75", openness.Score)
	}
	if openness.Percentile != 95 {
		t.Errorf("Openness percentile = %d; want 95 (z=%.3f)", openness.Percentile, (0.75-0.50)/0.15)
	}
	if len(openness.ConfidenceInterval) != 2 {
		t.Errorf("Openness CI length = %d; want 2", len(openness.ConfidenceInterval))
	}

	if profile.Summary == (analyze.SummaryVariables{}) {
		t.Error("SummaryVariables is zero; expected computed values")
	}
}

func TestBoundsUsePerMeasureSamplingError(t *testing.T) {
	scores := analyze.BigFiveScores{Openness: 0.5, NeedForCognition: 0.5}
	fv := analyze.FeatureVector{WordCount: 1000, ScoreSE: map[string]float64{"openness": 0.05, "need_for_cognition": 0.01}}
	p := NewScoreAggregator().Aggregate(scores, fv, 1000, 0.7)
	open, nfc := p.Traits["openness"].ConfidenceInterval, p.Traits["need_for_cognition"].ConfidenceInterval
	if width := open[1] - open[0]; math.Abs(width-0.20) > 1e-9 {
		t.Errorf("openness half-width should be 1.96*0.05 = 0.10, got interval %v", open)
	}
	if width := nfc[1] - nfc[0]; math.Abs(width-0.04) > 1e-9 {
		t.Errorf("need_for_cognition half-width should be 1.96*0.01 = 0.02, got interval %v", nfc)
	}
	if m := p.CalculationDetails.RangeBounds["openness"].Method; m != "per_measure_sampling_error" {
		t.Errorf("method = %q", m)
	}
	// Without recorded spread, fall back to the shared length rule.
	legacy := NewScoreAggregator().Aggregate(scores, analyze.FeatureVector{WordCount: 1000}, 1000, 0.7)
	if m := legacy.CalculationDetails.RangeBounds["openness"].Method; m != "shared_length_rule" {
		t.Errorf("fallback method = %q", m)
	}
}
