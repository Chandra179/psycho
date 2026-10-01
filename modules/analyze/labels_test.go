package analyze

import "testing"

func TestDimensionDisplayName(t *testing.T) {
	cases := []struct{ key, want string }{
		{"openness", "Openness"},
		{"conscientiousness", "Conscientiousness"},
		{"extraversion", "Extraversion"},
		{"agreeableness", "Agreeableness"},
		{"neuroticism", "Neuroticism"},
		{"regulatory_focus", "Regulatory Focus"},
		{"need_for_cognition", "Need for Cognition"},
		{"cognitive_style", "Cognitive Style"},
		{"need_for_closure", "Need for Closure"},
		{"unknown", "unknown"},
	}
	for _, c := range cases {
		if got := DimensionDisplayName(c.key); got != c.want {
			t.Errorf("DimensionDisplayName(%q) = %q; want %q", c.key, got, c.want)
		}
	}
}

func TestDimensionLabel(t *testing.T) {
	cases := []struct {
		key   string
		score float64
		want  string
	}{
		{"openness", 0.70, "high"},
		{"openness", 0.30, "low"},
		{"openness", 0.50, "moderate"},
		// The four dimensions with dedicated scales route to their own labelers.
		{"regulatory_focus", 0.70, "promotion_focus"},
		{"regulatory_focus", 0.30, "prevention_focus"},
		{"need_for_cognition", 0.70, "high"},
		{"cognitive_style", 0.70, "systematic"},
		{"need_for_closure", 0.70, "high"},
	}
	for _, c := range cases {
		if got := DimensionLabel(c.key, c.score); got != c.want {
			t.Errorf("DimensionLabel(%q, %.2f) = %q; want %q", c.key, c.score, got, c.want)
		}
	}
}

func TestHighModerateLowBoundaries(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{0.65, "high"},
		{0.6499, "moderate"},
		{0.50, "moderate"},
		{0.35, "moderate"},
		{0.3499, "low"},
	}
	for _, c := range cases {
		if got := HighModerateLow(c.score); got != c.want {
			t.Errorf("HighModerateLow(%.4f) = %q; want %q", c.score, got, c.want)
		}
	}
}
