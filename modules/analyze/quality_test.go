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
	}
	for _, c := range cases {
		if got := QualityFlag(c.words, c.coverage); got != c.want {
			t.Errorf("QualityFlag(%d, %.1f) = %q, want %q", c.words, c.coverage, got, c.want)
		}
	}
}

func TestQualityReasonsExplainEveryLimitingFactor(t *testing.T) {
	if got := QualityReasons(1500, 0.8); got != nil {
		t.Fatalf("no limits expected, got %v", got)
	}
	short := QualityReasons(300, 0.8)
	if len(short) != 1 || !strings.Contains(short[0], "short") {
		t.Fatalf("short text reason = %v", short)
	}
	both := QualityReasons(800, 0.41)
	if len(both) != 2 || !strings.Contains(both[1], "matched the dictionary") {
		t.Fatalf("medium text with low coverage reasons = %v", both)
	}
	// Every non-high flag must come with at least one reason.
	for _, c := range []struct {
		words    int
		coverage float64
	}{{100, 0.9}, {700, 0.9}, {2000, 0.4}} {
		if QualityFlag(c.words, c.coverage) != "high" && len(QualityReasons(c.words, c.coverage)) == 0 {
			t.Errorf("flag %q for %d words at %.1f coverage has no reason", QualityFlag(c.words, c.coverage), c.words, c.coverage)
		}
	}
}

func TestFormatCount(t *testing.T) {
	for in, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 10785: "10,785", 1234567: "1,234,567"} {
		if got := FormatCount(in); got != want {
			t.Errorf("FormatCount(%d) = %q, want %q", in, got, want)
		}
	}
	if got := QualityReasons(800, 0.8); len(got) != 1 || !strings.Contains(got[0], "1,000 words") {
		t.Errorf("reason must use a grouped count: %v", got)
	}
}
