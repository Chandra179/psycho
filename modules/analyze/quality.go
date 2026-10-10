package analyze

import (
	"fmt"
	"math"
	"strconv"
)

// Reading-quality thresholds. profile.computeConfidenceFlag and the report's
// plain-language reasons both read these, so the flag and its explanation
// cannot drift apart.
const (
	QualityLowWords    = 500
	QualityHighWords   = 1000
	QualityMinCoverage = 0.6
	// QualityLowCoverage sits below the 2nd percentile of the 3,992-post
	// reference corpus (median coverage 63%), so only text the dictionary barely
	// recognizes (another language, code, heavy jargon) is called low.
	QualityLowCoverage = 0.45
	// QualityNoisyShare is the share of non-blank lines Normalize dropped as
	// page numbers, captions or running headers above which the reading is at
	// most "medium". None of the 4,010 reference posts lost a line, so the
	// reference corpus cannot set this; 5% is where the two scan-style
	// samples (5.9% and 6.3%) fall and ordinary text (0%) does not.
	QualityNoisyShare = 0.05
)

// QualityFlag classifies a sample as "low", "medium" or "high" reading
// quality: under QualityLowWords or under QualityLowCoverage is low; coverage
// under QualityMinCoverage or under QualityHighWords is medium; otherwise high.
func QualityFlag(wordCount int, coverage float64) string {
	return QualityFlagWithNoise(wordCount, coverage, 0)
}

// QualityFlagWithNoise is QualityFlag with the share of lines removed as
// extraction noise; a share of QualityNoisyShare or more caps the flag at
// "medium".
func QualityFlagWithNoise(wordCount int, coverage, noiseShare float64) string {
	if wordCount < QualityLowWords || coverage < QualityLowCoverage {
		return "low"
	}
	if coverage < QualityMinCoverage || noiseShare >= QualityNoisyShare {
		return "medium"
	}
	if wordCount < QualityHighWords {
		return "medium"
	}
	return "high"
}

// QualityReasons returns one plain sentence per factor that held the flag
// below "high". It returns nil when nothing limits the reading.
func QualityReasons(wordCount int, coverage float64) []string {
	return QualityReasonsWithNoise(wordCount, coverage, 0)
}

// QualityReasonsWithNoise is QualityReasons plus the extraction-noise reason.
func QualityReasonsWithNoise(wordCount int, coverage, noiseShare float64) []string {
	var reasons []string
	if wordCount < QualityLowWords {
		reasons = append(reasons, fmt.Sprintf("The text is short (under %s words), so scores can swing a lot.", FormatCount(QualityLowWords)))
	} else if wordCount < QualityHighWords {
		reasons = append(reasons, fmt.Sprintf("The text is under %s words, so scores are rough.", FormatCount(QualityHighWords)))
	}
	if coverage < QualityLowCoverage {
		reasons = append(reasons, "Under 45% of the words matched the dictionary, so most of the text was not scored.")
	} else if coverage < QualityMinCoverage {
		reasons = append(reasons, "Under 60% of the words matched the dictionary, so part of the text was not scored.")
	}
	if noiseShare >= QualityNoisyShare {
		reasons = append(reasons, fmt.Sprintf("About %d%% of the lines looked like page numbers, captions or running headers and were removed, so the text may have extraction leftovers.", int(math.Round(noiseShare*100))))
	}
	return reasons
}

// ReadingCaveat is the one plain sentence shown at the top of every report.
// The accuracy figure comes from the offline evaluation described under
// Calculation details; the wording lives here so no renderer restates it.
const ReadingCaveat = "These are rough estimates from counting word patterns. In a test on 2,442 essays, these scores matched people's questionnaire answers only slightly better than a coin flip. They cannot tell you whether someone is honest, reliable or a good hire, and should not be used for hiring, clinical or other decisions about a person."

// FormatCount formats 10785 as "10,785" for user-facing copy.
func FormatCount(n int) string {
	s := strconv.Itoa(n)
	if n < 1000 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
