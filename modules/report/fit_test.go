package report

import (
	"sort"
	"strings"
	"testing"

	"psycho/modules/analyze"
)

// fitAnalysis builds an analysis with the signals FitNotes reads. longWord < 0
// omits the recorded long-word share; pos < 0 omits calculation details.
func fitAnalysis(words int, authenticity, longWord float64, pos, neg int) *Analysis {
	a := &Analysis{WordCount: words, DictionaryCoverage: 0.7, Summary: SummaryVariables{Authenticity: authenticity}}
	if pos < 0 {
		return a
	}
	inputs := map[string]float64{}
	if longWord >= 0 {
		inputs["long_word_ratio"] = longWord
	}
	a.CalculationDetails = &analyze.CalculationDetails{
		CategoryCounts: map[analyze.Category]int{"positive_emotion": pos, "negative_emotion": neg},
		Summary:        map[string]analyze.SummaryCalculation{"authenticity": {Inputs: inputs}},
	}
	return a
}

func fitKeys(a *Analysis) []string {
	_, by := FitNotes(a)
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestFitNotesFlagsFormalProseAndSparseEmotion(t *testing.T) {
	// Values mirror measured samples: research-abstract, ordinary blog, remote-work-genz.
	cases := []struct {
		name string
		a    *Analysis
		want string
	}{
		{"abstract: formal and no emotion words", fitAnalysis(1076, 0.0, 52.5, 0, 0), "authenticity clout cognitive_style emotional_tone neuroticism openness"},
		{"article with a few emotion words: formal only", fitAnalysis(1133, 0.05, 28.2, 4, 4), "authenticity clout cognitive_style openness"},
		{"formal at the edge of the gap", fitAnalysis(647, 0.13, 30.0, 6, 6), "authenticity clout cognitive_style openness"},
		{"ordinary blog post", fitAnalysis(886, 0.88, 14.4, 11, 10), ""},
		{"diary entry", fitAnalysis(527, 0.91, 14.8, 4, 11), ""},
		{"terse text: low authenticity but few long words", fitAnalysis(400, 0.05, 10, 8, 8), ""},
		{"sparse emotion only", fitAnalysis(1000, 0.7, 15, 1, 3), "emotional_tone neuroticism"},
		{"emotion share just above the cut", fitAnalysis(1000, 0.7, 15, 3, 3), ""},
		{"legacy analysis, formal by authenticity alone", fitAnalysis(900, 0.01, -1, -1, 0), "authenticity clout cognitive_style openness"},
		{"legacy analysis, ordinary", fitAnalysis(900, 0.6, -1, -1, 0), ""},
		{"long-word share not recorded", fitAnalysis(900, 0.05, -1, 6, 6), "authenticity clout cognitive_style openness"},
	}
	for _, c := range cases {
		got := strings.Join(fitKeys(c.a), " ")
		if got != c.want {
			t.Errorf("%s: flagged %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFitNotesTopSentencesMatchFlags(t *testing.T) {
	top, by := FitNotes(fitAnalysis(1076, 0.0, 52.5, 0, 0))
	if len(top) != 2 || len(by) != 6 {
		t.Fatalf("top=%v by=%v", top, by)
	}
	if top, by := FitNotes(fitAnalysis(886, 0.88, 14.4, 11, 10)); top != nil || by != nil {
		t.Fatalf("ordinary text must have no fit notes: %v %v", top, by)
	}
	for k, reason := range fitReasons {
		if reason == "" || strings.ContainsRune(reason, '—') {
			t.Errorf("fit reason for %s is empty or has an em-dash", k)
		}
	}
	for _, s := range top {
		if strings.ContainsRune(s, '—') {
			t.Errorf("top note has an em-dash: %q", s)
		}
	}
}

func TestFitNotesReachCardsGlanceAndPage(t *testing.T) {
	a := testAnalysis()
	a.Summary.Authenticity = 0.01
	a.CalculationDetails = fitAnalysis(570, 0.01, 40, 0, 0).CalculationDetails
	v := BuildReport(a)
	flagged := map[string]bool{}
	for _, tv := range v.Traits {
		if tv.FitNote != "" {
			flagged[tv.Key] = true
		}
	}
	for _, c := range v.Summary {
		if c.FitNote != "" {
			flagged[c.Key] = true
		}
	}
	for _, k := range append(append([]string{}, formalKeys...), sparseEmotionKeys...) {
		if k != "neuroticism" && !flagged[k] {
			t.Errorf("%s card has no fit note", k)
		}
	}
	if len(v.Glance.FitNotes) != 2 {
		t.Fatalf("glance fit notes = %v", v.Glance.FitNotes)
	}
	var out strings.Builder
	if err := RenderAnalysis("../../templates", a, &out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Low fit for this text.") || !strings.Contains(out.String(), "It says nothing about honesty.") {
		t.Fatal("flagged cards must show the visible low-fit note")
	}

	ordinary := testAnalysis()
	ordinary.Summary.Authenticity = 0.8
	var plain strings.Builder
	if err := RenderAnalysis("../../templates", ordinary, &plain, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.String(), "Low fit for this text.") {
		t.Fatal("an ordinary text must not show low-fit notes")
	}
}
