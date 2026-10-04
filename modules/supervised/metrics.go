package supervised

import (
	"math"
	"math/rand"
	"sort"
)

// Measure uses null plus a reason for undefined statistics, never a made-up 0.
type Measure struct {
	Value  *float64 `json:"value"`
	Reason string   `json:"reason,omitempty"`
}

func measured(v float64) Measure {
	if !finite(v) {
		return Measure{Reason: "statistic is undefined or nonfinite"}
	}
	return Measure{Value: &v}
}

type Interval struct {
	Bounds       *[2]float64 `json:"bounds"`
	Coverage     float64     `json:"coverage"`
	ValidSamples int         `json:"valid_samples"`
	Reason       string      `json:"reason,omitempty"`
}

type ReliabilityBin struct {
	Lower            float64 `json:"lower_inclusive"`
	Upper            float64 `json:"upper_exclusive_except_last"`
	Count            int     `json:"count"`
	Positive         int     `json:"positives"`
	MeanPrediction   Measure `json:"mean_prediction"`
	ObservedFraction Measure `json:"observed_fraction"`
}

type ProbabilityMetrics struct {
	AUC                  Measure             `json:"auc"`
	Brier                Measure             `json:"brier"`
	LogLoss              Measure             `json:"log_loss"`
	CalibrationIntercept Measure             `json:"calibration_intercept"`
	CalibrationSlope     Measure             `json:"calibration_slope"`
	DiagnosticFit        FitStatus           `json:"diagnostic_fit"`
	Reliability          []ReliabilityBin    `json:"reliability"`
	Intervals            map[string]Interval `json:"bootstrap_ci95"`
}

func binaryLoss(f float64, y bool) float64 {
	if y {
		return softplus(-f)
	}
	return softplus(f)
}

func probabilities(logits []float64) []float64 {
	p := make([]float64, len(logits))
	for i, f := range logits {
		p[i] = sigmoid(f)
	}
	return p
}

func metrics(logits []float64, y []bool, maxIterations int) ProbabilityMetrics {
	p := probabilities(logits)
	auc, brier, loss := sampleMetrics(logits, p, y)
	m := ProbabilityMetrics{AUC: measured(auc), Brier: measured(brier), LogLoss: measured(loss), Intervals: map[string]Interval{}}
	for j := 0; j < 10; j++ {
		bin := ReliabilityBin{Lower: float64(j) / 10, Upper: float64(j+1) / 10}
		sum := 0.0
		for i, v := range p {
			index := int(v * 10)
			if index == 10 {
				index = 9
			}
			if index == j {
				bin.Count++
				sum += v
				if y[i] {
					bin.Positive++
				}
			}
		}
		if bin.Count == 0 {
			bin.MeanPrediction = Measure{Reason: "empty bin"}
			bin.ObservedFraction = Measure{Reason: "empty bin"}
		} else {
			bin.MeanPrediction = measured(sum / float64(bin.Count))
			bin.ObservedFraction = measured(float64(bin.Positive) / float64(bin.Count))
		}
		m.Reliability = append(m.Reliability, bin)
	}
	intercept, slope, status, err := diagnosticCalibration(logits, y, maxIterations)
	m.DiagnosticFit = status
	if err != "" {
		m.CalibrationIntercept = Measure{Reason: err}
		m.CalibrationSlope = Measure{Reason: err}
	} else {
		m.CalibrationIntercept = measured(intercept)
		m.CalibrationSlope = measured(slope)
	}
	return m
}

// Test-set diagnostic fitting describes calibration; it never changes the
// model or its test predictions. A slope of 1 and intercept of 0 are ideal.
func diagnosticCalibration(logits []float64, y []bool, maxIterations int) (float64, float64, FitStatus, string) {
	status := FitStatus{Status: "undefined"}
	if len(y) < 2 || len(y) != len(logits) {
		return 0, 0, status, "insufficient diagnostic observations"
	}
	minP, maxP, minN, maxN := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	x := make([][]float64, len(y))
	targets := make([]float64, len(y))
	for i, f := range logits {
		x[i] = []float64{f}
		if y[i] {
			targets[i] = 1
			minP = math.Min(minP, f)
			maxP = math.Max(maxP, f)
		} else {
			minN = math.Min(minN, f)
			maxN = math.Max(maxN, f)
		}
	}
	if !finite(minP) || !finite(minN) {
		return 0, 0, status, "diagnostic requires both classes"
	}
	if math.Max(maxP, maxN)-math.Min(minP, minN) < 1e-12 {
		return 0, 0, status, "constant prediction logits"
	}
	if minP >= maxN || minN >= maxP {
		return 0, 0, status, "separated diagnostic labels; finite maximum-likelihood slope unavailable"
	}
	parameters, fit, err := optimizeLogistic(x, targets, 0, maxIterations)
	if err != nil {
		return 0, 0, fit, err.Error()
	}
	return parameters[1], parameters[0], fit, ""
}

