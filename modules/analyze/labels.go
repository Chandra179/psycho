package analyze

import (
	"fmt"
	"strings"

	"psycho/modules/ingest"
)

// Canonical display names and score labels for the trait dimensions.
// Every rendered surface — narrative, PDF, HTML previews — goes through
// these so wording and the 0.65/0.35 cut-offs can never drift apart
// between outputs (issue #14).

const (
	highLabelThreshold = 0.65
	lowLabelThreshold  = 0.35

	// Ranks at or beyond these read as extreme in PercentileMainLine.
	extremeRankHigh = 80
	extremeRankLow  = 20
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

// ScoreBand is shared explanatory copy for report legends and disclosures.
type ScoreBand struct {
	Label, Range, Description string
}

func BigFiveBands() []ScoreBand {
	return []ScoreBand{
		{HighModerateLow(0), fmt.Sprintf("0–%d", int(lowLabelThreshold*100)-1), fmt.Sprintf("Low is below %d/100 on this tool's text-score scale. It does not establish a low personality trait.", int(lowLabelThreshold*100))},
		{HighModerateLow(lowLabelThreshold), fmt.Sprintf("%d–%d", int(lowLabelThreshold*100), int(highLabelThreshold*100)-1), "Moderate means the word-pattern model found no strong high or low signal in this text."},
		{HighModerateLow(highLabelThreshold), fmt.Sprintf("%d–100", int(highLabelThreshold*100)), fmt.Sprintf("High is %d/100 or above on this tool's text-score scale. It does not establish a high personality trait.", int(highLabelThreshold*100))},
	}
}

// DimensionBandDescription derives each label from its actual classification
// function, including additional measures whose upper band starts above .65.
func DimensionBandDescription(key string, score float64) string {
	label := DimensionLabel(key, score)
	for _, band := range BigFiveBands() {
		if label == band.Label && key != "need_for_cognition" && key != "need_for_closure" {
			return band.Description
		}
	}
	rangeText := fmt.Sprintf("%d–%d/100", int(lowLabelThreshold*100), int(highLabelThreshold*100))
	if score < lowLabelThreshold {
		rangeText = fmt.Sprintf("below %d/100", int(lowLabelThreshold*100))
	} else if score >= highLabelThreshold {
		rangeText = fmt.Sprintf("%d/100 or above", int(highLabelThreshold*100))
	}
	meanings := map[string]string{
		"promotion_focus":  "Promotion-related language outweighs prevention-related language in this bipolar proxy.",
		"prevention_focus": "Prevention-related language outweighs promotion-related language in this bipolar proxy.",
		"balanced":         "No strong promotion or prevention tilt in this proxy.",
		"systematic":       "More systematic or analytical word-pattern signals on this proxy.",
		"intuitive":        "More intuitive word-pattern signals on this proxy.",
		"mixed":            "No strong systematic or intuitive tilt in this proxy.",
		"moderate":         "No strong high or low word-pattern signal in this proxy.",
		"high":             "An upper-band word-pattern score in this proxy.",
		"low":              "A lower-band word-pattern score in this proxy.",
	}
	return fmt.Sprintf("%s: %s. %s Labels describe word patterns, not a validated personality assessment.", label, rangeText, meanings[label])
}

// MeasureSummary is the one visible line saying what a measure counts. It
// is deliberately about word patterns: low Authenticity means formal
// wording, not dishonesty, and the traits are proxies, not the constructs.
// Keys cover the nine trait dimensions and the four summary variables.
func MeasureSummary(key string) string {
	summaries := map[string]string{
		"openness":            "Higher with more articles, prepositions and inclusive words; lower with more pronouns, time, motion and past-tense words.",
		"conscientiousness":   "Higher with achievement words; lower with negations, negative-emotion words and exclusion words such as \"but\".",
		"extraversion":        "Higher with more social, positive-emotion and pronoun words.",
		"agreeableness":       "Higher with inclusive, positive-emotion, space and motion words; lower with negative-emotion words.",
		"neuroticism":         "Higher with negative-emotion, negation, reasoning, certainty and hedging words. It counts word patterns, not mood.",
		"regulatory_focus":    "Compares gain and aspiration words (promotion) with duty and loss-avoidance words (prevention).",
		"need_for_cognition":  "Compares analytic words with intuitive words, as a proxy for enjoying effortful thinking.",
		"cognitive_style":     "Systematic wording (reasoning, cause, long words) versus intuitive wording (senses, personal, present-tense). Long formal words push it up.",
		"need_for_closure":    "Compares certainty words with hedging words, as a proxy for comfort with ambiguity.",
		"analytical_thinking": "Formal, reasoning-heavy wording versus personal, story-like wording.",
		"clout":               "Certain, social and achievement wording versus hedging, personal-pronoun and negative-emotion wording. Low does not mean low status.",
		"authenticity":        "Personal, informal wording versus formal wording with many long words. Low means formal, not dishonest.",
		"emotional_tone":      "Positive-emotion words minus negative-emotion words. It counts words only, so clearly distressed text can still read as neutral.",
	}
	return summaries[key]
}

// CategoryLabel turns a dictionary category key into reader-facing words for
// the evidence table; the raw key stays available as a title attribute.
func CategoryLabel(key string) string {
	if key == "long_word_ratio" {
		return "long words (over six bytes)"
	}
	return strings.ReplaceAll(key, "_", " ")
}

func SummarySignalLabel(name string, score float64) string {
	if name == "emotional_tone" {
		return SummaryTone(score) + " language"
	}
	return HighModerateLow(score) + " signal"
}

func SummarySignalDescription(name string, score float64) string {
	for _, band := range BigFiveBands() {
		if band.Label == HighModerateLow(score) {
			return fmt.Sprintf("%s (%s/100): a project-defined language summary, not an official LIWC score or a probability of accuracy.", SummarySignalLabel(name, score), band.Range)
		}
	}
	return "Project-defined language summary."
}

// PercentileMainLine is the plain rank sentence shown beside a trait score.
// It is empty unless the percentile came from an empirical reference sample,
// because a normal approximation or an unrecorded method is not a rank among
// real texts. When the band label says moderate (or balanced, mixed) but the
// rank is extreme, it explains why: most reference scores cluster tightly
// around 50, so a "moderate" score can still rank high or low.
func PercentileMainLine(percentile int, reference *ingest.PercentileReference, label string) string {
	if reference == nil || reference.Method != ingest.PercentileMethodEmpirical || percentile < 1 || percentile > 99 {
		return ""
	}
	line := fmt.Sprintf("Higher than about %d of 100 reference texts.", percentile)
	middle := label == "moderate" || label == "balanced" || label == "mixed"
	switch {
	case middle && percentile >= extremeRankHigh:
		line += fmt.Sprintf(" %s on the 0–100 scale, but reference scores cluster tightly, so it ranks high.", capitalize(label))
	case middle && percentile <= extremeRankLow:
		line += fmt.Sprintf(" %s on the 0–100 scale, but reference scores cluster tightly, so it ranks low.", capitalize(label))
	}
	return line
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func PercentileDescription(percentile int, reference *ingest.PercentileReference) string {
	if reference != nil {
		switch reference.Method {
		case ingest.PercentileMethodEmpirical:
			return fmt.Sprintf("Approximate reference-text percentile: %s.", Ordinal(percentile))
		case ingest.PercentileMethodNormalApproximation:
			return fmt.Sprintf("Model-estimated %s percentile from a normal approximation.", Ordinal(percentile))
		}
	}
	return fmt.Sprintf("Percentile method not recorded (%s percentile).", Ordinal(percentile))
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
