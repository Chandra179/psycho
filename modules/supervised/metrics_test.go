package supervised

import (
	"encoding/json"
	"math"
	"testing"
)

func TestProbabilityMetricsReliabilityAndUndefined(t *testing.T) {
	m := metrics([]float64{-math.Log(3), 0, 0, math.Log(3)}, []bool{false, false, true, true}, 2000)
	if m.AUC.Value == nil || *m.AUC.Value != .875 || math.Abs(*m.Brier.Value-.15625) > 1e-12 {
		t.Fatal(m)
	}
	count := 0
	for _, bin := range m.Reliability {
		count += bin.Count
		if bin.Count == 0 && (bin.MeanPrediction.Value != nil || bin.MeanPrediction.Reason == "") {
			t.Fatal("empty bin fabricated")
		}
	}
	if count != 4 {
		t.Fatal(count)
	}
	constant := metrics([]float64{0, 0}, []bool{true, false}, 2000)
	if constant.CalibrationSlope.Value != nil || constant.CalibrationSlope.Reason == "" {
		t.Fatal("constant diagnostic fabricated")
	}
	single := metrics([]float64{0, 1}, []bool{true, true}, 2000)
	if single.AUC.Value != nil || single.AUC.Reason == "" {
		t.Fatal("single-class AUC fabricated")
	}
	if _, err := json.Marshal(single); err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(AUCBinary([]float64{math.NaN(), 1}, []bool{false, true})) || !math.IsNaN(Pearson([]float64{1, 2}, []float64{1})) {
		t.Fatal("invalid metrics accepted")
	}
}

func TestPairedBootstrapAndQuantiles(t *testing.T) {
	y := []bool{false, true, false, true, false, true, false, true}
	f := []float64{-2, 2, -1, 1, .5, -.5, 0, 0}
	base := make([]float64, len(y))
	h := probabilities(f)
	b := bootstrap(f, f, base, h, y, 100, 42)
	if b.CalibratedAUCDifference.Bounds == nil || *b.CalibratedAUCDifference.Bounds != [2]float64{0, 0} {
		t.Fatal("identical rankings have nonzero paired difference")
	}
	if b.Uncalibrated["auc"].Coverage != .95 || b.CalibratedAUCDifference.Coverage != .99 {
		t.Fatal("wrong coverage")
	}
	if hashJSON(b) != hashJSON(bootstrap(f, f, base, h, y, 100, 42)) {
		t.Fatal("bootstrap not reproducible")
	}
	missing := bootstrap(f, nil, base, h, y, 10, 42)
	if missing.CalibratedAUCDifference.Bounds != nil || missing.CalibratedAUCDifference.Reason == "" {
		t.Fatal("missing calibration substituted")
	}
	if quantile([]float64{0, 10}, .25) != 2.5 {
		t.Fatal("quantile interpolation")
	}
	if i := interval(nil, .95); i.Bounds != nil || i.Reason == "" {
		t.Fatal("missing interval substituted")
	}
}

func TestAUCAvoidsSigmoidSaturationTies(t *testing.T) {
	f := []float64{-1000, -800, 800, 1000}
	y := []bool{false, true, false, true}
	m := metrics(f, y, 2000)
	if m.AUC.Value == nil || *m.AUC.Value != .75 {
		t.Fatal("saturated sigmoid destroyed ranking", m.AUC)
	}
	if m.LogLoss.Value == nil || !finite(*m.LogLoss.Value) {
		t.Fatal("extreme log loss became nonfinite")
	}
}
