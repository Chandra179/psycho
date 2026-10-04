package analyze

import (
	"math"
	"slices"
	"sort"
)

// MaxEvidenceWords bounds how many matched words Extract keeps per
// dictionary category as sample evidence.
const MaxEvidenceWords = 10

// Contribution is one dictionary category's contribution to a dimension
// score: the category's share of the words, the per-percentage-point weight
// applied to it, and their product. Sorted lists of these are the audit
// trail behind a score — every number in a report traces back to one.
type Contribution struct {
	Category     string   `json:"category"`
	WordPercent  float64  `json:"word_percent"`
	Weight       float64  `json:"weight"`
	Contribution float64  `json:"contribution"`
	MatchedWords []string `json:"matched_words,omitempty"`
}

// BigFiveEvidence returns the contributions behind one Big Five trait,
// strongest (largest absolute contribution) first. Categories the trait has
// no weight for, or that matched no words, are skipped.
func BigFiveEvidence(trait string, fv FeatureVector) []Contribution {
	var out []Contribution
	for catName, weights := range coefficients {
		w := weights.weightFor(trait)
		pct := fv.CategoryPercents[Category(catName)]
		if w == 0 || pct == 0 {
			continue
		}
		out = append(out, newContribution(catName, pct, w, fv.Evidence[Category(catName)]))
	}
	sortContributions(out)
	return out
}

func RegulatoryFocusEvidence(fv FeatureVector) []Contribution {
	return dimensionEvidence(regFocusCoefficients, fv)
}

func NeedForCognitionEvidence(fv FeatureVector) []Contribution {
	return dimensionEvidence(needCogCoefficients, fv)
}

func NeedForClosureEvidence(fv FeatureVector) []Contribution {
	return dimensionEvidence(needClosureCoefficients, fv)
}

// CognitiveStyleEvidence adds the computed long-word ratio, which
// contributes to the score but is not a dictionary category.
func CognitiveStyleEvidence(fv FeatureVector) []Contribution {
	out := dimensionEvidence(cognitiveStyleCoefficients, fv)
	if fv.BigWordRatio > 0 {
		out = append(out, newContribution("long_word_ratio", fv.BigWordRatio*100, bigWordsWeight, nil))
		sortContributions(out)
	}
	return out
}

func dimensionEvidence(weights map[string]float64, fv FeatureVector) []Contribution {
	var out []Contribution
	for catName, w := range weights {
		pct := fv.CategoryPercents[Category(catName)]
		if w == 0 || pct == 0 {
			continue
		}
		out = append(out, newContribution(catName, pct, w, fv.Evidence[Category(catName)]))
	}
	sortContributions(out)
	return out
}

func newContribution(category string, wordPercent, weight float64, matchedWords []string) Contribution {
	if len(matchedWords) > MaxEvidenceWords {
		matchedWords = matchedWords[:MaxEvidenceWords]
	}
	return Contribution{
		Category:     category,
		WordPercent:  round2(wordPercent),
		Weight:       weight,
		Contribution: round4(weight * wordPercent),
		MatchedWords: slices.Clone(matchedWords),
	}
}

func sortContributions(out []Contribution) {
	sort.Slice(out, func(i, j int) bool {
		a, b := math.Abs(out[i].Contribution), math.Abs(out[j].Contribution)
		if a == b {
			return out[i].Category < out[j].Category
		}
		return a > b
	})
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*1e4) / 1e4 }

func (tw TraitWeights) weightFor(trait string) float64 {
	switch trait {
	case "openness":
		return tw.Openness
	case "conscientiousness":
		return tw.Conscientiousness
	case "extraversion":
		return tw.Extraversion
	case "agreeableness":
		return tw.Agreeableness
	case "neuroticism":
		return tw.Neuroticism
	}
	return 0
}
