package analyze

// Schwartz value categories as they appear in the dictionary.
var schwartzValueCategories = []Category{
	"value_self_direction",
	"value_stimulation",
	"value_hedonism",
	"value_achievement",
	"value_power",
	"value_security",
	"value_conformity",
	"value_tradition",
	"value_benevolence",
	"value_universalism",
}

// valueDisplayNames maps category keys to human-readable labels.
var valueDisplayNames = map[Category]string{
	"value_self_direction": "Self-direction",
	"value_stimulation":    "Stimulation",
	"value_hedonism":       "Hedonism",
	"value_achievement":    "Achievement",
	"value_power":          "Power",
	"value_security":       "Security",
	"value_conformity":     "Conformity",
	"value_tradition":      "Tradition",
	"value_benevolence":    "Benevolence",
	"value_universalism":   "Universalism",
}

// ValueCategory returns the Category for a given Schwartz value key.
func ValueCategory(key string) Category {
	return Category(key)
}

// ValueDisplayName returns the human-readable name for a value category.
func ValueDisplayName(cat Category) string {
	if n, ok := valueDisplayNames[cat]; ok {
		return n
	}
	return string(cat)
}

// SchwartzValueKeys returns all value category keys.
func SchwartzValueKeys() []Category {
	out := make([]Category, len(schwartzValueCategories))
	copy(out, schwartzValueCategories)
	return out
}

// ComputeSchwartzValues extracts Schwartz value scores from a feature vector.
// Each score is the percentage of words matching that value's word list.
func ComputeSchwartzValues(fv FeatureVector) map[string]float64 {
	values := make(map[string]float64, len(schwartzValueCategories))
	for _, cat := range schwartzValueCategories {
		pct := fv.CategoryPercents[cat]
		key := string(cat)
		values[key] = pct
	}
	return values
}
