package supervised

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"

	"psycho/modules/analyze"
)

type ClassCounts struct {
	Positive int `json:"positives"`
	Negative int `json:"negatives"`
}

type ValidationFold struct {
	Fit     FitStatus `json:"fit"`
	LogLoss Measure   `json:"validation_log_loss"`
	Reason  string    `json:"reason,omitempty"`
}

type Trial struct {
	Lambda      float64          `json:"lambda"`
	MeanLogLoss Measure          `json:"mean_validation_log_loss"`
	Folds       []ValidationFold `json:"folds"`
}

type TraitReport struct {
	Status                      string                 `json:"status"`
	Reason                      string                 `json:"reason,omitempty"`
	Classes                     map[string]ClassCounts `json:"partition_classes"`
	Tuning                      []Trial                `json:"tuning"`
	SelectedLambda              *float64               `json:"selected_lambda"`
	Fit                         FitStatus              `json:"fit"`
	Calibration                 Calibration            `json:"probability_calibration"`
	FittingPrevalence           *float64               `json:"fitting_prevalence"`
	Uncalibrated                *ProbabilityMetrics    `json:"uncalibrated"`
	Calibrated                  *ProbabilityMetrics    `json:"calibrated"`
	Baseline                    *ProbabilityMetrics    `json:"prevalence_baseline"`
	HeuristicAUC                Measure                `json:"heuristic_auc"`
	UncalibratedAUCDifference   Measure                `json:"uncalibrated_minus_heuristic_auc"`
	CalibratedAUCDifference     Measure                `json:"calibrated_minus_heuristic_auc"`
	CalibratedBrierDifference   Measure                `json:"calibrated_minus_baseline_brier"`
	CalibratedLogLossDifference Measure                `json:"calibrated_minus_baseline_log_loss"`
	Bootstrap                   *Bootstrap             `json:"bootstrap"`
}

type Report struct {
	Version             int                    `json:"version"`
	Toolchain           string                 `json:"go_toolchain,omitempty"`
	Target              string                 `json:"target"`
	Provenance          string                 `json:"provenance_status"`
	CorpusChecksum      string                 `json:"corpus_sha256"`
	Encoding            string                 `json:"corpus_encoding"`
	RawRows             int                    `json:"source_rows"`
	RetainedRows        int                    `json:"retained_rows"`
	ExcludedShort       int                    `json:"excluded_short_texts"`
	Partitions          map[string]int         `json:"partition_counts"`
	PartitionHash       string                 `json:"partition_sha256"`
	DictionaryHash      string                 `json:"dictionary_sha256"`
	PreprocessingHash   string                 `json:"preprocessing_sha256"`
	TrainingHash        string                 `json:"training_rules_sha256"`
	HeuristicHash       string                 `json:"heuristic_rules_sha256"`
	FeatureNames        []string               `json:"feature_names"`
	Protocol            Protocol               `json:"protocol"`
	Formulas            map[string]string      `json:"formulas"`
	Limitations         []string               `json:"limitations"`
	ArtifactStatus      string                 `json:"artifact_status"`
	ArtifactFingerprint string                 `json:"artifact_sha256,omitempty"`
	Traits              map[string]TraitReport `json:"traits"`
}

type sample struct {
	authorHash string
	x          []float64
	y          [5]bool
	heuristic  [5]float64
}

func heuristicScores(s analyze.BigFiveScores) [5]float64 {
	return [5]float64{s.Extraversion, s.Neuroticism, s.Agreeableness, s.Conscientiousness, s.Openness}
}

func matrix(rows []sample, trait int) ([][]float64, []bool) {
	x := make([][]float64, len(rows))
	y := make([]bool, len(rows))
	for i, row := range rows {
		x[i] = row.x
		y[i] = row.y[trait]
	}
	return x, y
}

func classCounts(y []bool) ClassCounts {
	c := ClassCounts{}
	for _, v := range y {
		if v {
			c.Positive++
		} else {
			c.Negative++
		}
	}
	return c
}

