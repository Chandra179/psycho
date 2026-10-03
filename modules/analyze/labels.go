package analyze

import "fmt"

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

// Summary variable band wording lives here so every renderer (HTML report,
// PDF, narrative) draws from one table per register instead of carrying its
// own copy. Names match SummaryVariables JSON keys minus the "value_" style
// prefix: "analytical_thinking", "clout", "authenticity".

var summaryBandFormal = map[string][2]string{
	"analytical_thinking": {"highly analytical", "intuitive"},
	"clout":               {"confident/dominant", "submissive/uncertain"},
	"authenticity":        {"personal/honest", "guarded/distant"},
}

var summaryBandCompact = map[string][2]string{
	"analytical_thinking": {"analytical", "intuitive"},
	"clout":               {"confident", "reserved"},
	"authenticity":        {"personal", "guarded"},
}

// SummaryBandFormal returns the band wording used in long-form output (PDF,
// narrative): high-word, low-word pairs like "confident/dominant".
func SummaryBandFormal(name string, score float64) string {
	return summaryBand(summaryBandFormal, name, score)
}

// SummaryBandCompact returns the short wording used in the HTML report's
// summary cards.
func SummaryBandCompact(name string, score float64) string {
	return summaryBand(summaryBandCompact, name, score)
}

func summaryBand(table map[string][2]string, name string, score float64) string {
	band, ok := table[name]
	if !ok {
		return "moderate"
	}
	switch HighModerateLow(score) {
	case "high":
		return band[0]
	case "low":
		return band[1]
	}
	return "moderate"
}

// SummaryTone words the emotional-tone variable.
func SummaryTone(score float64) string {
	switch HighModerateLow(score) {
	case "high":
		return "positive"
	case "low":
		return "negative"
	}
	return "neutral"
}

// Ordinal returns n with its English ordinal suffix: 1st, 2nd, 3rd, 4th…
func Ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
