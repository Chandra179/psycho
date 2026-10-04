package supervised

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"

	"psycho/modules/analyze"
)

type Protocol struct {
	Seed              int64     `json:"seed"`
	MinWords          int       `json:"minimum_normalized_tokens"`
	Folds             int       `json:"tuning_folds"`
	Lambdas           []float64 `json:"lambda_grid"`
	BootstrapSamples  int       `json:"bootstrap_samples"`
	GradientTolerance float64   `json:"gradient_infinity_tolerance"`
	MaxIterations     int       `json:"maximum_iterations"`
	PartitionMethod   string    `json:"partition_method"`
	CalibrationMethod string    `json:"calibration_method"`
}

func DefaultProtocol() Protocol {
	return Protocol{Seed: 42, MinWords: 200, Folds: 5, Lambdas: []float64{.0001, .001, .01, .1, 1, 10}, BootstrapSamples: 5000, GradientTolerance: 1e-8, MaxIterations: 2000, PartitionMethod: "sort SHA256(seed:author); first floor(0.6*n) fit, next floor(0.2*n) calibration, remainder test", CalibrationMethod: "sigmoid on independent logits; Platt softened targets; no regularization"}
}

func (p Protocol) validate() error {
	d := DefaultProtocol()
	if p.MinWords < 1 || p.BootstrapSamples < 2 || p.MaxIterations < 1 || p.Folds != d.Folds || p.GradientTolerance != d.GradientTolerance || !slices.Equal(p.Lambdas, d.Lambdas) || p.PartitionMethod != d.PartitionMethod || p.CalibrationMethod != d.CalibrationMethod {
		return fmt.Errorf("unsupported or invalid training protocol")
	}
	return nil
}

type TrainedTrait struct {
	Model       LogisticModel `json:"model"`
	Calibration Calibration   `json:"calibration"`
}

type Artifact struct {
	Version           int                     `json:"version"`
	Target            string                  `json:"target"`
	Provenance        string                  `json:"provenance_status"`
	CorpusChecksum    string                  `json:"corpus_sha256"`
	DictionaryHash    string                  `json:"dictionary_sha256"`
	PreprocessingHash string                  `json:"preprocessing_sha256"`
	TrainingHash      string                  `json:"training_rules_sha256"`
	PartitionHash     string                  `json:"partition_sha256"`
	FeatureNames      []string                `json:"feature_names"`
	Protocol          Protocol                `json:"protocol"`
	Traits            map[string]TrainedTrait `json:"traits"`
	Fingerprint       string                  `json:"artifact_sha256"`
}

func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func trainingFingerprint() string {
	return hashJSON(map[string]any{"version": 1, "objective": "mean softplus(f)-y*f + lambda/2*sum(weights^2); unpenalized intercept", "scaling": "population standard deviation; Welford; zero-variance features become zero; fit partitions only", "solver": "gonum v0.17.0 LBFGS; single worker; zero weights, logit(prevalence) intercept; gradient infinity norm <=1e-8; no function-only stopping", "calibration": "independent Platt softened targets: positive=(npos+1)/(npos+2), negative=1/(nneg+2); sigmoid(a*f+c)", "prediction": "full-precision stable sigmoid; no score clamp or display rounding"})
}

func featureNames(dict analyze.Dictionary) []string {
	names := make([]string, 0, len(dict.Categories()))
	for _, cat := range dict.Categories() {
		names = append(names, string(cat))
	}
	sort.Strings(names)
	return names
}

func (a Artifact) contentFingerprint() string { a.Fingerprint = ""; return hashJSON(a) }

func validHash(h string) bool {
	b, err := hex.DecodeString(h)
	return err == nil && len(b) == sha256.Size
}

func validFit(f FitStatus) bool {
	return f.Status == "GradientThreshold" && f.Iterations >= 0 && f.Objective != nil && f.GradientInfinity != nil && finite(*f.Objective) && finite(*f.GradientInfinity) && *f.GradientInfinity >= 0 && *f.GradientInfinity <= 1e-8
}

