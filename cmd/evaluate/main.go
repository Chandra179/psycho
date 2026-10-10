// Command evaluate measures how well Psycho's inferred Big Five scores
// rank against ground-truth personality labels on a labeled corpus — the
// validity check the unit and validation tests cannot provide (issue #12).
//
// The expected corpus is the "Essays" dataset (Pennebaker & King, 1999;
// used by Mairesse et al., 2010): a CSV with #AUTHID, TEXT, and binary
// median-split labels cEXT, cNEU, cAGR, cCON, cOPN. Example mirror:
//
//	go run ./cmd/evaluate -csv corpus-eval/essays.csv
//
// With -ablation, it instead thins the dictionary to fractions of its
// categories and reports the AUC-vs-size curve — the breadth experiment
// summarized in docs/overview.md (Measured accuracy).
//
// Essays are scored with the production inference path (normalize →
// extract → Infer) using raw, uncalibrated scores: calibration offsets are
// monotone shifts and cannot change ranking. Metrics per trait: Spearman
// rank correlation and AUC (both appropriate for binary labels), plus a
// bootstrap CI on AUC.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/supervised"
)

type traitResult struct {
	Spearman float64   `json:"spearman"`
	AUC      float64   `json:"auc"`
	AUCCI    []float64 `json:"auc_ci95"`
	Positive int       `json:"positives"`
	Negative int       `json:"negatives"`
}

type report struct {
	Corpus      string                 `json:"corpus"`
	GeneratedAt string                 `json:"generated_at"`
	Essays      int                    `json:"essays"`
	Skipped     int                    `json:"skipped_short_texts"`
	MinWords    int                    `json:"min_words"`
	Scoring     string                 `json:"scoring"`
	Traits      map[string]traitResult `json:"traits"`
}

var labelTraits = []struct {
	label string
	dim   string
}{
	{"cEXT", "extraversion"},
	{"cNEU", "neuroticism"},
	{"cAGR", "agreeableness"},
	{"cCON", "conscientiousness"},
	{"cOPN", "openness"},
}

