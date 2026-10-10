package analyze

import (
	"strings"
	"testing"
)

func TestQualityFlagMatchesThresholds(t *testing.T) {
	cases := []struct {
		words    int
		coverage float64
		want     string
	}{
		{400, 0.8, "low"},
		{600, 0.5, "medium"},
		{600, 0.8, "medium"},
		{1500, 0.8, "high"},
		{1500, 0.5, "medium"},
		{1500, 0.4, "low"},
		{1500, 0.45, "medium"},
	}
	for _, c := range cases {
		if got := QualityFlagWithNoise(c.words, c.coverage, 0); got != c.want {
			t.Errorf("QualityFlag(%d, %.1f) = %q, want %q", c.words, c.coverage, got, c.want)
		}
	}
}

func TestQualityReasonsExplainEveryLimitingFactor(t *testing.T) {
	if got := QualityReasonsWithNoise(1500, 0.8, 0); got != nil {
		t.Fatalf("no limits expected, got %v", got)
	}
	short := QualityReasonsWithNoise(300, 0.8, 0)
	if len(short) != 1 || !strings.Contains(short[0], "short") {
		t.Fatalf("short text reason = %v", short)
	}
	both := QualityReasonsWithNoise(800, 0.5, 0)
	if len(both) != 2 || !strings.Contains(both[1], "Under 60%") {
		t.Fatalf("medium text with low coverage reasons = %v", both)
	}
	veryLow := QualityReasonsWithNoise(2000, 0.4, 0)
	if len(veryLow) != 1 || !strings.Contains(veryLow[0], "Under 45%") {
		t.Fatalf("very low coverage reasons = %v", veryLow)
	}
	// Every non-high flag must come with at least one reason.
	for _, c := range []struct {
		words    int
		coverage float64
	}{{100, 0.9}, {700, 0.9}, {2000, 0.4}} {
		if QualityFlagWithNoise(c.words, c.coverage, 0) != "high" && len(QualityReasonsWithNoise(c.words, c.coverage, 0)) == 0 {
			t.Errorf("flag %q for %d words at %.1f coverage has no reason", QualityFlagWithNoise(c.words, c.coverage, 0), c.words, c.coverage)
		}
	}
}

func TestFormatCount(t *testing.T) {
	for in, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 10785: "10,785", 1234567: "1,234,567"} {
		if got := FormatCount(in); got != want {
			t.Errorf("FormatCount(%d) = %q, want %q", in, got, want)
		}
	}
	if got := QualityReasonsWithNoise(800, 0.8, 0); len(got) != 1 || !strings.Contains(got[0], "1,000 words") {
		t.Errorf("reason must use a grouped count: %v", got)
	}
}

func TestNoiseShareCapsReadingQualityAtMedium(t *testing.T) {
	if got := QualityFlagWithNoise(2000, 0.8, 0); got != "high" {
		t.Fatalf("clean long text = %q, want high", got)
	}
	if got := QualityFlagWithNoise(2000, 0.8, QualityNoisyShare); got != "medium" {
		t.Fatalf("noisy long text = %q, want medium", got)
	}
	if got := QualityFlagWithNoise(300, 0.8, 0.5); got != "low" {
		t.Fatalf("noise must not raise a low flag, got %q", got)
	}
	reasons := QualityReasonsWithNoise(2000, 0.8, 0.063)
	if len(reasons) != 1 || !strings.Contains(reasons[0], "6%") || strings.ContainsRune(reasons[0], '—') {
		t.Fatalf("noise reason = %v", reasons)
	}
	if QualityReasonsWithNoise(2000, 0.8, 0.01) != nil {
		t.Fatal("a small noise share must not add a reason")
	}
}
