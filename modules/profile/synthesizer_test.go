package profile

import (
	"testing"

	"psycho/modules/analyze"
)

func TestComputeConfidenceFlag(t *testing.T) {
	if got := computeConfidenceFlag(400, 0.8); got != "low" {
		t.Errorf("computeConfidenceFlag(400, 0.8) = %q; want low", got)
	}
	if got := computeConfidenceFlag(600, 0.5); got != "medium" {
		t.Errorf("computeConfidenceFlag(600, 0.5) = %q; want medium", got)
	}
	if got := computeConfidenceFlag(600, 0.8); got != "medium" {
		t.Errorf("computeConfidenceFlag(600, 0.8) = %q; want medium", got)
	}
	if got := computeConfidenceFlag(1500, 0.8); got != "high" {
		t.Errorf("computeConfidenceFlag(1500, 0.8) = %q; want high", got)
	}
}

func TestComputeCIWidth(t *testing.T) {
	w := computeCIWidth(100, 0.8)
	if w <= 0 || w > 0.3 {
		t.Errorf("CI width = %f; want between 0 and 0.3", w)
	}
	// More words -> narrower CI
	w2 := computeCIWidth(10000, 0.8)
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
	if openness.Percentile != 98 {
		t.Errorf("Openness percentile = %d; want 98 (z=%.3f)", openness.Percentile, (0.75-0.50)/0.12)
	}
	if len(openness.ConfidenceInterval) != 2 {
		t.Errorf("Openness CI length = %d; want 2", len(openness.ConfidenceInterval))
	}

	if profile.Summary == (analyze.SummaryVariables{}) {
		t.Error("SummaryVariables is zero; expected computed values")
	}
}
