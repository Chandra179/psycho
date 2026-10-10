package report

import (
	"strings"
	"testing"

	"psycho/modules/analyze"
)

func renderVariant(t *testing.T, a *Analysis, variant string) string {
	t.Helper()
	var out strings.Builder
	var err error
	switch variant {
	case "fragment":
		err = RenderAnalysis("../../templates", a, &out, false)
	case "full page":
		err = RenderAnalysis("../../templates", a, &out, true)
	default:
		err = RenderStandaloneAnalysis("../../templates", a, &out)
	}
	if err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestStandalonePagesHaveMainLandmarkButFragmentDoesNot(t *testing.T) {
	a := testAnalysis()
	for _, variant := range []string{"full page", "standalone"} {
		if got := strings.Count(renderVariant(t, a, variant), "<main>"); got != 1 {
			t.Errorf("%s has %d <main> landmarks, want 1", variant, got)
		}
	}
	// The upload page already wraps #result in <main>; the fragment must not nest another.
	if strings.Contains(renderVariant(t, a, "fragment"), "<main") {
		t.Error("fragment must not add a <main> inside the upload page's <main>")
	}
}

func TestReportMarkupKeepsReadableSizesAndTapTargets(t *testing.T) {
	// Match the markup, not the compiled CSS, which the standalone variant inlines.
	_, body, _ := strings.Cut(renderVariant(t, testAnalysis(), "fragment"), "<div class=\"w-full max-w-[60rem]")
	if strings.Contains(body, "text-[11px]") {
		t.Error("report text must be at least 12px (text-xs)")
	}
	if got := strings.Count(body, "min-h-[44px]"); got != 3 {
		t.Errorf("header actions need 44px tap targets, found %d", got)
	}
}

func TestEvidenceTableHasStackedPhoneAlternative(t *testing.T) {
	a := testAnalysis()
	a.CalculationDetails = &analyze.CalculationDetails{ModelFingerprint: "historical-model", Traits: map[string]*analyze.ScoreCalculation{
		"openness": {Baseline: .42, ContributionTotal: .123456789, ModelScore: .54, CalibrationApplied: true, CalibrationOffset: .16, FinalScore: .7, Terms: []analyze.ScoreTerm{{Category: "recorded_category", MatchedCount: 7, TotalWords: 570, WordPercent: 1.2280701754385965, Weight: .123, Contribution: .15105263157894738}}},
	}}
	out := renderVariant(t, a, "fragment")
	// The wide table is hidden under md and a stacked list shown instead, so a phone never
	// needs to scroll an 800px table. display:none keeps only one of them in the a11y tree.
	for _, want := range []string{`hidden md:block`, `md:hidden space-y-5`, `aria-label="Score evidence, one block per measure"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	stack := out[strings.Index(out, "md:hidden space-y-5"):]
	for _, want := range []string{"7 / 570", "Baseline 0.4200", "recorded_category"} {
		if !strings.Contains(stack, want) {
			t.Errorf("stacked list must carry the same recorded values as the table: missing %q", want)
		}
	}
	// The table region stays labelled and focusable for keyboard scrolling.
	if !strings.Contains(out, `role="region"`) || !strings.Contains(out, `tabindex="0"`) {
		t.Error("table scroll region lost its role or tabindex")
	}
}
