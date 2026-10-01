package analyze

// Canonical display names and score labels for the trait dimensions.
// Every rendered surface — narrative, PDF, HTML previews — goes through
// these so wording and the 0.65/0.35 cut-offs can never drift apart
// between outputs (issue #14).

const (
	highLabelThreshold = 0.65
	lowLabelThreshold  = 0.35
)

// DimensionDisplayName returns the human-readable name for a trait key,
// or the key itself when it is unknown.
func DimensionDisplayName(key string) string {
	names := map[string]string{
		"openness":           "Openness",
		"conscientiousness":  "Conscientiousness",
		"extraversion":       "Extraversion",
		"agreeableness":      "Agreeableness",
		"neuroticism":        "Neuroticism",
		"regulatory_focus":   "Regulatory Focus",
		"need_for_cognition": "Need for Cognition",
		"cognitive_style":    "Cognitive Style",
		"need_for_closure":   "Need for Closure",
	}
	if n, ok := names[key]; ok {
		return n
	}
	return key
}

// DimensionLabel returns the descriptive band for a dimension score,
// routing dimensions with dedicated scales to their own labelers and the
// rest through the shared high/moderate/low thresholds.
func DimensionLabel(key string, score float64) string {
	switch key {
	case "regulatory_focus":
		return ComputeRegulatoryFocusLabel(score)
	case "need_for_cognition":
		return ComputeNeedForCognitionLabel(score)
	case "cognitive_style":
		return ComputeCognitiveStyleLabel(score)
	case "need_for_closure":
		return ComputeNeedForClosureLabel(score)
	}
	return HighModerateLow(score)
}

// HighModerateLow classifies a [0,1] score against the shared label
// thresholds: >= 0.65 is "high", < 0.35 is "low", else "moderate".
func HighModerateLow(score float64) string {
	if score >= highLabelThreshold {
		return "high"
	}
	if score < lowLabelThreshold {
		return "low"
	}
	return "moderate"
}
