// Package report renders Psycho's single HTML report (the "warm cards"
// design): a friendly reading with the statistical receipts collapsed into
// an expandable section. It serves two shapes from the same builder: an HTML
// fragment for HTMX to swap into the upload page, and a full standalone
// page for no-JS fallback and the CLI.
package report

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"path/filepath"
	"sort"
	"time"

	"psycho/modules/analyze"
)

// Analysis is the normalized input. Its JSON shape matches the /analyze and
// /analyze-dir responses, so an AnalysisOutput round-trips through JSON
// into this struct.
type Analysis struct {
	AnalysisID         string              `json:"analysis_id"`
	WordCount          int                 `json:"word_count"`
	DictionaryCoverage float64             `json:"dictionary_coverage"`
	ConfidenceFlag     string              `json:"confidence_flag"`
	Traits             map[string]Trait    `json:"traits"`
	Values             map[string]float64  `json:"values"`
	ValueEvidence      map[string][]string `json:"value_evidence"`
	Summary            SummaryVariables    `json:"summary"`
	Narrative          string              `json:"narrative"`
}

type Trait struct {
	Score              float64            `json:"score"`
	Percentile         int                `json:"percentile"`
	ConfidenceInterval []float64          `json:"confidence_interval"`
	Evidence           []ContributionJSON `json:"evidence"`
}

