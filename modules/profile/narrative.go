package profile

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"psycho/modules/analyze"
)

// NarrativeGenerator produces a human-readable narrative from recorded data.
type NarrativeGenerator interface{ GenerateSynthesis(profile Profile) string }
type TemplateNarrativeGenerator struct{}

func NewTemplateNarrativeGenerator() *TemplateNarrativeGenerator {
	return &TemplateNarrativeGenerator{}
}

func (g *TemplateNarrativeGenerator) GenerateSynthesis(p Profile) string {
	var out strings.Builder
	out.WriteString("## Your writing profile\n\nA rough estimate from word patterns, not a direct measure of personality.\n\n### Big Five text signals\n\n")
	traitLine := func(key string) {
		if t, ok := p.Traits[key]; ok {
			fmt.Fprintf(&out, "**%s:** Estimated text score: %.0f/100 (%s)\n\n", analyze.DimensionDisplayName(key), math.Round(t.Score*100), analyze.DimensionLabel(key, t.Score))
		}
	}
	for _, key := range []string{"openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism"} {
		traitLine(key)
	}
	bands := analyze.BigFiveBands()
	fmt.Fprintf(&out, "Bands: %s %s; %s %s; %s %s. These labels describe scores for this text only.\n\n", bands[0].Label, bands[0].Range, bands[1].Label, bands[1].Range, bands[2].Label, bands[2].Range)
	out.WriteString("### Additional text measures\n\nSimple word-pattern summaries made for this tool.\n\n")
	for _, key := range []string{"regulatory_focus", "need_for_cognition", "cognitive_style", "need_for_closure"} {
		traitLine(key)
	}
	if len(p.Values) > 0 {
		out.WriteString("### Value-related language\n\nHow often words from each value group appear, whether the text agrees with the value or rejects it. Percentages use all matches. The sentences are examples.\n\n")
		keys := make([]string, 0, len(p.Values))
		for key, value := range p.Values {
			if value > 0 {
				keys = append(keys, key)
			}
		}
		sort.Slice(keys, func(i, j int) bool {
			if p.Values[keys[i]] == p.Values[keys[j]] {
				return keys[i] < keys[j]
			}
			return p.Values[keys[i]] > p.Values[keys[j]]
		})
		for _, key := range keys {
			name := analyze.ValueDisplayName(analyze.ValueCategory(key))
			count := "counts not saved"
			if p.CalculationDetails != nil {
				if c, ok := p.CalculationDetails.Values[key]; ok {
					count = fmt.Sprintf("%d of %d words", c.MatchedCount, c.TotalWords)
				}
			}
			fmt.Fprintf(&out, "- **%s:** %s; %.2f%% of all words. %s.\n", name, count, p.Values[key], analyze.ValueDescription(analyze.ValueCategory(key)))
			if excerpts := p.ValueExcerpts[key]; len(excerpts) > 0 {
				for _, excerpt := range excerpts {
					fmt.Fprintf(&out, "  Example sentence: %s\n", excerpt.PlainText())
				}
			} else {
				out.WriteString("  Example sentences were not saved for this analysis.\n")
				if words := p.ValueEvidence[key]; len(words) > 0 {
					fmt.Fprintf(&out, "  Example matching words: %s\n", strings.Join(words, ", "))
				}
			}
		}
		out.WriteByte('\n')
	}
	out.WriteString("### Language summaries\n\nSimple summaries made for this tool, not official LIWC scores.\n\n")
	for _, item := range []struct {
		key, name string
		score     float64
	}{
		{"analytical_thinking", "Analytical thinking", p.Summary.AnalyticalThinking}, {"clout", "Confident wording", p.Summary.Clout},
		{"authenticity", "Personal wording", p.Summary.Authenticity}, {"emotional_tone", "Emotional tone", p.Summary.EmotionalTone},
	} {
		fmt.Fprintf(&out, "- **%s:** Estimated text score: %.0f/100 (%s)\n", item.name, math.Round(item.score*100), analyze.SummarySignalLabel(item.key, item.score))
	}
	out.WriteString("\n### Calculation details and limitations\n\nPercentiles compare scores with reference texts, not with people. The likely ranges are rough guides only.\n\n")
	for _, key := range []string{"openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism", "regulatory_focus", "need_for_cognition", "cognitive_style", "need_for_closure"} {
		if t, ok := p.Traits[key]; ok {
			fmt.Fprintf(&out, "- **%s:** %s", analyze.DimensionDisplayName(key), analyze.PercentileDescription(t.Percentile, p.PercentileReference))
			if len(t.ConfidenceInterval) == 2 {
				fmt.Fprintf(&out, " Likely range: %.0f to %.0f out of 100 (a rough guide only).", math.Round(t.ConfidenceInterval[0]*100), math.Round(t.ConfidenceInterval[1]*100))
			}
			out.WriteByte('\n')
		}
	}
	out.WriteString("\nThe tool counts words from its dictionary. It does not understand meaning, \"not\" in context, sarcasm or quotes. Example sentences give context but do not change scores.\n\nMade by Psycho. Rough word-pattern estimates, not proven personality or health measures.\n")
	return out.String()
}
