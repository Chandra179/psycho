// Command train fits and evaluates Big Five label classifiers offline. It has
// no production-server wiring and never publishes or deploys trained models.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"psycho/modules/supervised"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("train", flag.ContinueOnError)
	csvPath := flags.String("csv", "corpus-eval/essays.csv", "local labeled corpus CSV")
	dictPath := flags.String("dictionary", "modules/analyze/dictionary.json", "feature dictionary JSON")
	outDir := flags.String("out", "testresults/supervised", "local output directory; keep model artifacts gitignored")
	p := supervised.DefaultProtocol()
	flags.IntVar(&p.MinWords, "min-words", p.MinWords, "minimum normalized tokens")
	flags.IntVar(&p.BootstrapSamples, "resamples", p.BootstrapSamples, "paired author bootstrap samples")
	flags.Int64Var(&p.Seed, "seed", p.Seed, "frozen partition and bootstrap seed")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if err := os.MkdirAll(*outDir, 0700); err != nil {
		return err
	}
	// Clear a previous trained artifact before input validation, including on
	// malformed-corpus failures. Aggregate records remain useful history.
	modelPath := filepath.Join(*outDir, "model.json")
	if err := os.Remove(modelPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	corpus, err := supervised.ReadCorpus(*csvPath)
	if err != nil {
		return err
	}
	dictionary, err := os.ReadFile(*dictPath)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Fitting fixed author partitions; final-test labels are excluded from tuning and probability calibration.")
	report, artifact, err := supervised.Run(corpus, dictionary, p)
	if err != nil {
		return err
	}
	report.Toolchain = runtime.Version()
	if err := writeJSON(filepath.Join(*outDir, "report.json"), report, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*outDir, "report.md"), []byte(report.Markdown()), 0644); err != nil {
		return err
	}
	if artifact != nil {
		if err := writeJSON(modelPath, artifact, 0600); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "Source/retained/excluded: %d/%d/%d. Fit/calibration/test: %d/%d/%d.\n", report.RawRows, report.RetainedRows, report.ExcludedShort, report.Partitions["fit"], report.Partitions["calibration"], report.Partitions["test"])
	for _, trait := range supervised.Traits() {
		t := report.Traits[trait.Name]
		fmt.Fprintf(out, "%-18s %s", trait.Name, t.Status)
		if t.Calibrated != nil && t.Calibrated.AUC.Value != nil {
			fmt.Fprintf(out, " calibrated AUC %.6f", *t.Calibrated.AUC.Value)
		}
		if t.Reason != "" {
			fmt.Fprintf(out, " (%s)", t.Reason)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "Aggregate reports: %s. Artifact: %s.\n", *outDir, report.ArtifactStatus)
	if artifact == nil {
		return fmt.Errorf("one or more fits failed; inspect aggregate report")
	}
	return nil
}

func writeJSON(path string, v any, mode os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), mode)
}
