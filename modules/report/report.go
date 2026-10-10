// Package report renders Psycho's single HTML report (the Reading
// design): compact score rows with the statistical receipts collapsed into
// an expandable section. It serves two shapes from the same builder: an HTML
// fragment for HTMX to swap into the upload page, and a full standalone
// page for no-JS fallback and the CLI.
package report

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
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
	AnalysisID          string                          `json:"analysis_id"`
	WordCount           int                             `json:"word_count"`
	DictionaryCoverage  float64                         `json:"dictionary_coverage"`
	ConfidenceFlag      string                          `json:"confidence_flag"`
	Traits              map[string]Trait                `json:"traits"`
	Values              map[string]float64              `json:"values"`
	ValueEvidence       map[string][]string             `json:"value_evidence"`
	ValueExcerpts       map[string][]ingest.TextExcerpt `json:"value_excerpts,omitempty"`
	PercentileReference *ingest.PercentileReference     `json:"percentile_reference,omitempty"`
	CalculationDetails  *analyze.CalculationDetails     `json:"calculation_details,omitempty"`
	Summary             SummaryVariables                `json:"summary"`
	Narrative           string                          `json:"narrative"`
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

// CategoryLabel is the reader-facing name of the dictionary category.
func (c ContributionJSON) CategoryLabel() string { return analyze.CategoryLabel(c.Category) }

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

// --- Reading report view ---

// CardNotes are the small text lines under a score bar, shared by trait and
// summary cards through the common "report-score-row" template.
type CardNotes struct {
	Meaning   string // what the measure counts
	Detail    string // recorded counts behind the score, when available
	RangeNote string // repeatability range in plain words; says "too close to call" when it crosses a band
	FitNote   string // why this measure fits this text poorly, when it does not
}

type TraitView struct {
	Key               string
	Name              string
	Label             string
	ChipClass         string
	Score100          int
	PercentileText    string
	SignalDescription string
	CardNotes
	HasScoreRange  bool
	ScoreRangeLow  int
	ScoreRangeHigh int
	Evidence       []ContributionJSON
	Calculation    *analyze.ScoreCalculation
}

type ValueView struct {
	Key, Name, Description   string
	Percent                  float64
	HasCounts                bool
	MatchedCount, TotalWords int
	Words                    []string
	Excerpts                 []ingest.TextExcerpt
}

type SummaryCard struct {
	Key                                 string
	Name                                string
	Score100                            int
	Label, ChipClass, SignalDescription string
	CardNotes
}

type ReportView struct {
	GeneratedAt                    string
	Quality                        string
	Glance                         Glance
	Traits                         []TraitView
	HasBigFive                     bool
	Bands                          []analyze.ScoreBand
	Values                         []ValueView
	Summary                        []SummaryCard
	PercentileReferenceDescription string
	WordCount                      int
	WordCountText                  string // WordCount with thousands separators, matching the glance block
	Coverage                       int
	AnalysisID                     string
	CalculationJSON                string
	CalculationDownload            template.URL // data: link so the file works in saved offline reports too
	StandaloneCSS                  template.CSS
}

// chipClasses maps a trait's band label to Tailwind chip colors.
var chipClasses = map[string]string{
	"high":              "bg-teal-50 text-teal-700",
	"promotion_focus":   "bg-teal-50 text-teal-700",
	"systematic":        "bg-teal-50 text-teal-700",
	"moderate":          "bg-stone-100 text-stone-600",
	"balanced":          "bg-stone-100 text-stone-600",
	"mixed":             "bg-stone-100 text-stone-600",
	"high signal":       "bg-teal-50 text-teal-700",
	"moderate signal":   "bg-stone-100 text-stone-600",
	"low signal":        "bg-stone-100 text-stone-700",
	"positive language": "bg-teal-50 text-teal-700",
	"neutral language":  "bg-stone-100 text-stone-600",
	"negative language": "bg-stone-100 text-stone-700",
	"low":               "bg-stone-100 text-stone-700",
	"prevention_focus":  "bg-stone-100 text-stone-700",
	"intuitive":         "bg-stone-100 text-stone-700",
}

