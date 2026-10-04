package supervised

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"psycho/modules/ingest"
	"slices"
	"strings"
	"testing"
)

func testRun(t *testing.T) (Report, *Artifact, []byte, Corpus) {
	t.Helper()
	dictionary := []byte(`{"word":["word","work"],"idea":["hello","idea"]}`)
	c := syntheticCorpus(t, 150)
	p := DefaultProtocol()
	p.MinWords = 1
	p.BootstrapSamples = 20
	r, a, err := Run(c, dictionary, p)
	if err != nil || a == nil {
		t.Fatalf("workflow failed: %v, %+v", err, r.Traits)
	}
	return r, a, dictionary, c
}

func TestHeldOutLabelsAndFeaturesNeverChangeFitting(t *testing.T) {
	r, a, dict, c := testRun(t)
	order := append([]Essay(nil), c.Essays...)
	slices.SortFunc(order, func(a, b Essay) int {
		ha := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", r.Protocol.Seed, a.Author)))
		hb := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", r.Protocol.Seed, b.Author)))
		return strings.Compare(fmt.Sprintf("%x", ha), fmt.Sprintf("%x", hb))
	})
	nFit, nCal := r.Partitions["fit"], r.Partitions["calibration"]
	roles := map[string]int{}
	for i, essay := range order {
		roles[essay.Author] = i
	}
	changed := c
	changed.Essays = append([]Essay(nil), c.Essays...)
	for i, essay := range changed.Essays {
		if roles[essay.Author] >= nFit+nCal {
			changed.Essays[i].Document = ingest.NewNormalizer().Normalize(essay.Text + strings.Repeat(" hello ", 1000))
			for j := range essay.Labels {
				changed.Essays[i].Labels[j] = !essay.Labels[j]
			}
		}
	}
	_, testChanged, err := Run(changed, dict, r.Protocol)
	if err != nil || testChanged == nil {
		t.Fatal(err)
	}
	if hashJSON(a.Traits) != hashJSON(testChanged.Traits) {
		t.Fatal("final-test rows changed fitted models or calibrators")
	}
	for i, essay := range changed.Essays {
		if roles[essay.Author] >= nFit && roles[essay.Author] < nFit+nCal {
			changed.Essays[i].Document = ingest.NewNormalizer().Normalize(essay.Text + strings.Repeat(" word ", 1000))
			for j := range essay.Labels {
				changed.Essays[i].Labels[j] = !essay.Labels[j]
			}
		}
	}
	calReport, calChanged, err := Run(changed, dict, r.Protocol)
	if err != nil || calChanged == nil {
		t.Fatal(err)
	}
	for _, trait := range Traits() {
		if hashJSON(a.Traits[trait.Name].Model) != hashJSON(calChanged.Traits[trait.Name].Model) || hashJSON(r.Traits[trait.Name].Tuning) != hashJSON(calReport.Traits[trait.Name].Tuning) {
			t.Fatal("calibration rows changed fitting or fold standardization")
		}
	}
}

func TestWorkflowDeterminismAndPartitionIsolation(t *testing.T) {
	r, a, dictionary, c := testRun(t)
	again, another, err := Run(c, dictionary, r.Protocol)
	if err != nil || another == nil {
		t.Fatal(err)
	}
	if hashJSON(r) != hashJSON(again) || a.Fingerprint != another.Fingerprint {
		t.Fatal("rerun changed results")
	}
	if r.Partitions["fit"] != 90 || r.Partitions["calibration"] != 30 || r.Partitions["test"] != 30 {
		t.Fatal(r.Partitions)
	}
	if !slices.Equal(a.FeatureNames, []string{"idea", "word"}) {
		t.Fatal("feature order unsorted")
	}
	slices.Reverse(c.Essays)
	reversed, revArtifact, err := Run(c, dictionary, r.Protocol)
	if err != nil || revArtifact == nil {
		t.Fatal(err)
	}
	if r.PartitionHash != reversed.PartitionHash || hashJSON(r.Traits) != hashJSON(reversed.Traits) {
		t.Fatal("source order leaked into fitting")
	}
	for _, trait := range Traits() {
		tr := r.Traits[trait.Name]
		if tr.Status != "evaluated" || tr.Uncalibrated == nil || tr.Baseline == nil || len(tr.Tuning) != 6 {
			t.Fatal(tr)
		}
		for _, trial := range tr.Tuning {
			if len(trial.Folds) != 5 || trial.MeanLogLoss.Value == nil {
				t.Fatal(trial)
			}
		}
	}
	if strings.Contains(r.Markdown(), "author0") || strings.Contains(r.Markdown(), "unique essay number") {
		t.Fatal("participant rows leaked")
	}
	if !strings.Contains(r.Markdown(), "training_rules_sha256") {
		t.Fatal("exact aggregate record missing")
	}
}

