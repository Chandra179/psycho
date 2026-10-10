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
		{HighModerateLow(0), fmt.Sprintf("0–%d", int(lowLabelThreshold*100)-1), fmt.Sprintf("Low means under %d out of 100 for this text. It does not mean you are low in this trait.", int(lowLabelThreshold*100))},
		{HighModerateLow(lowLabelThreshold), fmt.Sprintf("%d–%d", int(lowLabelThreshold*100), int(highLabelThreshold*100)-1), "Moderate means this text did not show a strong high or low pattern."},
		{HighModerateLow(highLabelThreshold), fmt.Sprintf("%d–100", int(highLabelThreshold*100)), fmt.Sprintf("High means %d out of 100 or more for this text. It does not mean you are high in this trait.", int(highLabelThreshold*100))},
	}
}

// MeasureMeaning says in one plain sentence what a measure is about. It is
// the hover text on a score's name; MeasureSummary says what the tool counts.
func MeasureMeaning(key string) string {
	meanings := map[string]string{
		"openness":            "How curious and open to new ideas and experiences a person tends to be.",
		"conscientiousness":   "How organized, careful and goal-driven a person tends to be.",
		"extraversion":        "How social, talkative and full of energy a person tends to be.",
		"agreeableness":       "How kind, cooperative and trusting a person tends to be.",
		"neuroticism":         "How easily a person tends to feel stress, worry or low mood.",
		"regulatory_focus":    "Whether a person focuses more on reaching goals and gains, or on doing their duty and avoiding losses.",
		"need_for_cognition":  "How much a person enjoys thinking hard about problems.",
		"cognitive_style":     "Whether a person tends to write in a formal, organized way or in a story-like way.",
		"need_for_closure":    "How much a person wants clear, firm answers instead of staying unsure.",
		"analytical_thinking": "How formal and logical the writing sounds.",
		"clout":               "How sure and confident the writing sounds.",
		"authenticity":        "How personal and casual the writing sounds.",
		"emotional_tone":      "Whether the writing sounds more positive or more negative.",
	}
	return meanings[key]
}

// MeasureSummary is the one visible line saying what a measure counts. It
// is deliberately about word patterns: low Authenticity means formal
// wording, not dishonesty, and the traits are proxies, not the constructs.
// Keys cover the nine trait dimensions and the four summary variables.
func MeasureSummary(key string) string {
	summaries := map[string]string{
		"openness":            "Goes up with words like \"the\", \"of\", \"and\" and \"with\". Goes down with more \"I\" and \"me\" words, time words, movement words and past-tense verbs.",
		"conscientiousness":   "Goes up with words about achievement. Goes down with \"no\" and \"not\" words, negative-emotion words and words like \"but\".",
		"extraversion":        "Goes up with more social words, positive-emotion words and pronouns like \"I\" and \"we\".",
		"agreeableness":       "Goes up with words like \"and\" and \"with\", positive-emotion words, and place and movement words. Goes down with negative-emotion words.",
		"neuroticism":         "Goes up with negative-emotion words, \"no\" and \"not\" words, thinking words, sure-sounding words and hedging words like \"maybe\". It counts words, not your mood.",
		"regulatory_focus":    "Compares words about gains and goals with words about duty and avoiding loss.",
		"need_for_cognition":  "Compares analytic words with gut-feeling words. It is a rough sign of how much you enjoy hard thinking.",
		"cognitive_style":     "Formal, organized wording (many \"the\" and \"of\") versus story-like wording (many \"I\", \"was\" and \"and\"). It follows a published word list.",
		"need_for_closure":    "Compares sure-sounding words with hedging words like \"maybe\". It is a rough sign of how comfortable you are with not knowing.",
		"analytical_thinking": "Formal, reasoning-heavy wording versus personal, story-like wording.",
		"clout":               "Sure, social and achievement wording versus hedging, \"I\" and negative-emotion wording. Low does not mean low status.",
		"authenticity":        "Personal, casual wording versus formal, reasoning-heavy wording. Low means formal, not dishonest.",
		"emotional_tone":      "Positive-emotion words minus negative-emotion words. A word after \"not\" (like \"not happy\") counts the other way. It only counts words, so it misses sarcasm and context.",
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

func PercentileDescription(percentile int, reference *ingest.PercentileReference) string {
	if reference != nil {
		switch reference.Method {
		case ingest.PercentileMethodEmpirical:
			return fmt.Sprintf("Compared with the reference texts, this is about the %s percentile.", Ordinal(percentile))
		case ingest.PercentileMethodNormalApproximation:
			return fmt.Sprintf("Estimated %s percentile, using a rough bell-curve guess.", Ordinal(percentile))
		}
	}
	return fmt.Sprintf("Percentile method not saved (%s percentile).", Ordinal(percentile))
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
