package analyze

import (
	"fmt"
	"strings"

	"psycho/modules/ingest"
)

// Canonical display names and score labels for the trait dimensions.
// Every rendered surface (narrative, HTML report) goes through
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
		"regulatory_focus":   "Goals: gain vs. safety",
		"need_for_cognition": "Need for Cognition",
		"cognitive_style":    "Cognitive Style",
		"need_for_closure":   "Preference for certainty",
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
		"systematic":       "More formal, categorical word patterns (articles and prepositions) on this proxy.",
		"intuitive":        "More narrative, dynamic word patterns (pronouns and auxiliary verbs) on this proxy.",
		"mixed":            "No strong tilt toward categorical or dynamic wording in this proxy.",
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
		"cognitive_style":     "Formal, categorical wording (many articles and prepositions) versus narrative, dynamic wording (many pronouns, auxiliary verbs, adverbs and conjunctions). It follows a published function-word index.",
		"need_for_closure":    "Compares certainty words with hedging words, as a proxy for comfort with ambiguity.",
		"analytical_thinking": "Formal, reasoning-heavy wording versus personal, story-like wording.",
		"clout":               "Certain, social and achievement wording versus hedging, personal-pronoun and negative-emotion wording. Low does not mean low status.",
		"authenticity":        "Personal, informal wording versus formal, reasoning-heavy wording. Low means formal, not dishonest.",
		"emotional_tone":      "Positive-emotion words minus negative-emotion words, with a negated word (\"not happy\") counted on the opposite side. It still counts words only, so sarcasm and context are missed.",
	}
	return summaries[key]
}

// CategoryLabel turns a dictionary category key into reader-facing words for
// the evidence table; the raw key stays available as a title attribute.
func CategoryLabel(key string) string {
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

// Emotional tone is a net word count on a sigmoid with divisor 5, so the usual
// 35/65 bands would call a text neutral until positive words outnumber negative
// ones by about 1.9 points of all words. These tighter cutoffs (about 1 point)
// stop a clearly negative text from reading as neutral.
const (
	toneNegativeBelow = 0.45
	tonePositiveFrom  = 0.55
)

// SummaryTone words the emotional-tone variable.
func SummaryTone(score float64) string {
	switch {
	case score >= tonePositiveFrom:
		return "positive"
	case score <= toneNegativeBelow:
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
