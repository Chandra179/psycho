package supervised

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/optimize"
)

type Standardizer struct {
	Mean  []float64 `json:"mean"`
	Scale []float64 `json:"scale"`
}

func fitStandardizer(x [][]float64) Standardizer {
	p := len(x[0])
	s := Standardizer{Mean: make([]float64, p), Scale: make([]float64, p)}
	// Welford avoids catastrophic cancellation and uses only fitting rows.
	for i, row := range x {
		for j, value := range row {
			delta := value - s.Mean[j]
			s.Mean[j] += delta / float64(i+1)
			s.Scale[j] += delta * (value - s.Mean[j])
		}
	}
	for j := range s.Scale {
		s.Scale[j] = math.Sqrt(math.Max(0, s.Scale[j]/float64(len(x))))
	}
	return s
}

func (s Standardizer) transform(row []float64) []float64 {
	z := make([]float64, len(row))
	for j, value := range row {
		if s.Scale[j] > 0 {
			z[j] = (value - s.Mean[j]) / s.Scale[j]
		}
	}
	return z
}

func sigmoid(x float64) float64 {
	if x >= 0 {
		return 1 / (1 + math.Exp(-x))
	}
	e := math.Exp(x)
	return e / (1 + e)
}

func softplus(x float64) float64 { return math.Max(x, 0) + math.Log1p(math.Exp(-math.Abs(x))) }

// logisticObjective uses mean loss. The final parameter is the unpenalized
// intercept. Softplus keeps loss finite even for extreme finite logits.
func logisticObjective(parameters []float64, x [][]float64, y []float64, lambda float64, gradient []float64) float64 {
	p := len(parameters) - 1
	if gradient != nil {
		clear(gradient)
	}
	loss := 0.0
	for i, row := range x {
		f := parameters[p]
		for j, value := range row {
			f += parameters[j] * value
		}
		// This equivalent form avoids cancellation for large positive logits.
		loss += (1-y[i])*math.Max(f, 0) + y[i]*math.Max(-f, 0) + math.Log1p(math.Exp(-math.Abs(f)))
		if gradient != nil {
			delta := (sigmoid(f) - y[i]) / float64(len(x))
			for j, value := range row {
				gradient[j] += delta * value
			}
			gradient[p] += delta
		}
	}
	loss /= float64(len(x))
	for j := 0; j < p; j++ {
		loss += lambda * parameters[j] * parameters[j] / 2
		if gradient != nil {
			gradient[j] += lambda * parameters[j]
		}
	}
	return loss
}

type FitStatus struct {
	Status           string   `json:"status"`
	Iterations       int      `json:"iterations"`
	Objective        *float64 `json:"objective"`
	GradientInfinity *float64 `json:"gradient_infinity_norm"`
}

