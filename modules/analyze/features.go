package analyze

import (
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