func sampleMetrics(logits, p []float64, y []bool) (float64, float64, float64) {
	if len(y) == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	brier, loss := 0.0, 0.0
	for i, positive := range y {
		target := 0.0
		if positive {
			target = 1
		}
		d := p[i] - target
		brier += d * d
		loss += binaryLoss(logits[i], positive)
	}
	// Sigmoid is monotone mathematically; ranking logits avoids artificial
	// floating-point ties when extreme probabilities saturate at 0 or 1.
	return AUCBinary(logits, y), brier / float64(len(y)), loss / float64(len(y))
}

func quantile(sorted []float64, p float64) float64 {
	index := p * float64(len(sorted)-1)
	lo := int(index)
	hi := int(math.Ceil(index))
	return sorted[lo] + (sorted[hi]-sorted[lo])*(index-float64(lo))
}

func interval(values []float64, coverage float64) Interval {
	i := Interval{Coverage: coverage, ValidSamples: len(values)}
	if len(values) < 2 {
		i.Reason = "fewer than two finite bootstrap samples"
		return i
	}
	sort.Float64s(values)
	tail := (1 - coverage) / 2
	bounds := [2]float64{quantile(values, tail), quantile(values, 1-tail)}
	i.Bounds = &bounds
	return i
}

type Bootstrap struct {
	Uncalibrated                map[string]Interval `json:"uncalibrated_ci95"`
	Calibrated                  map[string]Interval `json:"calibrated_ci95"`
	Baseline                    map[string]Interval `json:"baseline_ci95"`
	HeuristicAUC                Interval            `json:"heuristic_auc_ci95"`
	UncalibratedAUCDifference   Interval            `json:"uncalibrated_minus_heuristic_auc_ci99"`
	CalibratedAUCDifference     Interval            `json:"calibrated_minus_heuristic_auc_ci99"`
	CalibratedBrierDifference   Interval            `json:"calibrated_minus_baseline_brier_ci95"`
	CalibratedLogLossDifference Interval            `json:"calibrated_minus_baseline_log_loss_ci95"`
}

// All methods reuse the exact same author draws for paired comparisons. There
// is one unique author per observation; model-fitting uncertainty is excluded.
func bootstrap(raw, cal, base, heuristic []float64, y []bool, draws int, seed int64) Bootstrap {
	series := map[string][]float64{}
	add := func(k string, v float64) {
		if finite(v) {
			series[k] = append(series[k], v)
		}
	}
	rng := rand.New(rand.NewSource(seed))
	n := len(y)
	for d := 0; d < draws; d++ {
		label := make([]bool, n)
		a, b, c, h := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
		for i := range label {
			j := rng.Intn(n)
			label[i] = y[j]
			a[i] = raw[j]
			c[i] = base[j]
			h[i] = heuristic[j]
			if cal != nil {
				b[i] = cal[j]
			}
		}
		ar, br, lr := sampleMetrics(a, probabilities(a), label)
		ab, bb, lb := sampleMetrics(c, probabilities(c), label)
		ah := AUCBinary(h, label)
		add("raw_auc", ar)
		add("raw_brier", br)
		add("raw_log_loss", lr)
		add("base_auc", ab)
		add("base_brier", bb)
		add("base_log_loss", lb)
		add("heuristic_auc", ah)
		add("raw_delta", ar-ah)
		if cal != nil {
			ac, bc, lc := sampleMetrics(b, probabilities(b), label)
			add("cal_auc", ac)
			add("cal_brier", bc)
			add("cal_log_loss", lc)
			add("cal_delta", ac-ah)
			add("brier_delta", bc-bb)
			add("loss_delta", lc-lb)
		}
	}
	maps := func(prefix string) map[string]Interval {
		m := map[string]Interval{}
		for _, metric := range []string{"auc", "brier", "log_loss"} {
			m[metric] = interval(series[prefix+metric], .95)
		}
		return m
	}
	return Bootstrap{Uncalibrated: maps("raw_"), Calibrated: maps("cal_"), Baseline: maps("base_"), HeuristicAUC: interval(series["heuristic_auc"], .95), UncalibratedAUCDifference: interval(series["raw_delta"], .99), CalibratedAUCDifference: interval(series["cal_delta"], .99), CalibratedBrierDifference: interval(series["brier_delta"], .95), CalibratedLogLossDifference: interval(series["loss_delta"], .95)}
}
