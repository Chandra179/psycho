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

// ValueDescription describes dictionary topics, not inferred personal values.
func ValueDescription(cat Category) string {
	descriptions := map[Category]string{
		"value_self_direction": "Independence, choice and creativity",
		"value_stimulation":    "Novelty, challenge and excitement",
		"value_hedonism":       "Pleasure, enjoyment and comfort",
		"value_achievement":    "Goals, accomplishment and competence",
		"value_power":          "Influence, authority and social standing",
		"value_security":       "Safety, stability and order",
		"value_conformity":     "Rules, obligations and social expectations",
		"value_tradition":      "Customs, heritage and cultural references",
		"value_benevolence":    "Care, help and concern for others",
		"value_universalism":   "Fairness and care beyond oneself",
	}
	return descriptions[cat]
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