// Validate binds the artifact to today's feature semantics and dictionary.
// Corrupt/mismatched artifacts return errors; there is no heuristic fallback.
func (a Artifact) Validate(dictionary []byte) error {
	if a.Version != 1 || a.Target != "positive questionnaire label y in the local Essays CSV" || a.Provenance != "unverified local dataset provenance and redistribution terms" {
		return fmt.Errorf("unsupported artifact metadata")
	}
	if err := a.Protocol.validate(); err != nil {
		return err
	}
	dict, err := analyze.LoadDictionaryFromJSON(dictionary)
	if err != nil {
		return err
	}
	if a.DictionaryHash != analyze.DictionaryFingerprint(dictionary) || a.PreprocessingHash != analyze.FeatureProcessingFingerprint() || a.TrainingHash != trainingFingerprint() {
		return fmt.Errorf("artifact dictionary or processing fingerprint mismatch")
	}
	if !validHash(a.CorpusChecksum) || !validHash(a.PartitionHash) || a.Fingerprint != a.contentFingerprint() {
		return fmt.Errorf("invalid artifact integrity or provenance hashes")
	}
	if len(a.FeatureNames) == 0 || !slices.Equal(a.FeatureNames, featureNames(dict)) || len(a.Traits) != len(Traits()) {
		return fmt.Errorf("invalid artifact feature order or trait dimensions")
	}
	for _, trait := range Traits() {
		t, ok := a.Traits[trait.Name]
		if !ok {
			return fmt.Errorf("missing trait %s", trait.Name)
		}
		m := t.Model
		n := len(a.FeatureNames)
		if len(m.Weights) != n || len(m.Standardizer.Mean) != n || len(m.Standardizer.Scale) != n || !finite(m.Intercept) || !slices.Contains(a.Protocol.Lambdas, m.Lambda) || !validFit(m.Fit) {
			return fmt.Errorf("invalid model %s", trait.Name)
		}
		for j, w := range m.Weights {
			if !finite(w) || !finite(m.Standardizer.Mean[j]) || !finite(m.Standardizer.Scale[j]) || m.Standardizer.Scale[j] < 0 || (m.Standardizer.Scale[j] == 0 && w != 0) {
				return fmt.Errorf("invalid standardized coefficients for %s", trait.Name)
			}
		}
		c := t.Calibration
		if c.Status == "fitted" {
			if c.Slope == nil || c.Intercept == nil || !finite(*c.Slope) || !finite(*c.Intercept) || !validFit(c.Fit) {
				return fmt.Errorf("invalid calibration for %s", trait.Name)
			}
		} else if c.Status != "degenerate" || c.Reason == "" || c.Slope != nil || c.Intercept != nil {
			return fmt.Errorf("missing or failed calibration for %s", trait.Name)
		}
	}
	return nil
}

func LoadArtifact(path string, dictionary []byte) (Artifact, error) {
	f, err := os.Open(path)
	if err != nil {
		return Artifact{}, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	var a Artifact
	if err := d.Decode(&a); err != nil {
		return a, err
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return a, fmt.Errorf("artifact contains trailing JSON")
	}
	return a, a.Validate(dictionary)
}

type Contribution struct {
	Category          string  `json:"category"`
	Percentage        float64 `json:"percentage"`
	Mean              float64 `json:"training_mean"`
	Scale             float64 `json:"training_scale"`
	Standardized      float64 `json:"standardized_value"`
	Weight            float64 `json:"weight"`
	LogitContribution float64 `json:"logit_contribution"`
}

type Prediction struct {
	Intercept             float64        `json:"intercept"`
	Contributions         []Contribution `json:"contributions"`
	RawLogit              float64        `json:"raw_logit"`
	Probability           float64        `json:"uncalibrated_probability"`
	CalibratedLogit       *float64       `json:"calibrated_logit"`
	CalibratedProbability *float64       `json:"calibrated_probability"`
	CalibrationStatus     string         `json:"calibration_status"`
}

// Predict is an offline replay seam. Callers must first load/validate the
// artifact; missing feature values are errors, including explicit zeroes.
func (a Artifact) Predict(trait string, percentages map[string]float64) (Prediction, error) {
	t, ok := a.Traits[trait]
	n := len(a.FeatureNames)
	if !ok || len(percentages) != n || len(t.Model.Weights) != n || len(t.Model.Standardizer.Mean) != n || len(t.Model.Standardizer.Scale) != n {
		return Prediction{}, fmt.Errorf("invalid trait or prediction dimensions")
	}
	p := Prediction{Intercept: t.Model.Intercept, RawLogit: t.Model.Intercept, CalibrationStatus: t.Calibration.Status}
	for j, name := range a.FeatureNames {
		value, present := percentages[name]
		if !present || !finite(value) || value < 0 || value > 100 {
			return Prediction{}, fmt.Errorf("missing or invalid percentage for %s", name)
		}
		s := t.Model.Standardizer
		z := 0.0
		if s.Scale[j] > 0 {
			z = (value - s.Mean[j]) / s.Scale[j]
		}
		contribution := t.Model.Weights[j] * z
		p.Contributions = append(p.Contributions, Contribution{name, value, s.Mean[j], s.Scale[j], z, t.Model.Weights[j], contribution})
		p.RawLogit += contribution
	}
	if !finite(p.RawLogit) {
		return Prediction{}, fmt.Errorf("nonfinite prediction logit")
	}
	p.Probability = sigmoid(p.RawLogit)
	if t.Calibration.Status == "fitted" {
		if t.Calibration.Slope == nil || t.Calibration.Intercept == nil {
			return Prediction{}, fmt.Errorf("missing calibration parameters")
		}
		f := *t.Calibration.Slope*p.RawLogit + *t.Calibration.Intercept
		if !finite(f) {
			return Prediction{}, fmt.Errorf("nonfinite calibrated logit")
		}
		v := sigmoid(f)
		p.CalibratedLogit = &f
		p.CalibratedProbability = &v
	}
	return p, nil
}
