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
// summarized in docs/system-design.md (Measured accuracy).
//
// Essays are scored with the production inference path (normalize →
// extract → Infer) using raw, uncalibrated scores: calibration offsets are
// monotone shifts and cannot change ranking. Metrics per trait: Spearman
// rank correlation and AUC (both appropriate for binary labels), plus a
// bootstrap CI on AUC.
package main

import (
	"encoding/csv"
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

// readLabeledCSV parses the essays CSV. The corpus is cp1252-encoded
// (typographic quotes), so high bytes are mapped to their Unicode forms
// before tokenizing.
func readLabeledCSV(path string) ([]map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read corpus: %w", err)
	}
	r := csv.NewReader(strings.NewReader(cp1252ToUTF8(string(raw))))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse corpus: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("corpus has %d records, need header + rows", len(records))
	}

	header := make([]string, len(records[0]))
	for i, h := range records[0] {
		header[i] = strings.TrimPrefix(strings.TrimSpace(h), "#")
	}
	var rows []map[string]string
	for _, rec := range records[1:] {
		row := make(map[string]string, len(header))
		for i, v := range rec {
			if i < len(header) {
				row[header[i]] = v
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

// cp1252ToUTF8 decodes cp1252 bytes to UTF-8: control-range high bytes
// (0x80–0x9F) via the standard table, remaining high bytes as latin-1.
// Operates on bytes — the input is not valid UTF-8, so ranging over runes
// would have already collapsed them to U+FFFD.
func cp1252ToUTF8(s string) string {
	cp1252High := []rune{
		0x20AC, 0x81, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
		0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x8D, 0x017D, 0x8F,
		0x90, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
		0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x9D, 0x017E, 0x9F,
	}
	needs := false
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			needs = true
			break
		}
	}
	if !needs {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 0x80 && c <= 0x9F:
			b.WriteRune(cp1252High[c-0x80])
		case c >= 0xA0:
			b.WriteRune(rune(c))
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
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
