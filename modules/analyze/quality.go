package analyze

import (
	"fmt"
	"strconv"
)

// Reading-quality thresholds. profile.computeConfidenceFlag and the report's
// plain-language reasons both read these, so the flag and its explanation
// cannot drift apart.
const (
	QualityLowWords    = 500
	QualityHighWords   = 1000
	QualityMinCoverage = 0.6
)

// QualityFlag classifies a sample as "low", "medium" or "high" reading
// quality: under QualityLowWords is low; low dictionary coverage or under
// QualityHighWords is medium; otherwise high.
func QualityFlag(wordCount int, coverage float64) string {
	if wordCount < QualityLowWords {
		return "low"
	}
	if coverage < QualityMinCoverage {
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
	var reasons []string
	if wordCount < QualityLowWords {
		reasons = append(reasons, fmt.Sprintf("The text is short (under %s words), so scores can swing a lot.", FormatCount(QualityLowWords)))
	} else if wordCount < QualityHighWords {
		reasons = append(reasons, fmt.Sprintf("The text is under %s words, so scores are rough.", FormatCount(QualityHighWords)))
	}
	if coverage < QualityMinCoverage {
		reasons = append(reasons, "Under 60% of the words matched the dictionary, so part of the text was not scored.")
	}
	return reasons
}

// ReadingCaveat is the one plain sentence shown at the top of every report.
// The accuracy figure comes from the offline evaluation described under
// Calculation details; the wording lives here so no renderer restates it.
const ReadingCaveat = "These are rough estimates from counting word patterns. In an offline test they ranked authors only slightly better than chance. They cannot tell you whether someone is honest, reliable or a good hire, and should not be used for hiring, clinical or other decisions about a person."

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