func TestStratifiedFoldsAndFailedWorkflow(t *testing.T) {
	y := []bool{true, false, true, false, true, false, true, false, true, false, true, false}
	assign, err := stratifiedFolds(y, 5)
	if err != nil {
		t.Fatal(err)
	}
	for fold := 0; fold < 5; fold++ {
		counts := [2]int{}
		for i, f := range assign {
			if f == fold {
				class := 0
				if y[i] {
					class = 1
				}
				counts[class]++
			}
		}
		if counts[0] == 0 || counts[1] == 0 {
			t.Fatal("missing fold class")
		}
	}
	if _, err := stratifiedFolds([]bool{true, false}, 5); err == nil {
		t.Fatal("insufficient authors accepted")
	}
	_, _, dict, c := testRun(t)
	p := DefaultProtocol()
	p.MinWords = 1
	p.BootstrapSamples = 2
	p.MaxIterations = 1
	r, a, err := Run(c, dict, p)
	if err != nil {
		t.Fatal(err)
	}
	if a != nil || r.ArtifactStatus != "incomplete" {
		t.Fatal("failed fits produced artifact")
	}
	for _, tr := range r.Traits {
		if tr.Status != "failed" || tr.Reason == "" {
			t.Fatal("failure hidden")
		}
	}
	if _, err := json.Marshal(r); err != nil {
		t.Fatal(err)
	}
	p.MinWords = 10000
	if _, _, err := Run(c, dict, p); err == nil {
		t.Fatal("empty retained corpus accepted")
	}
}

func TestArtifactValidationAndReplay(t *testing.T) {
	_, a, dict, _ := testRun(t)
	if err := a.Validate(dict); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model.json")
	data, _ := json.Marshal(a)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadArtifact(path, dict)
	if err != nil {
		t.Fatal(err)
	}
	for _, trait := range Traits() {
		p, err := loaded.Predict(trait.Name, map[string]float64{"idea": 12.5, "word": 20})
		if err != nil {
			t.Fatal(err)
		}
		f := p.Intercept
		for _, term := range p.Contributions {
			z := 0.0
			if term.Scale > 0 {
				z = (term.Percentage - term.Mean) / term.Scale
			}
			if term.Standardized != z || term.LogitContribution != term.Weight*z {
				t.Fatal("term not replayable")
			}
			f += term.LogitContribution
		}
		if f != p.RawLogit || sigmoid(f) != p.Probability {
			t.Fatal("prediction not replayable")
		}
		if p.CalibratedProbability != nil {
			c := loaded.Traits[trait.Name].Calibration
			if sigmoid(*c.Slope*f+*c.Intercept) != *p.CalibratedProbability {
				t.Fatal("calibration not replayable")
			}
		}
	}
	if _, err := loaded.Predict("missing", map[string]float64{"idea": 1, "word": 2}); err == nil {
		t.Fatal("missing trait accepted")
	}
	if _, err := loaded.Predict("openness", map[string]float64{"idea": 1}); err == nil {
		t.Fatal("missing feature silently filled")
	}
	if err := loaded.Validate([]byte(`{"idea":["different"],"word":["word"]}`)); err == nil {
		t.Fatal("dictionary mismatch accepted")
	}
	mutations := []func(*Artifact){
		func(a *Artifact) { a.Version = 2 }, func(a *Artifact) { a.PreprocessingHash = "wrong" }, func(a *Artifact) { a.TrainingHash = "wrong" },
		func(a *Artifact) { slices.Reverse(a.FeatureNames) }, func(a *Artifact) { delete(a.Traits, "openness") },
		func(a *Artifact) {
			m := a.Traits["openness"]
			m.Model.Weights = m.Model.Weights[:1]
			a.Traits["openness"] = m
		},
		func(a *Artifact) {
			m := a.Traits["openness"]
			m.Model.Standardizer.Scale[0] = -1
			a.Traits["openness"] = m
		},
		func(a *Artifact) {
			m := a.Traits["openness"]
			m.Calibration.Status = "failed"
			a.Traits["openness"] = m
		},
	}
	for _, mutate := range mutations {
		var bad Artifact
		_ = json.Unmarshal(data, &bad)
		mutate(&bad)
		bad.Fingerprint = bad.contentFingerprint()
		if err := bad.Validate(dict); err == nil {
			t.Fatal("malformed artifact accepted")
		}
	}
	if err := os.WriteFile(path, append(data, []byte(" {}")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadArtifact(path, dict); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	if _, err := LoadArtifact(filepath.Join(t.TempDir(), "absent"), dict); err == nil {
		t.Fatal("missing artifact accepted")
	}
}

func TestLambdaTieChoosesStrongerRegularization(t *testing.T) {
	x := make([][]float64, 20)
	y := make([]bool, 20)
	for i := range x {
		x[i] = []float64{3}
		y[i] = i%2 == 0
	}
	lambda, trials, err := tune(x, y, DefaultProtocol())
	if err != nil || lambda != 10 || len(trials) != 6 {
		t.Fatal("tie did not choose strongest lambda", lambda, err)
	}
}

func TestArtifactRejectsNonfiniteParameters(t *testing.T) {
	_, a, dict, _ := testRun(t)
	m := a.Traits["openness"]
	m.Model.Weights[0] = math.NaN()
	a.Traits["openness"] = m
	a.Fingerprint = a.contentFingerprint()
	if err := a.Validate(dict); err == nil {
		t.Fatal("NaN parameter accepted")
	}
	_, a, dict, _ = testRun(t)
	m = a.Traits["openness"]
	m.Model.Intercept = math.Inf(1)
	a.Traits["openness"] = m
	a.Fingerprint = a.contentFingerprint()
	if err := a.Validate(dict); err == nil {
		t.Fatal("infinite parameter accepted")
	}
}
