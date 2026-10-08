package analyze

import (
	"strings"
	"testing"

	"psycho/modules/ingest"
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
		"past_focus":      "past focus",
		"article":         "article",
		"long_word_ratio": "long words (over six bytes)",
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

func TestPercentileMainLine(t *testing.T) {
	emp := &ingest.PercentileReference{Method: ingest.PercentileMethodEmpirical}
	norm := &ingest.PercentileReference{Method: ingest.PercentileMethodNormalApproximation}

	if got := PercentileMainLine(94, emp, "high"); got != "Higher than about 94 of 100 reference texts." {
		t.Fatalf("plain rank line = %q", got)
	}
	if strings.Contains(PercentileMainLine(94, emp, "high"), "label") {
		t.Fatal("an aligned label needs no clustering note")
	}
	for _, c := range []struct {
		p    int
		want string
	}{{99, "ranks high"}, {80, "ranks high"}, {1, "ranks low"}, {20, "ranks low"}} {
		got := PercentileMainLine(c.p, emp, "moderate")
		if !strings.Contains(got, c.want) || !strings.Contains(got, "cluster tightly") {
			t.Errorf("moderate at rank %d = %q, want clustering note containing %q", c.p, got, c.want)
		}
	}
	if got := PercentileMainLine(50, emp, "moderate"); strings.Contains(got, "cluster") {
		t.Fatalf("a mid-range rank needs no note: %q", got)
	}
	for _, label := range []string{"balanced", "mixed"} {
		if !strings.Contains(PercentileMainLine(95, emp, label), strings.ToUpper(label[:1])+label[1:]+" on the 0–100 scale") {
			t.Errorf("the note must name the %q label shown on the card", label)
		}
	}
	// Only an empirical rank is shown as a rank.
	for _, ref := range []*ingest.PercentileReference{nil, norm, {}} {
		if got := PercentileMainLine(94, ref, "high"); got != "" {
			t.Errorf("non-empirical reference must give no rank line, got %q", got)
		}
	}
	if PercentileMainLine(0, emp, "low") != "" || PercentileMainLine(100, emp, "high") != "" {
		t.Fatal("out-of-range percentiles are not shown")
	}
	if strings.ContainsRune(PercentileMainLine(99, emp, "moderate"), '—') {
		t.Fatal("copy must not contain em-dashes")
	}
}