// stratifiedFolds assigns each unique author to exactly one validation fold.
// Input ordering is the frozen author-hash ordering, independent of CSV order.
func stratifiedFolds(y []bool, k int) ([]int, error) {
	c := classCounts(y)
	if c.Positive < k || c.Negative < k {
		return nil, fmt.Errorf("each fitting class needs at least %d authors", k)
	}
	folds := make([]int, len(y))
	counters := [2]int{}
	for i, positive := range y {
		class := 0
		if positive {
			class = 1
		}
		folds[i] = counters[class] % k
		counters[class]++
	}
	return folds, nil
}

func tune(x [][]float64, y []bool, p Protocol) (float64, []Trial, error) {
	folds, err := stratifiedFolds(y, p.Folds)
	if err != nil {
		return 0, nil, err
	}
	best, bestLoss := 0.0, math.Inf(1)
	var trials []Trial
	for _, lambda := range p.Lambdas {
		trial := Trial{Lambda: lambda}
		total := 0.0
		successful := true
		for fold := 0; fold < p.Folds; fold++ {
			var fitX, validationX [][]float64
			var fitY, validationY []bool
			for i, row := range x {
				if folds[i] == fold {
					validationX = append(validationX, row)
					validationY = append(validationY, y[i])
				} else {
					fitX = append(fitX, row)
					fitY = append(fitY, y[i])
				}
			}
			m, err := fitLogistic(fitX, fitY, lambda, p.MaxIterations)
			result := ValidationFold{Fit: m.Fit}
			if err != nil {
				result.Reason = err.Error()
				result.LogLoss = Measure{Reason: err.Error()}
				successful = false
			} else {
				loss := 0.0
				for i, row := range validationX {
					loss += binaryLoss(m.logit(row), validationY[i])
				}
				loss /= float64(len(validationY))
				result.LogLoss = measured(loss)
				total += loss
			}
			trial.Folds = append(trial.Folds, result)
		}
		if successful {
			mean := total / float64(p.Folds)
			trial.MeanLogLoss = measured(mean)
			if mean < bestLoss-1e-12 || (math.Abs(mean-bestLoss) <= 1e-12 && lambda > best) {
				best, bestLoss = lambda, mean
			}
		} else {
			trial.MeanLogLoss = Measure{Reason: "one or more validation fits failed"}
		}
		trials = append(trials, trial)
	}
	if !finite(bestLoss) {
		return 0, trials, fmt.Errorf("no lambda converged in all validation folds")
	}
	return best, trials, nil
}