// BuildReport assembles the single report view from a normalized analysis.
// Traits missing from the map are skipped; all visible numbers come from
// the recorded analysis rather than rerunning a model.
func BuildReport(a *Analysis) ReportView {
	v := ReportView{
		Quality:                        readingQuality(a.ConfidenceFlag),
		Glance:                         BuildGlance(a),
		Bands:                          analyze.BigFiveBands(),
		WordCount:                      a.WordCount,
		WordCountText:                  analyze.FormatCount(a.WordCount),
		Coverage:                       int(math.Round(a.DictionaryCoverage * 100)),
		AnalysisID:                     a.AnalysisID,
		PercentileReferenceDescription: percentileReferenceDescription(a.PercentileReference),
	}

	if a.CalculationDetails != nil {
		data, err := json.MarshalIndent(a.CalculationDetails, "", "  ")
		if err == nil {
			v.CalculationJSON = string(data)
			v.CalculationDownload = template.URL("data:application/json;charset=utf-8;base64," + base64.StdEncoding.EncodeToString(data))
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
	for _, k := range keys {
		val := a.Values[k]
		if val <= 0 {
			continue
		}
		value := ValueView{
			Key:         k,
			Name:        analyze.ValueDisplayName(analyze.ValueCategory(k)),
			Description: analyze.ValueDescription(analyze.ValueCategory(k)),
			Percent:     round2(val),
			Words:       a.ValueEvidence[k],
			Excerpts:    a.ValueExcerpts[k],
		}
		if a.CalculationDetails != nil {
			if calculation, ok := a.CalculationDetails.Values[k]; ok {
				value.HasCounts, value.MatchedCount, value.TotalWords = true, calculation.MatchedCount, calculation.TotalWords
			}
		}
		v.Values = append(v.Values, value)
	}

	v.Summary = []SummaryCard{
		newSummaryCard("analytical_thinking", "Analytical thinking", a.Summary.AnalyticalThinking),
		newSummaryCard("clout", "Confident wording", a.Summary.Clout),
		newSummaryCard("authenticity", "Personal wording", a.Summary.Authenticity),
		newSummaryCard("emotional_tone", "Emotional tone", a.Summary.EmotionalTone),
	}

	if pos, neg, ok := emotionCounts(a); ok {
		for i := range v.Summary {
			if v.Summary[i].Key == "emotional_tone" {
				v.Summary[i].Detail = fmt.Sprintf("Based on %d negative-feeling and %d positive-feeling dictionary words.", neg, pos)
			}
		}
	}

	_, fit := FitNotes(a)
	for i := range v.Summary {
		v.Summary[i].FitNote = fit[v.Summary[i].Key]
	}

	for _, k := range traitOrder {
		t, ok := a.Traits[k]
		if !ok {
			continue
		}
		label := analyze.DimensionLabel(k, t.Score)
		tv := TraitView{
			Key:               k,
			Name:              analyze.DimensionDisplayName(k),
			Label:             label,
			ChipClass:         chipClasses[label],
			Score100:          int(math.Round(t.Score * 100)),
			PercentileText:    percentileText(t.Percentile, a.PercentileReference),
			SignalDescription: analyze.DimensionBandDescription(k, t.Score),
			CardNotes: CardNotes{
				Meaning: analyze.MeasureSummary(k),
				FitNote: fit[k],
			},
			Evidence: t.Evidence,
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
			tv.CardNotes.RangeNote = rangeNote(k, t.ConfidenceInterval[0], t.ConfidenceInterval[1])
		}
		v.Traits = append(v.Traits, tv)
		if slices.Contains(traitOrder[:5], k) {
			v.HasBigFive = true
		}
	}
	return v
}

// rangeNote says how far the score would likely move on another stretch of
// similar text, and flags a score whose range reaches into a neighbouring band.
func rangeNote(key string, low, high float64) string {
	lo, hi := int(math.Round(low*100)), int(math.Round(high*100))
	note := fmt.Sprintf("Another stretch of similar text would likely score %d to %d. This shows repeatability, not accuracy.", lo, hi)
	if a, b := analyze.DimensionLabel(key, low), analyze.DimensionLabel(key, high); a != b {
		note += fmt.Sprintf(" Too close to call between %s and %s.", a, b)
	}
	return note
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func percentileText(percentile int, reference *ingest.PercentileReference) string {
	return analyze.PercentileDescription(percentile, reference)
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

func newSummaryCard(key, name string, score float64) SummaryCard {
	band := analyze.SummarySignalLabel(key, score)
	return SummaryCard{
		Key:      key,
		Name:     name,
		Score100: int(math.Round(score * 100)),
		Label:    band, ChipClass: chipClasses[band],
		SignalDescription: analyze.SummarySignalDescription(key, score),
		CardNotes:         CardNotes{Meaning: analyze.MeasureSummary(key)},
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
