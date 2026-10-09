// Command calibrate derives the trait-score calibration consumed by the
// analyze module: per-dimension offsets that center the corpus mean at 0.50
// and the 1st–99th percentile quantiles of the adjusted scores, so reports
// can state "Xth percentile" against a measured reference population
// instead of an assumed normal distribution.
//
// Usage:
//
//	go run ./cmd/calibrate -corpus corpus/ -out config/calibration.json
//
// The corpus is a directory of .txt files, one document per file. Only the
// resulting calibration JSON is committed — the corpus itself stays local.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
)

func main() {
	corpusDir := flag.String("corpus", "", "directory of .txt files to calibrate against (required)")
	outPath := flag.String("out", "config/calibration.json", "output calibration JSON path")
	minWords := flag.Int("min-words", 200, "skip documents shorter than this many words")
	corpusName := flag.String("name", "", "corpus label stored in the calibration file (default: directory name)")
	flag.Parse()

	if *corpusDir == "" {
		fmt.Fprintln(os.Stderr, "error: -corpus is required")
		flag.Usage()
		os.Exit(2)
	}

	texts, err := readCorpus(*corpusDir)
	if err != nil {
		fatal(err)
	}

	data, err := os.ReadFile("modules/analyze/dictionary.json")
	if err != nil {
		fatal(fmt.Errorf("read dictionary: %w", err))
	}
	dict, err := analyze.LoadDictionaryFromJSON(data)
	if err != nil {
		fatal(fmt.Errorf("load dictionary: %w", err))
	}
	extractor := analyze.NewFeatureExtractor(dict)
	model := analyze.NewBigFiveModel()

	var samples []analyze.BigFiveScores
	skipped := 0
	for _, text := range texts {
		doc := ingest.NewNormalizer().Normalize(text)
		if doc.WordCount < *minWords {
			skipped++
			continue
		}
		features, _ := extractor.Extract(doc)
		s := analyze.ScoreFeatures(model, features)
		samples = append(samples, s)
	}

	name := *corpusName
	if name == "" {
		name = filepath.Base(filepath.Clean(*corpusDir))
	}
	cal, err := analyze.BuildCalibration(name, time.Now().UTC().Format(time.RFC3339), samples)
	if err != nil {
		fatal(err)
	}

	// Pin both dictionary content and the compiled scoring specification.
	cal.DictionarySHA256 = analyze.DictionaryFingerprint(data)

	out, err := json.MarshalIndent(cal, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*outPath, append(out, '\n'), 0o644); err != nil {
		fatal(fmt.Errorf("write %s: %w", *outPath, err))
	}

	fmt.Printf("calibrated %d dimensions from %d documents (%d skipped under %d words)\n",
		len(cal.Dimensions), len(samples), skipped, *minWords)
	fmt.Printf("corpus: %s\n", cal.Corpus)
	fmt.Printf("written to %s\n\n", *outPath)
	fmt.Printf("%-20s %8s %8s %9s\n", "dimension", "mean", "sd", "offset")
	keys := analyze.CalibratedDimensions()
	for _, dim := range keys {
		d := cal.Dimensions[dim]
		fmt.Printf("%-20s %8.4f %8.4f %+9.4f\n", dim, d.Mean, d.SD, d.Offset)
	}
}

func readCorpus(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read corpus dir: %w", err)
	}
	var texts []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		texts = append(texts, string(b))
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("no .txt files found in %s", dir)
	}
	return texts, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