type ContributionJSON struct {
	Category     string  `json:"category"`
	WordPercent  float64 `json:"word_percent"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
}

type SummaryVariables struct {
	AnalyticalThinking float64 `json:"analytical_thinking"`
	Clout              float64 `json:"clout"`
	Authenticity       float64 `json:"authenticity"`
	EmotionalTone      float64 `json:"emotional_tone"`
}

var traitOrder = []string{
	"openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism",
	"regulatory_focus", "need_for_cognition", "cognitive_style", "need_for_closure",
}

var traitBlurbs = map[string][2]string{
	"openness":           {"Curious, drawn to new ideas and experiences, enjoys abstract thinking", "Prefers the familiar and practical, more concrete and routine-oriented"},
	"conscientiousness":  {"Organized, disciplined, plans ahead, follows through", "Flexible and spontaneous, less bound by structure"},
	"extraversion":       {"Outgoing, energized by social interaction, assertive", "Reserved, prefers solitude or small groups, low-key"},
	"agreeableness":      {"Cooperative, trusting, considerate of others' needs", "Competitive, skeptical, prioritizes own interests"},
	"neuroticism":        {"Prone to worry, more reactive to stress, emotionally sensitive", "Emotionally stable, calm under pressure, resilient"},
	"regulatory_focus":   {"Promotion-focused — pursues gains, ideals, and opportunities", "Prevention-focused — avoids losses, prioritizes safety and duty"},
	"need_for_cognition": {"Enjoys effortful thinking, seeks out complex problems", "Prefers simple, quick answers over deep deliberation"},
	"cognitive_style":    {"Analytical — breaks things down, reasons step by step", "Intuitive — relies on gut feel and holistic impressions"},
	"need_for_closure":   {"Prefers clear answers, uncomfortable with ambiguity, decides quickly", "Comfortable with open questions, willing to keep deliberating"},
}

func readingQuality(flag string) string {
	switch flag {
	case "low":
		return "Low"
	case "high":
		return "High"
	default:
		return "Medium"
	}
}

// --- Report view (Design A: warm cards) ---

type TraitView struct {
	Name       string
	Label      string
	ChipClass  string
	Score100   int
	Percentile int
	HasCI      bool
	CILo       int
	CIHi       int
	Blurb      string
	Evidence   []ContributionJSON
}

type ValueView struct {
	Rank     int
	Name     string
	Percent  float64
	RelWidth int // bar width relative to the top value, 0–100
	Words    []string
}

type SummaryCard struct {
	Name  string
	Label string
}

type ReportView struct {
	GeneratedAt string
	Quality     string
	Snapshot    string
	Traits      []TraitView
	Values      []ValueView
	Summary     []SummaryCard
	WordCount   int
	Coverage    int
	AnalysisID  string
}

// chipClasses maps a trait's band label to Tailwind chip colors.
var chipClasses = map[string]string{
	"high":             "bg-teal-50 text-teal-700",
	"promotion_focus":  "bg-teal-50 text-teal-700",
	"systematic":       "bg-teal-50 text-teal-700",
	"moderate":         "bg-amber-50 text-amber-700",
	"balanced":         "bg-sky-50 text-sky-700",
	"mixed":            "bg-violet-50 text-violet-700",
	"low":              "bg-rose-50 text-rose-700",
	"prevention_focus": "bg-rose-50 text-rose-700",
	"intuitive":        "bg-rose-50 text-rose-700",
}

// BuildReport assembles the single report view from a normalized analysis.
// Traits missing from the map are skipped; the snapshot is generated from
// the actual scores.
func BuildReport(a *Analysis) ReportView {
	v := ReportView{
		Quality:    readingQuality(a.ConfidenceFlag),
		Snapshot:   generateSnapshot(a),
		WordCount:  a.WordCount,
		Coverage:   int(math.Round(a.DictionaryCoverage * 100)),
		AnalysisID: a.AnalysisID,
	}

	maxPct := 0.0
	for _, val := range a.Values {
		if val > maxPct {
			maxPct = val
		}
	}
	keys := make([]string, 0, len(a.Values))
	for k := range a.Values {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return a.Values[keys[i]] > a.Values[keys[j]] })
	for i, k := range keys {
		val := a.Values[k]
		if val <= 0 {
			continue
		}
		v.Values = append(v.Values, ValueView{
			Rank:     i + 1,
			Name:     analyze.ValueDisplayName(analyze.ValueCategory(k)),
			Percent:  round2(val),
			RelWidth: int(val / maxPct * 100),
			Words:    a.ValueEvidence[k],
		})
	}

	v.Summary = []SummaryCard{
		{"Analytical thinking", summaryLabel(a.Summary.AnalyticalThinking, "analytical", "intuitive")},
		{"Clout", summaryLabel(a.Summary.Clout, "confident", "reserved")},
		{"Authenticity", summaryLabel(a.Summary.Authenticity, "personal", "guarded")},
		{"Emotional tone", toneLabel(a.Summary.EmotionalTone)},
	}

	for _, k := range traitOrder {
		t, ok := a.Traits[k]
		if !ok {
			continue
		}
		b := traitBlurbs[k]
		label := analyze.DimensionLabel(k, t.Score)
		tv := TraitView{
			Name:       analyze.DimensionDisplayName(k),
			Label:      label,
			ChipClass:  chipClasses[label],
			Score100:   int(t.Score * 100),
			Percentile: t.Percentile,
			Blurb:      b[0],
			Evidence:   t.Evidence,
		}
		if t.Score < 0.5 {
			tv.Blurb = b[1]
		}
		if len(t.ConfidenceInterval) == 2 {
			tv.HasCI = true
			tv.CILo = int(round2(t.ConfidenceInterval[0] * 100))
			tv.CIHi = int(round2(t.ConfidenceInterval[1] * 100))
		}
		if len(tv.Evidence) > 3 {
			tv.Evidence = tv.Evidence[:3]
		}
		v.Traits = append(v.Traits, tv)
	}
	return v
}

// generateSnapshot writes the short prose summary shown at the top of the
// report: the strongest trait signal, the voice, and the top value — all
// from the actual numbers.
func generateSnapshot(a *Analysis) string {
	s := a.Summary

	var top TraitView
	topPct := -1
	for _, k := range traitOrder {
		t, ok := a.Traits[k]
		if !ok || t.Percentile <= topPct {
			continue
		}
		topPct = t.Percentile
		top = TraitView{Name: analyze.DimensionDisplayName(k), Percentile: t.Percentile}
	}

	out := fmt.Sprintf("The strongest signal in this piece of writing is %s — higher than %d%% of people.",
		top.Name, top.Percentile)
	out += fmt.Sprintf(" The voice reads %s (%.2f), comes across %s (clout %.2f), and leans %s (%.2f).",
		bandPhrase(s.Authenticity, "personal and honest", "even-keeled", "guarded and distant"), s.Authenticity,
		bandPhrase(s.Clout, "confident and dominant", "measured", "reserved rather than dominant"), s.Clout,
		bandPhrase(s.AnalyticalThinking, "toward deliberate analysis", "between analysis and intuition", "toward intuition"), s.AnalyticalThinking)

	if tv := topValue(a); tv.Name != "" {
		if len(tv.Words) > 0 {
			out += fmt.Sprintf(" %s dominates your values, showing up in words like \u201c%s\u201d.",
				tv.Name, tv.Words[0])
		} else {
			out += fmt.Sprintf(" %s dominates your values.", tv.Name)
		}
	}
	return out
}

func topValue(a *Analysis) ValueView {
	max := -1.0
	var maxKey string
	for k, val := range a.Values {
		if val > max {
			max = val
			maxKey = k
		}
	}
	if maxKey == "" || max <= 0 {
		return ValueView{}
	}
	return ValueView{
		Name:  analyze.ValueDisplayName(analyze.ValueCategory(maxKey)),
		Words: a.ValueEvidence[maxKey],
	}
}

func round2(f float64) float64 {
	return float64(int(f*100)) / 100
}

func summaryLabel(score float64, high, low string) string {
	switch analyze.HighModerateLow(score) {
	case "high":
		return high
	case "low":
		return low
	}
	return "moderate"
}

// bandPhrase picks wording for each band of a [0,1] score.
func bandPhrase(score float64, high, mid, low string) string {
	switch analyze.HighModerateLow(score) {
	case "high":
		return high
	case "low":
		return low
	}
	return mid
}

func toneLabel(score float64) string {
	if analyze.HighModerateLow(score) == "high" {
		return "positive"
	}
	if analyze.HighModerateLow(score) == "low" {
		return "negative"
	}
	return "neutral"
}

// RenderAnalysis renders the report to w. With fullPage true it emits a
// complete standalone HTML document (no-JS fallback, CLI); otherwise just
// the report fragment for HTMX to swap into the upload page.
func RenderAnalysis(templatesDir string, a *Analysis, w io.Writer, fullPage bool) error {
	v := BuildReport(a)
	v.GeneratedAt = time.Now().Format("January 2, 2006")
	if fullPage {
		tpl, err := template.ParseFiles(
			filepath.Join(templatesDir, "report-page.html"),
			filepath.Join(templatesDir, "report.html"),
		)
		if err != nil {
			return fmt.Errorf("parse templates: %w", err)
		}
		return tpl.ExecuteTemplate(w, "report-page", v)
	}
	tpl, err := template.ParseFiles(filepath.Join(templatesDir, "report.html"))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	return tpl.ExecuteTemplate(w, "report-body", v)
}
