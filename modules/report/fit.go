package report

// Fit thresholds. They were set on the 3,992-post reference corpus and the nine
// samples after Cognitive Style moved to the function-word index and Authenticity
// lost its long-word term: formal prose scored Authenticity 0.08 to 0.27 and
// Cognitive Style 0.63 to 0.80 with 28% to 53% long words, while the diary entry
// (0.86, 0.30), the angry review (0.66, 0.41) and the tweet thread (0.31, 0.55)
// sit well clear. The three conditions together flag 1.9% of reference posts.
// Emotion words under half a percent of the text leave Emotional tone and
// Neuroticism with almost no signal.
const (
	formalMaxAuthenticity   = 0.30
	formalMinCognitiveStyle = 0.60
	formalMinLongWordPct    = 25.0
	sparseEmotionShare      = 0.005
)

// formalKeys are the measures that mostly reflect register, not the writer,
// when the text is formal. sparseEmotionKeys lean on emotion words.
var (
	formalKeys        = []string{"openness", "authenticity", "clout", "cognitive_style"}
	sparseEmotionKeys = []string{"neuroticism", "emotional_tone"}
)

var fitReasons = map[string]string{
	"openness":        "Articles and prepositions, common in formal prose, mostly drive this score.",
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
		top = append(top, "This reads like formal writing with many long words. Openness, Clout, Cognitive Style and Authenticity mostly reflect wording and topic here, not the writer.")
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
	if a.CalculationDetails == nil || a.CalculationDetails.BigWordCount == 0 || a.WordCount == 0 {
		return true
	}
	return float64(a.CalculationDetails.BigWordCount)/float64(a.WordCount)*100 >= formalMinLongWordPct
}
