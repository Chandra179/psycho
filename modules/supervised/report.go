package supervised

import (
	"encoding/json"
	"fmt"
	"strings"
)

func formatMeasure(m Measure) string {
	if m.Value == nil {
		if m.Reason == "" {
			return "undefined (not evaluated)"
		}
		return "undefined (" + m.Reason + ")"
	}
	return fmt.Sprintf("%.17g", *m.Value)
}

func formatInterval(i Interval) string {
	if i.Bounds == nil {
		return "undefined (" + i.Reason + ")"
	}
	return fmt.Sprintf("[%.17g, %.17g]", i.Bounds[0], i.Bounds[1])
}

// Markdown includes full precision and embeds the complete aggregate JSON,
// including every tuning fold and failure, without any participant rows.
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprint(&b, "# Offline supervised Big Five evaluation\n\n")
	fmt.Fprintf(&b, "Target: %s. Provenance: %s.\n\n", r.Target, r.Provenance)
	fmt.Fprintf(&b, "Source rows: %d; retained: %d; excluded below %d tokens: %d. Fit/calibration/test: %d/%d/%d.\n\n", r.RawRows, r.RetainedRows, r.Protocol.MinWords, r.ExcludedShort, r.Partitions["fit"], r.Partitions["calibration"], r.Partitions["test"])
	fmt.Fprint(&b, "Probabilities describe the dataset's positive label. They are neither personality scores nor population percentiles. Production scoring remains unchanged.\n\n")
	fmt.Fprintln(&b, "| Trait | Status | Lambda | Raw AUC | Calibrated AUC | Heuristic AUC | Calibrated Brier | Baseline Brier | Calibrated log loss | Baseline log loss |\n|---|---|---|---|---|---|---|---|---|---|")
	for _, trait := range Traits() {
		t := r.Traits[trait.Name]
		raw, cal, base := t.Uncalibrated, t.Calibrated, t.Baseline
		missing := Measure{Reason: "model unavailable"}
		ra, ca, cb, bb, cl, bl := missing, missing, missing, missing, missing, missing
		if raw != nil {
			ra = raw.AUC
		}
		if cal != nil {
			ca, cb, cl = cal.AUC, cal.Brier, cal.LogLoss
		}
		if base != nil {
			bb, bl = base.Brier, base.LogLoss
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n", trait.Name, t.Status, formatMeasure(Measure{Value: t.SelectedLambda}), formatMeasure(ra), formatMeasure(ca), formatMeasure(t.HeuristicAUC), formatMeasure(cb), formatMeasure(bb), formatMeasure(cl), formatMeasure(bl))
	}
	fmt.Fprint(&b, "\n## Paired uncertainty\n\nIntervals are conditional on the fitted models. Negative Brier/log-loss differences favor the calibrated model.\n\n")
	fmt.Fprintln(&b, "| Trait | Calibrated minus heuristic AUC, 99% interval | Calibrated minus baseline Brier, 95% interval | Calibrated minus baseline log loss, 95% interval |\n|---|---|---|---|")
	for _, trait := range Traits() {
		t := r.Traits[trait.Name]
		if t.Bootstrap != nil {
			bs := t.Bootstrap
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", trait.Name, formatInterval(bs.CalibratedAUCDifference), formatInterval(bs.CalibratedBrierDifference), formatInterval(bs.CalibratedLogLossDifference))
		}
	}
	fmt.Fprint(&b, "\n## Limits\n\n")
	for _, limit := range r.Limitations {
		fmt.Fprintf(&b, "- %s\n", limit)
	}
	fmt.Fprint(&b, "\n## Reproduction\n\n```sh\ngo run ./cmd/train -csv corpus-eval/essays.csv -out testresults/supervised\n```\n\nThe report records all settings and hashes. Alternate flags must match those recorded for reproduction.\n\n")
	fmt.Fprintln(&b, "## Complete aggregate record\n\n```json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintf(&b, "aggregate serialization failed: %s\n", err)
	} else {
		b.Write(data)
		b.WriteByte('\n')
	}
	fmt.Fprintln(&b, "```")
	return b.String()
}
