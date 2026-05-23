package analyze

import (
	"math"
	"strings"
	"unicode"

	"psycho/modules/ingest"
)

// FeatureVector holds the percentages for each dictionary category.
type FeatureVector struct {
	CategoryPercents map[Category]float64
	TypeTokenRatio   float64
	BigWordRatio     float64
	AvgWordLength    float64
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
	var dictMatched int
	var totalWordLen int
	var bigWords int

	for _, w := range words {
		totalWordLen += len(w)
		if len(w) >= 6 {
			bigWords++
		}
		cats := fe.dict.Lookup(w)
		if len(cats) > 0 {
			dictMatched++
		}
		for _, c := range cats {
			catCounts[c]++
		}
	}

	wordCount := float64(len(words))
	coverage := float64(dictMatched) / wordCount

	catPercents := make(map[Category]float64, len(fe.dict.Categories()))
	for _, c := range fe.dict.Categories() {
		catPercents[c] = float64(catCounts[c]) / wordCount * 100
	}

	var avgWordLen float64
	if len(words) > 0 {
		avgWordLen = float64(totalWordLen) / float64(len(words))
	}

	fv := FeatureVector{
		CategoryPercents: catPercents,
		TypeTokenRatio:   doc.TypeTokenRatio,
		BigWordRatio:     float64(bigWords) / wordCount,
		AvgWordLength:    avgWordLen,
	}
	return fv, coverage
}

// SummaryVariables holds LIWC-style summary dimensions derived from category percentages.
// Values range [0,1]; 0.5 represents a balanced/average score.
// Based on Pennebaker et al. (2014, 2015) — "When Small Words Foretell Academic Success"
// and the LIWC2015 Operator's Manual.
type SummaryVariables struct {
	AnalyticalThinking float64 `json:"analytical_thinking"`
	Clout              float64 `json:"clout"`
	Authenticity       float64 `json:"authenticity"`
	EmotionalTone      float64 `json:"emotional_tone"`
}

// ComputeSummaryVariables calculates LIWC-style summary variables from a FeatureVector.
// All values are mapped to [0,1] via sigmoid scaling — higher = more of the named quality.
func ComputeSummaryVariables(fv FeatureVector) SummaryVariables {
	p := fv.CategoryPercents

	et := sigmoid((p["positive_emotion"] - p["negative_emotion"]) / 5.0)

	at := sigmoid((p["article"] + p["cognitive_process"] + p["cause"] + p["certainty"] + p["exclusive"] + p["quantitative"] -
		p["pronoun"] - p["tentative"] - p["inclusive"] - p["sensation"] - p["time"]) / 8.0)

	cl := sigmoid((p["certainty"] + p["social"] + p["achievement"] + p["exclusive"] -
		p["pronoun"] - p["tentative"] - p["negative_emotion"]) / 5.0)

	au := sigmoid((p["pronoun"] + p["tentative"] + p["present_focus"] + p["inclusive"] + p["sensation"] -
		p["big_words"] - p["cognitive_process"] - p["cause"] - p["past_focus"] - p["exclusive"] - p["certainty"]) / 6.0)

	return SummaryVariables{
		AnalyticalThinking: math.Round(at*100) / 100,
		Clout:              math.Round(cl*100) / 100,
		Authenticity:       math.Round(au*100) / 100,
		EmotionalTone:      math.Round(et*100) / 100,
	}
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func tokenizeWords(s string) []string {
	var words []string
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\''
	})
	for _, w := range fields {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			words = append(words, w)
		}
	}
	return words
}
