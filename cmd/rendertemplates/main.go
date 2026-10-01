// One-off script: renders templates/{general,technical,balanced}.html against
// an analysis JSON (as returned by POST /analyze-dir, read from stdin) and
// writes the resulting HTML files to the current directory. Not wired into
// the server; meant to be deleted or promoted to a real profile.PDFGenerator
// later.
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"sort"
	"time"

	"psycho/modules/analyze"
)

type contributionJSON struct {
	Category     string  `json:"category"`
	WordPercent  float64 `json:"word_percent"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
}

type traitJSON struct {
	Score              float64            `json:"score"`
	Percentile         int                `json:"percentile"`
	ConfidenceInterval []float64          `json:"confidence_interval"`
	Evidence           []contributionJSON `json:"evidence"`
}

type analysisJSON struct {
	AnalysisID         string               `json:"analysis_id"`
	WordCount          int                  `json:"word_count"`
	DictionaryCoverage float64              `json:"dictionary_coverage"`
	ConfidenceFlag     string               `json:"confidence_flag"`
	Traits             map[string]traitJSON `json:"traits"`
	Values             map[string]float64   `json:"values"`
	ValueEvidence      map[string][]string  `json:"value_evidence"`
	Summary            struct {
		AnalyticalThinking float64 `json:"analytical_thinking"`
		Clout              float64 `json:"clout"`
		Authenticity       float64 `json:"authenticity"`
		EmotionalTone      float64 `json:"emotional_tone"`
	} `json:"summary"`
	Narrative string `json:"narrative"`
}

var traitOrder = []string{
	"openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism",
	"regulatory_focus", "need_for_cognition", "cognitive_style", "need_for_closure",
}

var traitNames = map[string]string{
	"openness":           "Openness",
	"conscientiousness":  "Conscientiousness",
	"extraversion":       "Extraversion",
	"agreeableness":      "Agreeableness",
	"neuroticism":        "Neuroticism",
	"regulatory_focus":   "Regulatory Focus",
	"need_for_cognition": "Need for Cognition",
	"cognitive_style":    "Cognitive Style",
	"need_for_closure":   "Need for Closure",
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

func label(score float64) string {
	if score >= 0.65 {
		return "high"
	}
	if score < 0.35 {
		return "low"
	}
	return "moderate"
}

func specificLabel(key string, score float64) string {
	switch key {
	case "regulatory_focus":
		return analyze.ComputeRegulatoryFocusLabel(score)
	case "need_for_cognition":
		return analyze.ComputeNeedForCognitionLabel(score)
	case "cognitive_style":
		return analyze.ComputeCognitiveStyleLabel(score)
	case "need_for_closure":
		return analyze.ComputeNeedForClosureLabel(score)
	}
	return label(score)
}

func summaryLabel(score float64, high, low string) string {
	if score >= 0.65 {
		return high
	}
	if score < 0.35 {
		return low
	}
	return "moderate"
}

func toneLabel(score float64) string {
	if score >= 0.65 {
		return "positive"
	}
	if score < 0.35 {
		return "negative"
	}
	return "neutral"
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

type ValueView struct {
	Rank    int
	Name    string
	Percent float64
	Words   []string
}

func sortedValues(values map[string]float64, evidence map[string][]string) []ValueView {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return values[keys[i]] > values[keys[j]] })
	out := make([]ValueView, 0, len(keys))
	for i, k := range keys {
		dn := analyze.ValueDisplayName(analyze.ValueCategory(k))
		out = append(out, ValueView{Rank: i + 1, Name: dn, Percent: round2(values[k]), Words: evidence[k]})
	}
	return out
}

func round2(f float64) float64 {
	return float64(int(f*100)) / 100
}

// --- General template view ---

type generalTrait struct {
	Name       string
	Score100   int
	Label      string
	AboveBlurb string
	BelowBlurb string
}
type generalSummary struct {
	Name  string
	Label string
}
type generalView struct {
	GeneratedAt    string
	ReadingQuality string
	Narrative      string
	Traits         []generalTrait
	Values         []ValueView
	Summary        []generalSummary
}

// --- Technical template view ---

type technicalTrait struct {
	Key        string
	Name       string
	Score      float64
	Label      string
	Percentile int
	CILow      float64
	CIHigh     float64
}
type evidenceRow struct {
	Trait        string
	Category     string
	WordPercent  float64
	Weight       float64
	Contribution float64
}
type technicalSummary struct {
	Name  string
	Score float64
	Label string
}
type technicalView struct {
	AnalysisID     string
	GeneratedAt    string
	ConfidenceFlag string
	WordCount      int
	Coverage       float64
	Traits         []technicalTrait
	Evidence       []evidenceRow
	Values         []ValueView
	Summary        []technicalSummary
}

// --- Balanced template view ---

type balancedTrait struct {
	Name       string
	Score100   int
	Label      string
	Percentile int
	Blurb      string
}
type balancedSummary struct {
	Name  string
	Label string
}
type balancedView struct {
	ReadingQuality string
	AnalysisID     string
	Traits         []balancedTrait
	Values         []ValueView
	Summary        []balancedSummary
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	var a analysisJSON
	if err := json.Unmarshal(raw, &a); err != nil {
		panic(err)
	}

	now := time.Now().Format("January 2, 2006")

	// general
	var gv generalView
	gv.GeneratedAt = now
	gv.ReadingQuality = readingQuality(a.ConfidenceFlag)
	for _, k := range traitOrder {
		t := a.Traits[k]
		b := traitBlurbs[k]
		gv.Traits = append(gv.Traits, generalTrait{
			Name: traitNames[k], Score100: int(t.Score * 100), Label: specificLabel(k, t.Score),
			AboveBlurb: b[0], BelowBlurb: b[1],
		})
	}
	gv.Values = sortedValues(a.Values, a.ValueEvidence)
	gv.Summary = []generalSummary{
		{"Analytical Thinking", summaryLabel(a.Summary.AnalyticalThinking, "highly analytical", "intuitive")},
		{"Clout", summaryLabel(a.Summary.Clout, "confident/dominant", "submissive/uncertain")},
		{"Authenticity", summaryLabel(a.Summary.Authenticity, "personal/honest", "guarded/distant")},
		{"Emotional Tone", toneLabel(a.Summary.EmotionalTone)},
	}
	renderTemplate("templates/general.html", "profile-general.html", gv)

	// technical
	var tv technicalView
	tv.AnalysisID = a.AnalysisID
	tv.GeneratedAt = now
	tv.ConfidenceFlag = a.ConfidenceFlag
	tv.WordCount = a.WordCount
	tv.Coverage = round2(a.DictionaryCoverage)
	for _, k := range traitOrder {
		t := a.Traits[k]
		ci := [2]float64{0, 0}
		if len(t.ConfidenceInterval) == 2 {
			ci[0], ci[1] = t.ConfidenceInterval[0]*100, t.ConfidenceInterval[1]*100
		}
		tv.Traits = append(tv.Traits, technicalTrait{
			Key: k, Name: traitNames[k], Score: t.Score * 100, Label: specificLabel(k, t.Score),
			Percentile: t.Percentile, CILow: round2(ci[0]), CIHigh: round2(ci[1]),
		})
		for i, c := range t.Evidence {
			if i >= 3 {
				break
			}
			tv.Evidence = append(tv.Evidence, evidenceRow{
				Trait: traitNames[k], Category: c.Category,
				WordPercent: c.WordPercent, Weight: c.Weight, Contribution: c.Contribution,
			})
		}
	}
	tv.Values = sortedValues(a.Values, a.ValueEvidence)
	tv.Summary = []technicalSummary{
		{"Analytical Thinking", a.Summary.AnalyticalThinking, summaryLabel(a.Summary.AnalyticalThinking, "highly analytical", "intuitive")},
		{"Clout", a.Summary.Clout, summaryLabel(a.Summary.Clout, "confident/dominant", "submissive/uncertain")},
		{"Authenticity", a.Summary.Authenticity, summaryLabel(a.Summary.Authenticity, "personal/honest", "guarded/distant")},
		{"Emotional Tone", a.Summary.EmotionalTone, toneLabel(a.Summary.EmotionalTone)},
	}
	renderTemplate("templates/technical.html", "profile-technical.html", tv)

	// balanced
	var bv balancedView
	bv.ReadingQuality = readingQuality(a.ConfidenceFlag)
	bv.AnalysisID = a.AnalysisID
	for _, k := range traitOrder {
		t := a.Traits[k]
		b := traitBlurbs[k]
		bv.Traits = append(bv.Traits, balancedTrait{
			Name: traitNames[k], Score100: int(t.Score * 100), Label: specificLabel(k, t.Score),
			Percentile: t.Percentile, Blurb: b[0],
		})
	}
	bv.Values = sortedValues(a.Values, a.ValueEvidence)
	bv.Summary = []balancedSummary{
		{"Analytical Thinking", summaryLabel(a.Summary.AnalyticalThinking, "highly analytical", "intuitive")},
		{"Clout", summaryLabel(a.Summary.Clout, "confident/dominant", "submissive/uncertain")},
		{"Authenticity", summaryLabel(a.Summary.Authenticity, "personal/honest", "guarded/distant")},
		{"Emotional Tone", toneLabel(a.Summary.EmotionalTone)},
	}
	renderTemplate("templates/balanced.html", "profile-balanced.html", bv)

	fmt.Println("rendered 3 html files")
}

func renderTemplate(tplPath, outPath string, data any) {
	tpl, err := template.ParseFiles(tplPath)
	if err != nil {
		panic(err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := tpl.Execute(f, data); err != nil {
		panic(err)
	}
}
