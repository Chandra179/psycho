// Package report renders Psycho's single HTML report (the "warm cards"
// design): a friendly reading with the statistical receipts collapsed into
// an expandable section. It serves two shapes from the same builder: an HTML
// fragment for HTMX to swap into the upload page, and a full standalone
// page for no-JS fallback and the CLI.
package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
)

// Analysis is the normalized input. Its JSON shape matches the /analyze and
// /analyze-dir responses, so an AnalysisOutput round-trips through JSON
// into this struct.
type Analysis struct {
	AnalysisID          string                      `json:"analysis_id"`
	WordCount           int                         `json:"word_count"`
	DictionaryCoverage  float64                     `json:"dictionary_coverage"`
	ConfidenceFlag      string                      `json:"confidence_flag"`
	Traits              map[string]Trait            `json:"traits"`
	Values              map[string]float64          `json:"values"`
	ValueEvidence       map[string][]string         `json:"value_evidence"`
	PercentileReference *ingest.PercentileReference `json:"percentile_reference,omitempty"`
	CalculationDetails  *analyze.CalculationDetails `json:"calculation_details,omitempty"`
	Summary             SummaryVariables            `json:"summary"`
	Narrative           string                      `json:"narrative"`
}

type Trait struct {
	Score              float64            `json:"score"`
	Percentile         int                `json:"percentile"`
	ConfidenceInterval []float64          `json:"confidence_interval"`
	Evidence           []ContributionJSON `json:"evidence"`
}

type ContributionJSON struct {
	MatchedCount int      `json:"matched_count,omitempty"`
	TotalWords   int      `json:"total_words,omitempty"`
	Category     string   `json:"category"`
	WordPercent  float64  `json:"word_percent"`
	Weight       float64  `json:"weight"`
	Contribution float64  `json:"contribution"`
	MatchedWords []string `json:"matched_words,omitempty"`
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
	Name              string
	Label             string
	ChipClass         string
	Score100          int
	PercentileText    string
	SignalDescription string
	HasScoreRange     bool
	ScoreRangeLow     int
	ScoreRangeHigh    int
	Evidence          []ContributionJSON
	Calculation       *analyze.ScoreCalculation
}

type ValueView struct {
	Rank     int
	Name     string
	Percent  float64
	RelWidth int // bar width relative to the top value, 0–100
	Words    []string
}

type SummaryCard struct {
	Name     string
	Score100 int
	Band     string
}

type ReportView struct {
	GeneratedAt                    string
	Quality                        string
	Snapshot                       string
	Traits                         []TraitView
	Values                         []ValueView
	Summary                        []SummaryCard
	PercentileReferenceDescription string
	WordCount                      int
	Coverage                       int
	AnalysisID                     string
	CalculationJSON                string
	StandaloneCSS                  template.CSS
}

// chipClasses maps a trait's band label to Tailwind chip colors.
var chipClasses = map[string]string{
	"high":             "bg-teal-50 text-teal-700",
	"promotion_focus":  "bg-teal-50 text-teal-700",
	"systematic":       "bg-teal-50 text-teal-700",
	"moderate":         "bg-amber-50 text-amber-700",
	"balanced":         "bg-sky-50 text-sky-700",
	"mixed":            "bg-violet-50 text-violet-700",
	"low":              "bg-stone-100 text-stone-700",
	"prevention_focus": "bg-stone-100 text-stone-700",
	"intuitive":        "bg-stone-100 text-stone-700",
}

