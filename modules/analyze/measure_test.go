package analyze

import (
	"strings"
	"testing"
)

func TestMeasureSummaryCoversEveryMeasureWithoutEmDash(t *testing.T) {
	keys := []string{
		"openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism",
		"regulatory_focus", "need_for_cognition", "cognitive_style", "need_for_closure",
		"analytical_thinking", "clout", "authenticity", "emotional_tone",
	}
	for _, k := range keys {
		s := MeasureSummary(k)
		if s == "" {
			t.Errorf("MeasureSummary(%q) is empty", k)
		}
		if strings.ContainsRune(s, '—') {
			t.Errorf("MeasureSummary(%q) contains an em-dash: %q", k, s)
		}
	}
	if !strings.Contains(MeasureSummary("authenticity"), "not dishonest") {
		t.Error("Authenticity must say low means formal, not dishonest")
	}
	if MeasureSummary("unknown") != "" {
		t.Error("unknown keys have no summary")
	}
}

func TestCategoryLabel(t *testing.T) {
	for in, want := range map[string]string{
		"past_focus": "past focus",
		"article":    "article",
	} {
		if got := CategoryLabel(in); got != want {
			t.Errorf("CategoryLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDedicatedLabelersShareTheBoundaryRule(t *testing.T) {
	// Exactly 0.65 is the top band and exactly 0.35 is the middle band for every measure,
	// matching the legend ("35-64", "65-100").
	cases := []struct {
		key            string
		high, mid, low string
	}{
		{"openness", "high", "moderate", "low"},
		{"need_for_cognition", "high", "moderate", "low"},
		{"need_for_closure", "high", "moderate", "low"},
		{"regulatory_focus", "promotion_focus", "balanced", "prevention_focus"},
		{"cognitive_style", "systematic", "mixed", "intuitive"},
	}
	for _, c := range cases {
		if got := DimensionLabel(c.key, 0.65); got != c.high {
			t.Errorf("%s at 0.65 = %q, want %q", c.key, got, c.high)
		}
		if got := DimensionLabel(c.key, 0.6499); got != c.mid {
			t.Errorf("%s at 0.6499 = %q, want %q", c.key, got, c.mid)
		}
		if got := DimensionLabel(c.key, 0.35); got != c.mid {
			t.Errorf("%s at 0.35 = %q, want %q", c.key, got, c.mid)
		}
		if got := DimensionLabel(c.key, 0.3499); got != c.low {
			t.Errorf("%s at 0.3499 = %q, want %q", c.key, got, c.low)
		}
	}
}
