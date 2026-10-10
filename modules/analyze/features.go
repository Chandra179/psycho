package analyze

import (
	"math"
	"slices"
	"unicode/utf8"

	"psycho/modules/ingest"
)

// FeatureVector holds the percentages for each dictionary category.
type FeatureVector struct {
	CategoryPercents  map[Category]float64
	CategoryCounts    map[Category]int
	WordCount         int
	BigWordCount      int
	DictionaryMatches int
	TypeTokenRatio    float64
	BigWordRatio      float64
	AvgWordLength     float64
	// Evidence holds a bounded sample of the words that matched each
	// category — the raw material behind the percentages.
	Evidence      map[Category][]string
	ValueExcerpts map[string][]ingest.TextExcerpt
	// ScoreSE is the sampling standard error of each calibrated dimension's
	// model score, from the spread of per-word contributions (see
	// dimensionWeights). It treats words as independent draws, so it
	// understates the error for bursty real text.
	ScoreSE map[string]float64
	// NegatedPositive and NegatedNegative count emotion words that follow a
	// negator ("not happy", "not bad"); the emotional-tone summary flips them.
	NegatedPositive int
	NegatedNegative int
	// NoiseShare is ingest.Document.NoiseShare, carried so the reading
	// quality can account for extraction leftovers.
	NoiseShare float64
}

// FeatureExtractor computes psycholinguistic features from a document.
type FeatureExtractor struct {
	dict Dictionary
}

func NewFeatureExtractor(dict Dictionary) *FeatureExtractor {
	return &FeatureExtractor{dict: dict}
}

// Extract returns the feature vector and dictionary coverage.
func (fe *FeatureExtractor) Extract(doc ingest.Document) (FeatureVector, float64) {
	words := tokenizeWords(doc.RawText)
	if len(words) == 0 {
		return FeatureVector{}, 0
	}

	catCounts := make(map[Category]int)
	evidence := make(map[Category][]string)
	var dictMatched int
	var totalWordLen int
	var bigWords int
	var negPos, negNeg int
	acc := newSEAccumulator()

	for i, w := range words {
		// Long words are counted in letters, as LIWC Sixltr does, so accented
		// words and contractions are not measured by their UTF-8 bytes.
		letters := wordLetters(w)
		totalWordLen += letters
		isBig := letters > 6
		if isBig {
			bigWords++
		}
		cats := fe.dict.Lookup(w)
		if len(cats) > 0 {
			dictMatched++
		}
		cats = countedCategories(words, i, fe.dict)
		acc.add(cats)
		for _, c := range cats {
			if c == "positive_emotion" && negatedAt(words, i) {
				negPos++
			} else if c == "negative_emotion" && negatedAt(words, i) {
				negNeg++
			}
			catCounts[c]++
			if ev := evidence[c]; len(ev) < MaxEvidenceWords && !slices.Contains(ev, w) {
				evidence[c] = append(ev, w)
			}
		}
	}

	wordCount := float64(len(words))
	coverage := float64(dictMatched) / wordCount

	catPercents := make(map[Category]float64, len(fe.dict.Categories()))
	for _, c := range fe.dict.Categories() {
		// Retain explicit zero counts for unmatched categories.
		if _, present := catCounts[c]; !present {
			catCounts[c] = 0
		}
		catPercents[c] = float64(catCounts[c]) / wordCount * 100
	}

	var avgWordLen float64
	if len(words) > 0 {
		avgWordLen = float64(totalWordLen) / float64(len(words))
	}

	fv := FeatureVector{
		CategoryPercents:  catPercents,
		CategoryCounts:    catCounts,
		WordCount:         len(words),
		BigWordCount:      bigWords,
		DictionaryMatches: dictMatched,
		TypeTokenRatio:    doc.TypeTokenRatio,
		BigWordRatio:      float64(bigWords) / wordCount,
		AvgWordLength:     avgWordLen,
		Evidence:          evidence,
		ValueExcerpts:     valueExcerpts(doc.RawText, fe.dict),
		ScoreSE:           acc.standardErrors(),
		NegatedPositive:   negPos,
		NegatedNegative:   negNeg,
		NoiseShare:        doc.NoiseShare,
	}
	return fv, coverage
}

// SummaryVariables holds project-defined language proxies inspired by research
// on LIWC measures. These sigmoid formulas are not the proprietary LIWC
// summary algorithms or standardized LIWC percentiles.
type SummaryVariables struct {
	AnalyticalThinking float64 `json:"analytical_thinking"`
	Clout              float64 `json:"clout"`
	Authenticity       float64 `json:"authenticity"`
	EmotionalTone      float64 `json:"emotional_tone"`
}

