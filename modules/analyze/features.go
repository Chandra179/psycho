package analyze

import (
	"math"
	"slices"

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

	for _, w := range words {
		totalWordLen += len(w)
		// Preserve the model's byte-length proxy for long words. For ASCII
		// words this agrees with LIWC Sixltr; Unicode byte lengths differ.
		if len(w) > 6 {
			bigWords++
		}
		cats := fe.dict.Lookup(w)
		if len(cats) > 0 {
			dictMatched++
		}
		for _, c := range cats {
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

// ComputeSummaryVariables preserves the existing project formulas.
func ComputeSummaryVariables(fv FeatureVector) SummaryVariables {
	s, _ := ComputeSummaryVariablesWithDetails(fv)
	return s
}

func ComputeSummaryVariablesWithDetails(fv FeatureVector) (SummaryVariables, map[string]SummaryCalculation) {
	p := fv.CategoryPercents
	et := p["positive_emotion"] - p["negative_emotion"]
	at := p["article"] + p["cognitive_process"] + p["cause"] + p["certainty"] + p["exclusive"] + p["quantitative"] -
		p["pronoun"] - p["tentative"] - p["inclusive"] - p["sensation"] - p["time"]
	cl := p["certainty"] + p["social"] + p["achievement"] + p["exclusive"] -
		p["pronoun"] - p["tentative"] - p["negative_emotion"]
	au := p["pronoun"] + p["tentative"] + p["present_focus"] + p["inclusive"] + p["sensation"] -
		fv.BigWordRatio*100 - p["cognitive_process"] - p["cause"] - p["past_focus"] - p["exclusive"] - p["certainty"]
	details := map[string]SummaryCalculation{
		"emotional_tone":      summaryCalculation(fv, "positive_emotion - negative_emotion", et, 5.0, []string{"positive_emotion", "negative_emotion"}),
		"analytical_thinking": summaryCalculation(fv, "article + cognitive_process + cause + certainty + exclusive + quantitative - pronoun - tentative - inclusive - sensation - time", at, 8.0, []string{"article", "cognitive_process", "cause", "certainty", "exclusive", "quantitative", "pronoun", "tentative", "inclusive", "sensation", "time"}),
		"clout":               summaryCalculation(fv, "certainty + social + achievement + exclusive - pronoun - tentative - negative_emotion", cl, 5.0, []string{"certainty", "social", "achievement", "exclusive", "pronoun", "tentative", "negative_emotion"}),
		"authenticity":        summaryCalculation(fv, "pronoun + tentative + present_focus + inclusive + sensation - long_word_ratio - cognitive_process - cause - past_focus - exclusive - certainty", au, 6.0, []string{"pronoun", "tentative", "present_focus", "inclusive", "sensation", "long_word_ratio", "cognitive_process", "cause", "past_focus", "exclusive", "certainty"}),
	}
	return SummaryVariables{
		AnalyticalThinking: details["analytical_thinking"].Score,
		Clout:              details["clout"].Score,
		Authenticity:       details["authenticity"].Score,
		EmotionalTone:      details["emotional_tone"].Score,
	}, details
}

func summaryCalculation(fv FeatureVector, expression string, numerator, divisor float64, categories []string) SummaryCalculation {
	inputs := make(map[string]float64, len(categories))
	for _, cat := range categories {
		inputs[cat] = fv.CategoryPercents[Category(cat)]
		if cat == "long_word_ratio" {
			inputs[cat] = fv.BigWordRatio * 100
		}
	}
	scaled := numerator / divisor
	value := sigmoid(scaled)
	return SummaryCalculation{Formula: "numerator = " + expression + "; scaled_input = numerator / divisor; sigmoid_value = 1 / (1 + exp(-scaled_input)); score = round2(sigmoid_value)", Inputs: inputs, Numerator: numerator, Divisor: divisor, ScaledInput: scaled, SigmoidValue: value, Score: math.Round(value*100) / 100}
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func tokenizeWords(s string) []string { return ingest.TokenizeWords(s) }
