package report

import (
	"fmt"

	"psycho/modules/analyze"
)

// Emotion-word thresholds for the supportive note. Both must hold so a text
// with a few negative words, or with plenty of positive ones, is not flagged.
const (
	supportMinNegativeShare = 0.02
	supportNegativeRatio    = 2
)

// Glance is the plain-language block shown above the scores. Every field is
// derived from the recorded analysis; nothing here reruns a model.
type Glance struct {
	Read           string   // what was read: size and dictionary coverage
	QualityReasons []string // why the reading-quality flag is not "high"
	Emotion        string   // positive and negative word counts, when recorded
	FitNotes       []string // text-type notes (see fit.go)
	Caveat         string
	SupportNote    string
}

// BuildGlance summarizes what was read and what limits the reading.
func BuildGlance(a *Analysis) Glance {
	g := Glance{
		Read:           fmt.Sprintf("This reading is based on %s words, and %d%% of them matched the dictionary.", analyze.FormatCount(a.WordCount), percentOf(a.DictionaryCoverage)),
		QualityReasons: analyze.QualityReasonsWithNoise(a.WordCount, a.DictionaryCoverage, noiseShare(a)),
		Caveat:         analyze.ReadingCaveat,
	}
	pos, neg, ok := emotionCounts(a)
	if ok {
		g.Emotion = fmt.Sprintf("Emotion words found: %d negative-feeling, %d positive-feeling.", neg, pos)
		if a.WordCount > 0 && float64(neg)/float64(a.WordCount) >= supportMinNegativeShare && neg >= supportNegativeRatio*pos {
			g.SupportNote = "This tool cannot assess mental health. If these words describe how you have been feeling lately, talking with someone you trust or a health professional can help."
		}
	}
	g.FitNotes, _ = FitNotes(a)
	return g
}

// emotionCounts reads the recorded dictionary counts. ok is false for saved
// analyses that predate calculation details.
func emotionCounts(a *Analysis) (pos, neg int, ok bool) {
	if a.CalculationDetails == nil || a.CalculationDetails.CategoryCounts == nil {
		return 0, 0, false
	}
	counts := a.CalculationDetails.CategoryCounts
	p, pok := counts[analyze.Category("positive_emotion")]
	n, nok := counts[analyze.Category("negative_emotion")]
	if !pok && !nok {
		return 0, 0, false
	}
	return p, n, true
}

func percentOf(f float64) int {
	return int(f*100 + 0.5)
}

// noiseShare reads the recorded extraction-noise share; saved analyses that
// predate it count as clean.
func noiseShare(a *Analysis) float64 {
	if a.CalculationDetails == nil {
		return 0
	}
	return a.CalculationDetails.NoiseShare
}