// BuildReport assembles the single report view from a normalized analysis.
// Traits missing from the map are skipped; the snapshot is generated from
// the actual scores.
func BuildReport(a *Analysis) ReportView {
	v := ReportView{
		Quality:                        readingQuality(a.ConfidenceFlag),
		Snapshot:                       generateSnapshot(a),
		WordCount:                      a.WordCount,
		Coverage:                       int(math.Round(a.DictionaryCoverage * 100)),
		AnalysisID:                     a.AnalysisID,
		PercentileReferenceDescription: percentileReferenceDescription(a.PercentileReference),
	}

	if a.CalculationDetails != nil {
		data, err := json.MarshalIndent(a.CalculationDetails, "", "  ")
		if err == nil {
			v.CalculationJSON = string(data)
		}
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
	sort.Slice(keys, func(i, j int) bool {
		if a.Values[keys[i]] == a.Values[keys[j]] {
			return keys[i] < keys[j]
		}
		return a.Values[keys[i]] > a.Values[keys[j]]
	})
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
		newSummaryCard("Analytical thinking", a.Summary.AnalyticalThinking, false),
		newSummaryCard("Clout", a.Summary.Clout, false),
		newSummaryCard("Authenticity", a.Summary.Authenticity, false),
		newSummaryCard("Emotional tone", a.Summary.EmotionalTone, true),
	}

	for _, k := range traitOrder {
		t, ok := a.Traits[k]
		if !ok {
			continue
		}
		label := analyze.DimensionLabel(k, t.Score)
		tv := TraitView{
			Name:              analyze.DimensionDisplayName(k),
			Label:             label,
			ChipClass:         chipClasses[label],
			Score100:          int(math.Round(t.Score * 100)),
			PercentileText:    percentileText(t.Percentile, a.PercentileReference),
			SignalDescription: scoreSignalDescription(label),
			Evidence:          t.Evidence,
		}
		if a.CalculationDetails != nil {
			tv.Calculation = a.CalculationDetails.Traits[k]
			if tv.Calculation != nil {
				// Consume recorded operands. Never apply the current model to old scores.
				examples := make(map[string][]string)
				for _, row := range t.Evidence {
					examples[row.Category] = row.MatchedWords
				}
				tv.Evidence = nil
				for _, term := range tv.Calculation.Terms {
					tv.Evidence = append(tv.Evidence, ContributionJSON{Category: term.Category, WordPercent: term.WordPercent, Weight: term.Weight, Contribution: term.Contribution, MatchedCount: term.MatchedCount, TotalWords: term.TotalWords, MatchedWords: examples[term.Category]})
				}
			}
		}
		if len(t.ConfidenceInterval) == 2 {
			tv.HasScoreRange = true
			tv.ScoreRangeLow = int(math.Round(t.ConfidenceInterval[0] * 100))
			tv.ScoreRangeHigh = int(math.Round(t.ConfidenceInterval[1] * 100))
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

	var topName string
	topPercentile := 0
	topPct := -1
	for _, k := range traitOrder {
		t, ok := a.Traits[k]
		if !ok || t.Percentile <= topPct {
			continue
		}
		topPct = t.Percentile
		topName = analyze.DimensionDisplayName(k)
		topPercentile = t.Percentile
	}

	if topName == "" {
		return "No trait scores were available. See the evidence below for the word categories detected in this text."
	}
	out := fmt.Sprintf("%s has the highest relative rank in this report (%s percentile). These are estimates from word patterns in this text, not direct measurements of personality.",
		topName, analyze.Ordinal(topPercentile))
	out += fmt.Sprintf(" Language-pattern summary scores (0–100): authenticity %d, clout %d, analytical thinking %d.",
		int(math.Round(s.Authenticity*100)), int(math.Round(s.Clout*100)), int(math.Round(s.AnalyticalThinking*100)))

	if tv := topValue(a); tv.Name != "" {
		if len(tv.Words) > 0 {
			out += fmt.Sprintf(" The top value-related word category is %s; a matched-word example is \u201c%s\u201d.",
				tv.Name, tv.Words[0])
		} else {
			out += fmt.Sprintf(" The top value-related word category is %s.", tv.Name)
		}
	}
	return out
}

func topValue(a *Analysis) ValueView {
	max := -1.0
	var maxKey string
	for k, val := range a.Values {
		if val > max || (val == max && k < maxKey) {
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
	return math.Round(f*100) / 100
}

func percentileText(percentile int, reference *ingest.PercentileReference) string {
	if reference == nil {
		return fmt.Sprintf("Percentile method not recorded (%s percentile).", analyze.Ordinal(percentile))
	}
	switch reference.Method {
	case ingest.PercentileMethodEmpirical:
		return fmt.Sprintf("Higher than %d%% of scores in the configured reference texts.", percentile)
	case ingest.PercentileMethodNormalApproximation:
		return fmt.Sprintf("Model-estimated %s percentile from a normal approximation.", analyze.Ordinal(percentile))
	default:
		return fmt.Sprintf("Percentile method not recorded (%s percentile).", analyze.Ordinal(percentile))
	}
}

func percentileReferenceDescription(reference *ingest.PercentileReference) string {
	if reference == nil {
		return "Reference details were not recorded for this analysis."
	}
	switch reference.Method {
	case ingest.PercentileMethodEmpirical:
		if reference.SampleSize > 0 {
			if reference.Corpus != "" {
				return fmt.Sprintf("Percentiles compare scores with %d texts in %s. This is a comparison within that text sample, not a general-population estimate.", reference.SampleSize, reference.Corpus)
			}
			return fmt.Sprintf("Percentiles compare scores with %d texts in the configured reference sample, not a general-population estimate.", reference.SampleSize)
		}
		if reference.Corpus != "" {
			return fmt.Sprintf("Percentiles compare scores with the configured reference texts (%s), not a general-population estimate.", reference.Corpus)
		}
		return "Percentiles compare scores with the configured reference sample, not a general-population estimate."
	case ingest.PercentileMethodNormalApproximation:
		return "No empirical reference sample was configured. Percentiles use a normal approximation with a mean score of 50 and a standard deviation of 15."
	default:
		return "Reference details were not recorded for this analysis."
	}
}

func scoreSignalDescription(label string) string {
	switch label {
	case "high":
		return "The text's word pattern falls in the high band for this measure."
	case "low":
		return "The text's word pattern falls in the low band for this measure."
	case "moderate":
		return "Moderate means the model found no strong high or low signal in this text."
	case "promotion_focus":
		return "The text's word pattern tilts toward promotion-focused terms."
	case "prevention_focus":
		return "The text's word pattern tilts toward prevention-focused terms."
	case "balanced":
		return "Balanced means the score shows no strong promotion or prevention tilt."
	case "systematic":
		return "The text's word pattern falls in the systematic band for this measure."
	case "intuitive":
		return "The text's word pattern falls in the intuitive band for this measure."
	case "mixed":
		return "Mixed means the score shows no strong systematic or intuitive tilt."
	default:
		return "This is an experimental score based on word patterns in this text."
	}
}

func newSummaryCard(name string, score float64, emotionalTone bool) SummaryCard {
	band := analyze.HighModerateLow(score) + " signal"
	if emotionalTone {
		band = analyze.SummaryTone(score) + " language"
	}
	return SummaryCard{
		Name:     name,
		Score100: int(math.Round(score * 100)),
		Band:     band,
	}
}

// Template sets are parsed once per templates directory and reused; a
// running server holds exactly one process-lifetime directory.
var tplCache sync.Map // templatesDir|mode -> *template.Template

// RenderAnalysis renders the report to w. With fullPage true it emits a
// complete standalone HTML document (no-JS fallback, CLI); otherwise just
// the report fragment for HTMX to swap into the upload page.
func RenderAnalysis(templatesDir string, a *Analysis, w io.Writer, fullPage bool) error {
	return renderAnalysis(templatesDir, a, w, fullPage, false)
}

// RenderStandaloneAnalysis embeds trusted, compiled local CSS for offline exports.
func RenderStandaloneAnalysis(templatesDir string, a *Analysis, w io.Writer) error {
	return renderAnalysis(templatesDir, a, w, true, true)
}

func renderAnalysis(templatesDir string, a *Analysis, w io.Writer, fullPage, standalone bool) error {
	v := BuildReport(a)
	if standalone {
		css, err := os.ReadFile(filepath.Join(templatesDir, "..", "assets", "app.css"))
		if err != nil {
			return fmt.Errorf("read standalone styles: %w", err)
		}
		// This file is a build artifact controlled by the project, never user input.
		v.StandaloneCSS = template.CSS(css)
	}
	v.GeneratedAt = time.Now().Format("January 2, 2006")

	name, files := "report-body", []string{filepath.Join(templatesDir, "report.html")}
	if fullPage {
		name = "report-page"
		files = []string{filepath.Join(templatesDir, "report-page.html"), files[0]}
	}
	key := templatesDir + "|" + name
	tplAny, ok := tplCache.Load(key)
	if !ok {
		tpl, err := template.ParseFiles(files...)
		if err != nil {
			return fmt.Errorf("parse templates: %w", err)
		}
		tplAny, _ = tplCache.LoadOrStore(key, tpl)
	}
	return tplAny.(*template.Template).ExecuteTemplate(w, name, v)
}
