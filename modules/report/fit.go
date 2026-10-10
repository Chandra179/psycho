package report

import "psycho/modules/analyze"

// Fit thresholds. They were set on the 3,992-post reference corpus and the nine
// samples after Cognitive Style moved to the function-word index and Authenticity
// lost its long-word term: formal prose scored Authenticity 0.08 to 0.27 and
// Cognitive Style 0.63 to 0.80 with 28% to 53% long words, while the diary entry
// (0.86, 0.30), the angry review (0.66, 0.41) and the tweet thread (0.31, 0.55)
// sit well clear. The three conditions together flag 1.9% of reference posts.
// Extraversion is included because the formal samples score 0.44 to 0.45 against
// 0.53 for the diary entry; Agreeableness (0.47 to 0.53, no pattern) and Need
// for Cognition (only the abstract is high) showed no consistent shift, so they
// are not flagged.
// Emotion words under half a percent of the text leave Emotional tone and
// Neuroticism with almost no signal.
const (
	formalMaxAuthenticity   = 0.30
	formalMinCognitiveStyle = 0.60
	formalMinLongWordPct    = 25.0
	// formalMaxFirstPerson: formal samples use no first-person singular words
	// (0.0% to 0.04%); the diary entry, angry review and tweet thread use 7.7%
	// to 12%. Only 7% of reference posts fall under 1%, so this only trims
	// false alarms from personal writing that scores like formal prose.
	formalMaxFirstPerson = 0.01
	sparseEmotionShare   = 0.005
)

// formalKeys are the measures that mostly reflect register, not the writer,
// when the text is formal. sparseEmotionKeys lean on emotion words.
var (
	formalKeys        = []string{"openness", "extraversion", "authenticity", "clout", "cognitive_style"}
	sparseEmotionKeys = []string{"neuroticism", "emotional_tone"}
)

var fitReasons = map[string]string{
	"openness":        "Articles and prepositions, common in formal prose, mostly drive this score.",
	"extraversion":    "Impersonal wording with few words about people pulls this score down, whatever the writer is like.",
	"authenticity":    "Formal wording, with few personal pronouns, drives this score down. It says nothing about honesty.",
	"clout":           "Formal, impersonal wording mostly sets this score, not status or confidence.",
	"cognitive_style": "Articles and prepositions in formal writing push this up, whatever the writer is like.",
	"neuroticism":     "Very few emotion words were found, so this score rests on other word types.",
	"emotional_tone":  "Very few emotion words were found, so this score says little about this text.",
}

// FitNotes reports where the text type limits the reading. top holds
// sentences for the glance block; byKey maps a measure key to a one-line
// reason shown on that measure's card. Short text and low dictionary
// coverage are explained by analyze.QualityReasons instead, so they are not
// repeated here. Fields missing from older saved analyses simply skip the
// rule that needs them.
func FitNotes(a *Analysis) (top []string, byKey map[string]string) {
	byKey = map[string]string{}
	if isFormalProse(a) {
		top = append(top, "This reads like formal writing with many long words. Every score here is rough, and Openness, Extraversion, Confident wording, Cognitive Style and Personal wording especially so, because they mostly reflect wording and topic, not the writer.")
		for _, k := range formalKeys {
			byKey[k] = fitReasons[k]
		}
	}
	if pos, neg, ok := emotionCounts(a); ok && a.WordCount > 0 && float64(pos+neg)/float64(a.WordCount) < sparseEmotionShare {
		top = append(top, "Very few emotion words were found, so Emotional tone and Neuroticism say little about this text.")
		for _, k := range sparseEmotionKeys {
			byKey[k] = fitReasons[k]
		}
	}
	if len(byKey) == 0 {
		byKey = nil
	}
	return top, byKey
}

// isFormalProse needs a low Authenticity score and a high Cognitive Style
// score. When the recorded long-word share is available it must also be high,
// which guards against terse, impersonal texts that score the same way for
// other reasons.
func isFormalProse(a *Analysis) bool {
	if a.Summary.Authenticity > formalMaxAuthenticity {
		return false
	}
	if style, ok := a.Traits["cognitive_style"]; !ok || style.Score < formalMinCognitiveStyle {
		return false
	}
	if a.CalculationDetails == nil || a.WordCount == 0 {
		return true
	}
	if n, ok := a.CalculationDetails.CategoryCounts[analyze.Category("first_person_singular")]; ok &&
		float64(n)/float64(a.WordCount) >= formalMaxFirstPerson {
		return false
	}
	if a.CalculationDetails.BigWordCount == 0 {
		return true
	}
	return float64(a.CalculationDetails.BigWordCount)/float64(a.WordCount)*100 >= formalMinLongWordPct
}