type LogisticModel struct {
	Standardizer Standardizer `json:"standardizer"`
	Weights      []float64    `json:"weights"`
	Intercept    float64      `json:"intercept"`
	Lambda       float64      `json:"lambda"`
	Fit          FitStatus    `json:"fit"`
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func optimizeLogistic(x [][]float64, y []float64, lambda float64, maxIterations int) ([]float64, FitStatus, error) {
	status := FitStatus{Status: "invalid_input"}
	if len(x) == 0 || len(x) != len(y) || !finite(lambda) || lambda < 0 || maxIterations < 1 {
		return nil, status, fmt.Errorf("invalid optimizer inputs")
	}
	p := len(x[0])
	total := 0.0
	for i, row := range x {
		if len(row) != p || !finite(y[i]) || y[i] < 0 || y[i] > 1 {
			return nil, status, fmt.Errorf("invalid training row")
		}
		for _, v := range row {
			if !finite(v) {
				return nil, status, fmt.Errorf("nonfinite training feature")
			}
		}
		total += y[i]
	}
	prevalence := total / float64(len(y))
	if prevalence <= 0 || prevalence >= 1 {
		return nil, status, fmt.Errorf("training requires both classes")
	}
	initial := make([]float64, p+1)
	initial[p] = math.Log(prevalence / (1 - prevalence))
	problem := optimize.Problem{
		Func: func(b []float64) float64 { return logisticObjective(b, x, y, lambda, nil) },
		Grad: func(g, b []float64) { logisticObjective(b, x, y, lambda, g) },
	}
	settings := &optimize.Settings{GradientThreshold: 1e-8, MajorIterations: maxIterations, Concurrent: 1, Converger: optimize.NeverTerminate{}}
	result, err := optimize.Minimize(problem, initial, settings, &optimize.LBFGS{})
	if result != nil {
		status.Status = result.Status.String()
		status.Iterations = result.MajorIterations
		if finite(result.F) {
			objective := result.F
			status.Objective = &objective
		}
		gradientNorm := 0.0
		validGradient := len(result.Gradient) == len(initial)
		for _, v := range result.Gradient {
			if finite(v) {
				gradientNorm = math.Max(gradientNorm, math.Abs(v))
			} else {
				validGradient = false
			}
		}
		if validGradient {
			status.GradientInfinity = &gradientNorm
		}
	}
	if err != nil {
		return nil, status, fmt.Errorf("optimizer: %w", err)
	}
	if result == nil || result.Status.Early() || status.GradientInfinity == nil || *status.GradientInfinity > 1e-8 || !finite(result.F) {
		return nil, status, fmt.Errorf("optimizer did not converge: %s", status.Status)
	}
	for _, v := range result.X {
		if !finite(v) {
			return nil, status, fmt.Errorf("optimizer returned nonfinite coefficients")
		}
	}
	return result.X, status, nil
}

func fitLogistic(x [][]float64, y []bool, lambda float64, maxIterations int) (LogisticModel, error) {
	m := LogisticModel{Lambda: lambda}
	if len(x) == 0 || len(x) != len(y) {
		return m, fmt.Errorf("empty or mismatched fitting partition")
	}
	width := len(x[0])
	for _, row := range x {
		if len(row) != width {
			return m, fmt.Errorf("inconsistent feature dimensions")
		}
		for _, value := range row {
			if !finite(value) {
				return m, fmt.Errorf("nonfinite training feature")
			}
		}
	}
	m.Standardizer = fitStandardizer(x)
	z := make([][]float64, len(x))
	targets := make([]float64, len(y))
	for i, row := range x {
		z[i] = m.Standardizer.transform(row)
		if y[i] {
			targets[i] = 1
		}
	}
	parameters, status, err := optimizeLogistic(z, targets, lambda, maxIterations)
	m.Fit = status
	if err != nil {
		return m, err
	}
	m.Weights = append([]float64(nil), parameters[:len(parameters)-1]...)
	m.Intercept = parameters[len(parameters)-1]
	return m, nil
}

func (m LogisticModel) logit(row []float64) float64 {
	z := m.Standardizer.transform(row)
	f := m.Intercept
	for j, value := range z {
		f += m.Weights[j] * value
	}
	return f
}

type Calibration struct {
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Slope     *float64  `json:"slope"`
	Intercept *float64  `json:"intercept"`
	Fit       FitStatus `json:"fit"`
}

// Platt calibration fits independent logits to softened binary targets, as in
// Niculescu-Mizil & Caruana (ICML 2005). It never sees final-test labels.
func fitCalibration(logits []float64, y []bool, maxIterations int) Calibration {
	c := Calibration{Status: "failed"}
	if len(logits) == 0 || len(logits) != len(y) {
		c.Reason = "empty or mismatched calibration partition"
		return c
	}
	positive := 0
	lo, hi := logits[0], logits[0]
	for i, f := range logits {
		if !finite(f) {
			c.Reason = "nonfinite logits"
			return c
		}
		lo = math.Min(lo, f)
		hi = math.Max(hi, f)
		if y[i] {
			positive++
		}
	}
	negative := len(y) - positive
	if positive == 0 || negative == 0 || hi-lo < 1e-12 {
		c.Status = "degenerate"
		c.Reason = "calibration requires both classes and varying logits"
		return c
	}
	x := make([][]float64, len(y))
	targets := make([]float64, len(y))
	for i, f := range logits {
		x[i] = []float64{f}
		if y[i] {
			targets[i] = float64(positive+1) / float64(positive+2)
		} else {
			targets[i] = 1 / float64(negative+2)
		}
	}
	parameters, status, err := optimizeLogistic(x, targets, 0, maxIterations)
	c.Fit = status
	if err != nil {
		c.Reason = err.Error()
		return c
	}
	c.Status = "fitted"
	c.Slope = &parameters[0]
	c.Intercept = &parameters[1]
	return c
}