func main() {
	csvPath := flag.String("csv", "corpus-eval/essays.csv", "labeled corpus CSV (#AUTHID, TEXT, cEXT..cOPN y/n)")
	outPath := flag.String("out", "testresults/accuracy-essays.json", "output metrics JSON path")
	minWords := flag.Int("min-words", 200, "skip essays shorter than this many words")
	resamples := flag.Int("resamples", 1000, "bootstrap resamples for the AUC confidence interval")
	seed := flag.Int64("seed", 42, "random seed for the bootstrap")
	ablation := flag.Bool("ablation", false, "run the dictionary-ablation sweep instead of the baseline report")
	ablationLevels := flag.String("ablation-levels", "50,75", "comma-separated dictionary size levels (% of categories) for -ablation")
	draws := flag.Int("draws", 5, "random category subsets per level for -ablation")
	ablationOut := flag.String("ablation-out", "testresults/ablation-essays.json", "output JSON path for -ablation")
	flag.Parse()

	rows, err := readLabeledCSV(*csvPath)
	if err != nil {
		fatal(err)
	}

	data, err := os.ReadFile("modules/analyze/dictionary.json")
	if err != nil {
		fatal(fmt.Errorf("read dictionary: %w", err))
	}

	if *ablation {
		levels, err := parseLevels(*ablationLevels)
		if err != nil {
			fatal(err)
		}
		runAblation(rows, data, *csvPath, *minWords, *draws, levels, *seed, *ablationOut)
		return
	}

	dict, err := analyze.LoadDictionaryFromJSON(data)
	if err != nil {
		fatal(fmt.Errorf("load dictionary: %w", err))
	}
	extractor := analyze.NewFeatureExtractor(dict)
	model := analyze.NewBigFiveModel()
	normalizer := ingest.NewNormalizer()

	scores := map[string][]float64{}
	labels := map[string][]bool{}
	scoresF := map[string][]float64{}
	labelsF := map[string][]float64{}
	var traitNames []string
	for _, t := range labelTraits {
		traitNames = append(traitNames, t.dim)
	}

	skipped := 0
	for _, row := range rows {
		doc := normalizer.Normalize(row["TEXT"])
		if doc.WordCount < *minWords {
			skipped++
			continue
		}
		features, _ := extractor.Extract(doc)
		s := model.Infer(features)

		for _, t := range labelTraits {
			v := dimensionScore(t.dim, &s)
			scores[t.dim] = append(scores[t.dim], v)
			scoresF[t.dim] = append(scoresF[t.dim], v)
			isPos := strings.EqualFold(row[t.label], "y")
			labels[t.dim] = append(labels[t.dim], isPos)
			if isPos {
				labelsF[t.dim] = append(labelsF[t.dim], 1)
			} else {
				labelsF[t.dim] = append(labelsF[t.dim], 0)
			}
		}
	}
	if skipped == len(rows) {
		fatal(fmt.Errorf("all %d rows skipped by -min-words %d", len(rows), *minWords))
	}

	rng := rand.New(rand.NewSource(*seed))
	rep := report{
		Corpus:      filepath.Base(*csvPath),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Essays:      len(rows) - skipped,
		Skipped:     skipped,
		MinWords:    *minWords,
		Scoring:     "raw Big Five scores (production inference path; calibration offsets are monotone and ranking-invariant)",
		Traits:      map[string]traitResult{},
	}
	for _, dim := range traitNames {
		auc := aucBinary(scores[dim], labels[dim])
		lo, hi := aucBootstrapCI(scores[dim], labels[dim], *resamples, rng)
		nPos, nNeg := 0, 0
		for _, p := range labels[dim] {
			if p {
				nPos++
			} else {
				nNeg++
			}
		}
		rep.Traits[dim] = traitResult{
			Spearman: round4(spearman(scoresF[dim], labelsF[dim])),
			AUC:      round4(auc),
			AUCCI:    []float64{round4(lo), round4(hi)},
			Positive: nPos,
			Negative: nNeg,
		}
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatal(err)
	}
	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*outPath, append(out, '\n'), 0o644); err != nil {
		fatal(fmt.Errorf("write %s: %w", *outPath, err))
	}

	fmt.Printf("evaluated %d essays (%d skipped under %d words) from %s\n", rep.Essays, rep.Skipped, rep.MinWords, rep.Corpus)
	fmt.Printf("written to %s\n\n", *outPath)
	fmt.Printf("%-18s %10s %8s %18s\n", "trait", "spearman", "auc", "auc 95% CI")
	for _, t := range labelTraits {
		r := rep.Traits[t.dim]
		fmt.Printf("%-18s %10.4f %8.4f [%7.4f, %7.4f]\n", t.dim, r.Spearman, r.AUC, r.AUCCI[0], r.AUCCI[1])
	}
}

// readLabeledCSV shares strict corpus validation with the offline trainer.
func readLabeledCSV(path string) ([]map[string]string, error) {
	corpus, err := supervised.ReadCorpus(path)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(corpus.Essays))
	for _, essay := range corpus.Essays {
		row := map[string]string{"AUTHID": essay.Author, "TEXT": essay.Text}
		for j, trait := range supervised.Traits() {
			row[trait.Column] = "n"
			if essay.Labels[j] {
				row[trait.Column] = "y"
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func dimensionScore(dim string, s *analyze.BigFiveScores) float64 {
	switch dim {
	case "openness":
		return s.Openness
	case "conscientiousness":
		return s.Conscientiousness
	case "extraversion":
		return s.Extraversion
	case "agreeableness":
		return s.Agreeableness
	case "neuroticism":
		return s.Neuroticism
	}
	return math.NaN()
}

// Kept for the existing encoding regression test.
func cp1252ToUTF8(s string) string {
	decoded, _, _ := supervised.DecodeCorpus([]byte(s))
	return decoded
}

func round4(v float64) float64 {
	if math.IsNaN(v) {
		return v
	}
	return math.Round(v*10000) / 10000
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
