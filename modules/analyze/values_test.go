package analyze

import (
	"testing"
)

func TestComputeSchwartzValues_ReturnsAllTen(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"value_self_direction": 2.5,
			"value_stimulation":    1.2,
			"value_hedonism":       0.8,
			"value_achievement":    3.1,
			"value_power":          1.5,
			"value_security":       2.0,
			"value_conformity":     0.9,
			"value_tradition":      1.8,
			"value_benevolence":    2.2,
			"value_universalism":   1.6,
		},
	}

	values := ComputeSchwartzValues(fv)
	if len(values) != 10 {
		t.Errorf("len(values) = %d; want 10", len(values))
	}

	for _, key := range []string{
		"value_self_direction", "value_stimulation", "value_hedonism",
		"value_achievement", "value_power", "value_security",
		"value_conformity", "value_tradition", "value_benevolence",
		"value_universalism",
	} {
		if _, ok := values[key]; !ok {
			t.Errorf("missing value key %q", key)
		}
	}
}

func TestComputeSchwartzValues_MissingCategories(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{},
	}

	values := ComputeSchwartzValues(fv)
	if len(values) != 10 {
		t.Errorf("len(values) = %d; want 10 even with empty input", len(values))
	}
	for k, v := range values {
		if v != 0 {
			t.Errorf("value %s = %f; want 0 for missing category", k, v)
		}
	}
}

func TestComputeSchwartzValues_PreservesPercentages(t *testing.T) {
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{
			"value_achievement": 4.5,
		},
	}

	values := ComputeSchwartzValues(fv)
	if values["value_achievement"] != 4.5 {
		t.Errorf("value_achievement = %f; want 4.5", values["value_achievement"])
	}
}

func TestValueDisplayName(t *testing.T) {
	cases := []struct {
		cat  Category
		want string
	}{
		{"value_self_direction", "Self-direction"},
		{"value_stimulation", "Stimulation"},
		{"value_hedonism", "Hedonism"},
		{"value_achievement", "Achievement"},
		{"value_power", "Power"},
		{"value_security", "Security"},
		{"value_conformity", "Conformity"},
		{"value_tradition", "Tradition"},
		{"value_benevolence", "Benevolence"},
		{"value_universalism", "Universalism"},
		{"unknown", "unknown"},
	}
	for _, c := range cases {
		got := ValueDisplayName(c.cat)
		if got != c.want {
			t.Errorf("ValueDisplayName(%q) = %q; want %q", c.cat, got, c.want)
		}
	}
}

func TestSchwartzValueKeys(t *testing.T) {
	keys := SchwartzValueKeys()
	if len(keys) != 10 {
		t.Errorf("len(keys) = %d; want 10", len(keys))
	}
}