// authenticityCenter is the reference-corpus median of the authenticity
// numerator before centering (20.7 over the 3,992 posts, measured 2026-10-09
// after the long-word term was removed and the mostly-wrong-sense words were
// dropped from the scored lists; it was 19.2 before that). Without it the sigmoid saturates near
// 1 for ordinary text. The constant is a project assumption, not a LIWC value.
const authenticityCenter = 20.7

// ComputeSummaryVariables preserves the existing project formulas.
func ComputeSummaryVariables(fv FeatureVector) SummaryVariables {
	s, _ := ComputeSummaryVariablesWithDetails(fv)
	return s
}

func ComputeSummaryVariablesWithDetails(fv FeatureVector) (SummaryVariables, map[string]SummaryCalculation) {
	p := fv.CategoryPercents
	// A negated emotion word flips sides: "not happy" counts as negative and
	// "not bad" as positive. It is removed from its own side and added to the
	// other, so the net moves by twice the negated share.
	negPos, negNeg := fv.negatedEmotionPercents()
	et := p["positive_emotion"] - p["negative_emotion"] - 2*negPos + 2*negNeg
	at := p["article"] + p["cognitive_process"] + p["cause"] + p["certainty"] + p["exclusive"] + p["quantitative"] -
		p["pronoun"] - p["tentative"] - p["inclusive"] - p["sensation"] - p["time"]
	cl := p["certainty"] + p["social"] + p["achievement"] + p["exclusive"] -
		p["pronoun"] - p["tentative"] - p["negative_emotion"]
	au := p["pronoun"] + p["tentative"] + p["present_focus"] + p["inclusive"] + p["sensation"] -
		p["cognitive_process"] - p["cause"] - p["past_focus"] - p["exclusive"] - p["certainty"] - authenticityCenter
	details := map[string]SummaryCalculation{
		"emotional_tone":      summaryCalculation(fv, "positive_emotion - negative_emotion - 2 * negated_positive_emotion + 2 * negated_negative_emotion", et, 5.0, []string{"positive_emotion", "negative_emotion", "negated_positive_emotion", "negated_negative_emotion"}),
		"analytical_thinking": summaryCalculation(fv, "article + cognitive_process + cause + certainty + exclusive + quantitative - pronoun - tentative - inclusive - sensation - time", at, 8.0, []string{"article", "cognitive_process", "cause", "certainty", "exclusive", "quantitative", "pronoun", "tentative", "inclusive", "sensation", "time"}),
		"clout":               summaryCalculation(fv, "certainty + social + achievement + exclusive - pronoun - tentative - negative_emotion", cl, 5.0, []string{"certainty", "social", "achievement", "exclusive", "pronoun", "tentative", "negative_emotion"}),
		"authenticity":        summaryCalculation(fv, "pronoun + tentative + present_focus + inclusive + sensation - cognitive_process - cause - past_focus - exclusive - certainty - 20.7", au, 6.0, []string{"pronoun", "tentative", "present_focus", "inclusive", "sensation", "cognitive_process", "cause", "past_focus", "exclusive", "certainty"}),
	}
	return SummaryVariables{
		AnalyticalThinking: details["analytical_thinking"].Score,
		Clout:              details["clout"].Score,
		Authenticity:       details["authenticity"].Score,
		EmotionalTone:      details["emotional_tone"].Score,
	}, details
}

// negatedEmotionPercents returns the share of words, in percent, that are a
// positive or negative emotion word following a negator.
func (fv FeatureVector) negatedEmotionPercents() (pos, neg float64) {
	if fv.WordCount == 0 {
		return 0, 0
	}
	n := float64(fv.WordCount)
	return float64(fv.NegatedPositive) / n * 100, float64(fv.NegatedNegative) / n * 100
}

func summaryCalculation(fv FeatureVector, expression string, numerator, divisor float64, categories []string) SummaryCalculation {
	inputs := make(map[string]float64, len(categories))
	for _, cat := range categories {
		inputs[cat] = fv.CategoryPercents[Category(cat)]
		switch cat {
		case "negated_positive_emotion":
			inputs[cat], _ = fv.negatedEmotionPercents()
		case "negated_negative_emotion":
			_, inputs[cat] = fv.negatedEmotionPercents()
		}
	}
	scaled := numerator / divisor
	value := sigmoid(scaled)
	return SummaryCalculation{Formula: "numerator = " + expression + "; scaled_input = numerator / divisor; sigmoid_value = 1 / (1 + exp(-scaled_input)); score = round2(sigmoid_value)", Inputs: inputs, Numerator: numerator, Divisor: divisor, ScaledInput: scaled, SigmoidValue: value, Score: math.Round(value*100) / 100}
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// wordLetters counts a token's letters and digits, excluding apostrophes.
func wordLetters(w string) int {
	n := utf8.RuneCountInString(w)
	for _, r := range w {
		if r == '\'' {
			n--
		}
	}
	return n
}

func tokenizeWords(s string) []string { return ingest.TokenizeWords(s) }