// Run freezes partitions before any tuning and never refits on calibration or
// final-test rows. All output is aggregate; author text/identifiers stay local.
func Run(corpus Corpus, dictionary []byte, p Protocol) (Report, *Artifact, error) {
	if err := p.validate(); err != nil {
		return Report{}, nil, err
	}
	if len(corpus.Essays) != corpus.RawRows || !validHash(corpus.Checksum) {
		return Report{}, nil, fmt.Errorf("invalid corpus metadata")
	}
	dict, err := analyze.LoadDictionaryFromJSON(dictionary)
	if err != nil {
		return Report{}, nil, err
	}
	names := featureNames(dict)
	if len(names) == 0 {
		return Report{}, nil, fmt.Errorf("empty feature dictionary")
	}
	extractor, heuristic := analyze.NewFeatureExtractor(dict), analyze.NewBigFiveModel()
	var rows []sample
	for _, essay := range corpus.Essays {
		if essay.Document.WordCount < p.MinWords {
			continue
		}
		fv, _ := extractor.Extract(essay.Document)
		vector := make([]float64, len(names))
		for j, name := range names {
			vector[j] = fv.CategoryPercents[analyze.Category(name)]
		}
		hash := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", p.Seed, essay.Author)))
		rows = append(rows, sample{hex.EncodeToString(hash[:]), vector, essay.Labels, heuristicScores(heuristic.Infer(fv))})
	}
	if len(rows) < 15 {
		return Report{}, nil, fmt.Errorf("too few retained authors for fitting, calibration and testing")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].authorHash < rows[j].authorHash })
	nFit, nCal := len(rows)*3/5, len(rows)/5
	fitRows, calRows, testRows := rows[:nFit], rows[nFit:nFit+nCal], rows[nFit+nCal:]
	partition := make([]string, len(rows))
	for i, row := range rows {
		role := "test"
		if i < nFit {
			role = "fit"
		} else if i < nFit+nCal {
			role = "calibration"
		}
		partition[i] = role + ":" + row.authorHash
	}
	r := Report{Version: 1, Target: "positive questionnaire label y in the local Essays CSV", Provenance: "unverified local dataset provenance and redistribution terms", CorpusChecksum: corpus.Checksum, Encoding: corpus.Encoding, RawRows: corpus.RawRows, RetainedRows: len(rows), ExcludedShort: corpus.RawRows - len(rows), Partitions: map[string]int{"fit": nFit, "calibration": nCal, "test": len(testRows)}, PartitionHash: hashJSON(partition), DictionaryHash: analyze.DictionaryFingerprint(dictionary), PreprocessingHash: analyze.FeatureProcessingFingerprint(), TrainingHash: trainingFingerprint(), HeuristicHash: analyze.ModelFingerprint(), FeatureNames: names, Protocol: p, ArtifactStatus: "incomplete", Traits: map[string]TraitReport{}}
	r.Formulas = map[string]string{
		"features":        "x_j = 100 * category_match_count_j / normalized_token_count; overlapping categories need not sum to 100",
		"standardization": "z_j = (x_j - fitting_mean_j) / fitting_population_sd_j; z_j=0 when sd_j=0",
		"logit":           "f = intercept + sum(weight_j * z_j)",
		"probability":     "p = 1 / (1 + exp(-f)); stable implementation; no score rounding",
		"objective":       "mean(softplus(f_i) - y_i*f_i) + lambda/2 * sum(weight_j^2); intercept unpenalized",
		"calibration":     "p_cal = sigmoid(a*f+c); independently fitted softened targets: y+=(npos+1)/(npos+2), y-=1/(nneg+2)",
		"brier":           "mean((p_i-y_i)^2)", "log_loss": "mean(y_i*softplus(-f_i)+(1-y_i)*softplus(f_i))",
		"auc":                    "tie-correct Mann-Whitney U / (n_positive*n_negative); rank predicted logits to avoid sigmoid saturation ties",
		"diagnostic_calibration": "unpenalized test-label logistic fit against predicted logits; ideal intercept=0, slope=1; diagnostic only",
		"baseline":               "constant probability = positive fitting authors / fitting authors; test prevalence is not used",
		"bootstrap":              "paired sampling of test authors with replacement; linear-interpolated percentile intervals; fixed fitted models",
	}
	r.Limitations = []string{
		"Offline experiment only; no automatic release gate and no change to production scoring.",
		"Local label provenance is unverified; y is not presented as a confirmed above-median target.",
		"Results describe this local essay sample, dictionary and author split; accuracy for chat, email, web pages, other languages or populations is unestablished.",
		"Binary labels cannot establish continuous personality scores or general-population percentiles.",
		"Bootstrap intervals are conditional on fitted models and independent test authors, excluding training and dataset-selection uncertainty.",
		"95% intervals are descriptive; 99% paired AUC intervals concern five traits within each model variant, not simultaneous coverage of all reported metrics.",
		"The final-test set must not be reused for tuning, calibration or model selection; later experiments require fresh confirmation data.",
		"Learned category contributions are log-odds terms, not causal explanations or validated personality measurements.",
	}
	a := &Artifact{Version: 1, Target: r.Target, Provenance: r.Provenance, CorpusChecksum: r.CorpusChecksum, DictionaryHash: r.DictionaryHash, PreprocessingHash: r.PreprocessingHash, TrainingHash: r.TrainingHash, PartitionHash: r.PartitionHash, FeatureNames: names, Protocol: p, Traits: map[string]TrainedTrait{}}
	for traitIndex, trait := range Traits() {
		fitX, fitY := matrix(fitRows, traitIndex)
		calX, calY := matrix(calRows, traitIndex)
		testX, testY := matrix(testRows, traitIndex)
		t := TraitReport{Status: "failed", Classes: map[string]ClassCounts{"fit": classCounts(fitY), "calibration": classCounts(calY), "test": classCounts(testY)}}
		lambda, trials, err := tune(fitX, fitY, p)
		t.Tuning = trials
		if err != nil {
			t.Reason = err.Error()
			r.Traits[trait.Name] = t
			continue
		}
		t.SelectedLambda = &lambda
		m, err := fitLogistic(fitX, fitY, lambda, p.MaxIterations)
		t.Fit = m.Fit
		if err != nil {
			t.Reason = err.Error()
			r.Traits[trait.Name] = t
			continue
		}
		calLogits := make([]float64, len(calX))
		for i, row := range calX {
			calLogits[i] = m.logit(row)
		}
		c := fitCalibration(calLogits, calY, p.MaxIterations)
		t.Calibration = c
		raw, base, h := make([]float64, len(testY)), make([]float64, len(testY)), make([]float64, len(testY))
		var calibrated []float64
		prevalence := float64(t.Classes["fit"].Positive) / float64(len(fitY))
		t.FittingPrevalence = &prevalence
		baselineLogit := math.Log(prevalence / (1 - prevalence))
		if c.Status == "fitted" {
			calibrated = make([]float64, len(testY))
		}
		for i, row := range testX {
			raw[i] = m.logit(row)
			base[i] = baselineLogit
			h[i] = testRows[i].heuristic[traitIndex]
			if calibrated != nil {
				calibrated[i] = *c.Slope*raw[i] + *c.Intercept
			}
		}
		u, b := metrics(raw, testY, p.MaxIterations), metrics(base, testY, p.MaxIterations)
		t.Uncalibrated = &u
		t.Baseline = &b
		t.HeuristicAUC = measured(AUCBinary(h, testY))
		t.UncalibratedAUCDifference = difference(u.AUC, t.HeuristicAUC)
		if calibrated != nil {
			cm := metrics(calibrated, testY, p.MaxIterations)
			t.Calibrated = &cm
			t.CalibratedAUCDifference = difference(cm.AUC, t.HeuristicAUC)
			t.CalibratedBrierDifference = difference(cm.Brier, b.Brier)
			t.CalibratedLogLossDifference = difference(cm.LogLoss, b.LogLoss)
		} else {
			t.CalibratedAUCDifference = Measure{Reason: c.Reason}
			t.CalibratedBrierDifference = Measure{Reason: c.Reason}
			t.CalibratedLogLossDifference = Measure{Reason: c.Reason}
		}
		bs := bootstrap(raw, calibrated, base, h, testY, p.BootstrapSamples, p.Seed+int64(traitIndex)*10000)
		t.Bootstrap = &bs
		u.Intervals = bs.Uncalibrated
		b.Intervals = bs.Baseline
		if t.Calibrated != nil {
			t.Calibrated.Intervals = bs.Calibrated
		}
		t.Status = "evaluated"
		if c.Status == "failed" {
			t.Status = "calibration_failed"
			t.Reason = c.Reason
		} else {
			a.Traits[trait.Name] = TrainedTrait{Model: m, Calibration: c}
		}
		r.Traits[trait.Name] = t
	}
	if len(a.Traits) != len(Traits()) {
		return r, nil, nil
	}
	a.Fingerprint = a.contentFingerprint()
	if err := a.Validate(dictionary); err != nil {
		return r, nil, err
	}
	r.ArtifactStatus = "complete_offline_only"
	r.ArtifactFingerprint = a.Fingerprint
	return r, a, nil
}

func difference(a, b Measure) Measure {
	if a.Value == nil || b.Value == nil {
		return Measure{Reason: "comparison contains undefined statistic"}
	}
	return measured(*a.Value - *b.Value)
}
