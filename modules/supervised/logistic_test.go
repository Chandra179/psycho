package supervised

import (
	"math"
	"testing"
)

func TestObjectiveGradientAndUnpenalizedIntercept(t *testing.T) {
	x := [][]float64{{-2, 1}, {1, 0}, {2, 4}}
	y := []float64{0, 1, .7}
	b := []float64{.3, -.8, .4}
	g := make([]float64, 3)
	logisticObjective(b, x, y, .1, g)
	for j := range b {
		h := 1e-6
		b[j] += h
		hi := logisticObjective(b, x, y, .1, nil)
		b[j] -= 2 * h
		lo := logisticObjective(b, x, y, .1, nil)
		b[j] += h
		if math.Abs(g[j]-(hi-lo)/(2*h)) > 1e-7 {
			t.Fatalf("gradient %d: %g", j, g[j])
		}
	}
	without := logisticObjective([]float64{0, 0, 3}, x, y, 0, nil)
	with := logisticObjective([]float64{0, 0, 3}, x, y, 10, nil)
	if without != with {
		t.Fatal("intercept was penalized")
	}
}

func TestStableNumericalExtremes(t *testing.T) {
	for _, f := range []float64{-1000, -100, 0, 100, 1000} {
		if !finite(softplus(f)) || !finite(sigmoid(f)) || sigmoid(f) < 0 || sigmoid(f) > 1 {
			t.Fatal("unstable", f)
		}
		for _, target := range []float64{0, 1} {
			if !finite(logisticObjective([]float64{f}, [][]float64{{}}, []float64{target}, 0, nil)) {
				t.Fatal("nonfinite objective", f, target)
			}
		}
	}
	if binaryLoss(1000, true) != 0 || binaryLoss(-1000, false) != 0 || binaryLoss(1000, false) != 1000 {
		t.Fatal("extreme log loss incorrect")
	}
}

func TestStandardizerFitsOnlySuppliedRows(t *testing.T) {
	s := fitStandardizer([][]float64{{1, 7}, {3, 7}})
	if s.Mean[0] != 2 || s.Scale[0] != 1 || s.Scale[1] != 0 {
		t.Fatal(s)
	}
	z := s.transform([]float64{1000, 1000})
	if z[0] != 998 || z[1] != 0 || s.Mean[0] != 2 {
		t.Fatal("held-out values affected scaling", s, z)
	}
}

func TestLogisticRegularizationDeterminismAndFailure(t *testing.T) {
	x := [][]float64{{-3, 7}, {-2, 7}, {-1, 7}, {1, 7}, {2, 7}, {3, 7}}
	y := []bool{false, false, false, true, true, true}
	weak, err := fitLogistic(x, y, .001, 2000)
	if err != nil {
		t.Fatal(err)
	}
	strong, err := fitLogistic(x, y, 10, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if weak.Weights[0] <= strong.Weights[0] || weak.Weights[1] != 0 || weak.logit(x[0]) >= weak.logit(x[5]) {
		t.Fatal("fit or regularization incorrect", weak, strong)
	}
	again, err := fitLogistic(x, y, .001, 2000)
	if err != nil || hashJSON(weak) != hashJSON(again) {
		t.Fatal("fit is nondeterministic", err)
	}
	failed, err := fitLogistic(x, y, .001, 1)
	if err == nil || failed.Fit.Status != "IterationLimit" {
		t.Fatal("iteration limit not reported", failed, err)
	}
	for _, bad := range [][][]float64{{{1}, {2, 3}}, {{math.NaN()}, {1}}, nil} {
		if _, err := fitLogistic(bad, y, .1, 2000); err == nil {
			t.Fatal("accepted invalid features")
		}
	}
	if _, _, err := optimizeLogistic([][]float64{{1}, {2}}, []float64{1, 1}, .1, 2000); err == nil {
		t.Fatal("accepted one class")
	}
}

func TestCalibrationIndependentSoftTargetsAndDegenerate(t *testing.T) {
	logits := []float64{-3, -2, -1, 1, 2, 3}
	y := []bool{false, false, false, true, true, true}
	c := fitCalibration(logits, y, 2000)
	if c.Status != "fitted" || c.Slope == nil || *c.Slope <= 0 {
		t.Fatal(c)
	}
	if c.Fit.GradientInfinity == nil || *c.Fit.GradientInfinity > 1e-8 {
		t.Fatal(c)
	}
	if c := fitCalibration([]float64{1, 1}, []bool{true, false}, 2000); c.Status != "degenerate" || c.Reason == "" {
		t.Fatal(c)
	}
	if c := fitCalibration(logits, y, 1); c.Status != "failed" || c.Reason == "" {
		t.Fatal("failure hidden", c)
	}
	if c := fitCalibration([]float64{1, 2}, []bool{true, true}, 2000); c.Status != "degenerate" {
		t.Fatal(c)
	}
}
