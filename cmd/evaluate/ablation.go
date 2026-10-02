package main

// Dictionary-size ablation (discovery Cycle 0, see
// docs/discovery/2026-10-02-four-risks.md): rescore the labeled corpus with
// the dictionary thinned to a fraction of its categories and read the
// AUC-vs-size curve. A rising curve says dictionary growth buys ranking
// accuracy; a flat curve says the current levers are exhausted and the
// quality strategy must pivot.
//
// Subsets are drawn at the category level — all words of a kept category
// stay, nothing else changes — with k = round(level/100 × N) categories
// picked by a seeded shuffle of the sorted category names, so every run is
// reproducible from -seed. The 100% point is the unchanged production
// dictionary. Function/content splits contrast the closed-class style
// categories with all remaining (content) categories to locate where the
// signal lives. Scores always come from the production inference path
// (normalize → extract → Infer), raw and uncalibrated.

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
)

// functionCategories are the closed-class style categories — articles,
// prepositions, pronouns, negation — that carry authorial style rather than
// content (Pennebaker & King, 1999). Conservative split: everything not
// listed here counts as content.
var functionCategories = map[string]bool{
	"article":     true,
	"preposition": true,
	"pronoun":     true,
	"negation":    true,
}

type ablationTraitPoint struct {
	MeanAUC float64   `json:"mean_auc"`
	MinAUC  float64   `json:"min_auc"`
	MaxAUC  float64   `json:"max_auc"`
	AUCs    []float64 `json:"aucs"`
}

type ablationPoint struct {
	LevelPct      int                           `json:"level_pct"`
	Categories    int                           `json:"categories"`
	Deterministic bool                          `json:"deterministic"`
	MeanCoverage  float64                       `json:"mean_coverage"`
	Traits        map[string]ablationTraitPoint `json:"traits"`
}

type ablationSplit struct {
	Name         string             `json:"name"`
	Categories   []string           `json:"categories"`
	MeanCoverage float64            `json:"mean_coverage"`
	AUC          map[string]float64 `json:"auc"`
}

type ablationReport struct {
	Corpus          string             `json:"corpus"`
	GeneratedAt     string             `json:"generated_at"`
	Essays          int                `json:"essays"`
	Skipped         int                `json:"skipped_short_texts"`
	MinWords        int                `json:"min_words"`
	Draws           int                `json:"draws_per_level"`
	Seed            int64              `json:"seed"`
	SubsetSelection string             `json:"subset_selection"`
	Scoring         string             `json:"scoring"`
	Levels          []ablationPoint    `json:"levels"`
	Splits          []ablationSplit    `json:"splits"`
	ProjectedAUC2x  map[string]float64 `json:"projected_auc_2x_linear"`
}

type essayCase struct {
	doc    ingest.Document
	labels []bool // aligned with labelTraits
}

// runAblation executes the sweep and writes the report JSON. Decision rule
// (from the discovery doc): the cycle passes when the linear projection of
// the AUC curve to 2× dictionary size reaches 0.60 on the traits; a flat
// curve (sub-0.01 gain from the lowest level to 100%) fails it.
func runAblation(rows []map[string]string, dictData []byte, csvPath string, minWords, draws int, levels []int, seed int64, outPath string) {
	cases, skipped := prepareCases(rows, minWords)
	if len(cases) == 0 {
		fatal(fmt.Errorf("all %d rows skipped by -min-words %d", len(rows), minWords))
	}

	var raw map[string][]string
	if err := json.Unmarshal(dictData, &raw); err != nil {
		fatal(fmt.Errorf("parse dictionary: %w", err))
	}
	allCats := make([]string, 0, len(raw))
	for c := range raw {
		allCats = append(allCats, c)
	}
	sort.Strings(allCats)

	// Baseline: the unchanged dictionary, one deterministic run.
	fullDict, err := dictFromRaw(raw, allCats)
	if err != nil {
		fatal(err)
	}
	points := []ablationPoint{baselinePoint(cases, fullDict, 100, len(allCats))}

	// Thinned levels: draws random category subsets per level.
	for _, level := range levels {
		k := int(math.Round(float64(level) / 100 * float64(len(allCats))))
		perDraw := map[string][]float64{}
		var covSum float64
		for d := 0; d < draws; d++ {
			rng := rand.New(rand.NewSource(seed + int64(level)*1000 + int64(d)))
			keep := pickCategories(allCats, k, rng)
			dict, err := dictFromRaw(raw, keep)
			if err != nil {
				fatal(err)
			}
			scores, labels, cov := scoreCorpus(cases, dict)
			covSum += cov
			for _, t := range labelTraits {
				perDraw[t.dim] = append(perDraw[t.dim], aucBinary(scores[t.dim], labels[t.dim]))
			}
		}
		points = append(points, drawsPoint(level, k, perDraw, covSum/float64(draws)))
	}

	// Function vs content splits: where does the signal live?
	funcCats, contentCats := splitCategories(allCats)
	splits := []ablationSplit{
		runSplit(cases, raw, "function_only", funcCats),
		runSplit(cases, raw, "content_only", contentCats),
	}

	// Crude linear projection to 2× dictionary size: extend the line through
	// (lowest level, mean AUC) and (100%, baseline AUC) one more interval.
	// Multiplies the measured slope, so treat it as an upper-bound hint, not
	// a forecast.
	projected := map[string]float64{}
	lowest := points[len(points)-1]
	if lowest.LevelPct < 100 {
		for _, t := range labelTraits {
			base := points[0].Traits[t.dim].MeanAUC
			projected[t.dim] = round4(projectAUC2x(base, lowest.Traits[t.dim].MeanAUC, lowest.LevelPct))
		}
	}

	rep := ablationReport{
		Corpus:          filepath.Base(csvPath),
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Essays:          len(cases),
		Skipped:         skipped,
		MinWords:        minWords,
		Draws:           draws,
		Seed:            seed,
		SubsetSelection: "seeded shuffle of sorted category names, first k kept; seed = -seed + level*1000 + draw",
		Scoring:         "raw Big Five scores (production inference path; calibration offsets are monotone and ranking-invariant)",
		Levels:          points,
		Splits:          splits,
		ProjectedAUC2x:  projected,
	}
	writeAblationReport(rep, outPath)
	printAblationReport(rep, outPath)
}

// prepareCases normalizes every essay once — the normalizer does not depend
// on the dictionary — and filters by the minimum word count.
func prepareCases(rows []map[string]string, minWords int) ([]essayCase, int) {
	normalizer := ingest.NewNormalizer()
	cases := make([]essayCase, 0, len(rows))
	skipped := 0
	for _, row := range rows {
		doc := normalizer.Normalize(row["TEXT"])
		if doc.WordCount < minWords {
			skipped++
			continue
		}
		ec := essayCase{doc: doc}
		for _, t := range labelTraits {
			ec.labels = append(ec.labels, strings.EqualFold(row[t.label], "y"))
		}
		cases = append(cases, ec)
	}
	return cases, skipped
}

// scoreCorpus runs the production inference path over every case with the
// given dictionary and returns per-trait scores/labels plus mean coverage.
func scoreCorpus(cases []essayCase, dict analyze.Dictionary) (map[string][]float64, map[string][]bool, float64) {
	extractor := analyze.NewFeatureExtractor(dict)
	model := analyze.NewBigFiveModel()
	scores := make(map[string][]float64, len(labelTraits))
	labels := make(map[string][]bool, len(labelTraits))
	var covSum float64
	for _, ec := range cases {
		features, cov := extractor.Extract(ec.doc)
		s := model.Infer(features)
		covSum += cov
		for i, t := range labelTraits {
			scores[t.dim] = append(scores[t.dim], dimensionScore(t.dim, &s))
			labels[t.dim] = append(labels[t.dim], ec.labels[i])
		}
	}
	return scores, labels, covSum / float64(len(cases))
}

// baselinePoint reports one deterministic run of the full dictionary.
func baselinePoint(cases []essayCase, dict analyze.Dictionary, level, cats int) ablationPoint {
	scores, labels, cov := scoreCorpus(cases, dict)
	point := ablationPoint{
		LevelPct:      level,
		Categories:    cats,
		Deterministic: true,
		MeanCoverage:  cov,
		Traits:        map[string]ablationTraitPoint{},
	}
	for _, t := range labelTraits {
		auc := round4(aucBinary(scores[t.dim], labels[t.dim]))
		point.Traits[t.dim] = ablationTraitPoint{
			MeanAUC: auc, MinAUC: auc, MaxAUC: auc, AUCs: []float64{auc},
		}
	}
	return point
}

func drawsPoint(level, cats int, perDraw map[string][]float64, meanCov float64) ablationPoint {
	point := ablationPoint{
		LevelPct:      level,
		Categories:    cats,
		Deterministic: false,
		MeanCoverage:  meanCov,
		Traits:        map[string]ablationTraitPoint{},
	}
	for _, t := range labelTraits {
		aucs := append([]float64(nil), perDraw[t.dim]...)
		sort.Float64s(aucs)
		sum := 0.0
		for _, a := range aucs {
			sum += a
		}
		point.Traits[t.dim] = ablationTraitPoint{
			MeanAUC: round4(sum / float64(len(aucs))),
			MinAUC:  round4(aucs[0]),
			MaxAUC:  round4(aucs[len(aucs)-1]),
			AUCs:    roundAll(aucs),
		}
	}
	return point
}

func runSplit(cases []essayCase, raw map[string][]string, name string, cats []string) ablationSplit {
	split := ablationSplit{Name: name, Categories: cats, AUC: map[string]float64{}}
	if len(cats) == 0 {
		return split
	}
	dict, err := dictFromRaw(raw, cats)
	if err != nil {
		fatal(err)
	}
	scores, labels, cov := scoreCorpus(cases, dict)
	split.MeanCoverage = cov
	for _, t := range labelTraits {
		split.AUC[t.dim] = round4(aucBinary(scores[t.dim], labels[t.dim]))
	}
	return split
}

// dictFromRaw materializes a Dictionary over the named categories only.
func dictFromRaw(raw map[string][]string, keep []string) (analyze.Dictionary, error) {
	sub := make(map[string][]string, len(keep))
	for _, cat := range keep {
		words, ok := raw[cat]
		if !ok {
			return nil, fmt.Errorf("category %q not in dictionary", cat)
		}
		sub[cat] = words
	}
	data, err := json.Marshal(sub)
	if err != nil {
		return nil, err
	}
	return analyze.LoadDictionaryFromJSON(data)
}

func pickCategories(all []string, k int, rng *rand.Rand) []string {
	shuffled := slices.Clone(all)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	if k > len(shuffled) {
		k = len(shuffled)
	}
	return shuffled[:k]
}

func splitCategories(all []string) (funcCats, contentCats []string) {
	for _, c := range all {
		if functionCategories[c] {
			funcCats = append(funcCats, c)
		} else {
			contentCats = append(contentCats, c)
		}
	}
	return funcCats, contentCats
}

// projectAUC2x extends the line through (lowLevel, meanLow) and (100, base)
// to 200% dictionary size and clamps to [0, 1]. NaN when the low level is
// missing or degenerate.
func projectAUC2x(base, meanLow float64, lowLevelPct int) float64 {
	if lowLevelPct <= 0 || lowLevelPct >= 100 || math.IsNaN(base) || math.IsNaN(meanLow) {
		return math.NaN()
	}
	p := base + (base-meanLow)*(100.0/float64(100-lowLevelPct))
	return math.Min(1, math.Max(0, p))
}

func roundAll(vs []float64) []float64 {
	out := make([]float64, len(vs))
	for i, v := range vs {
		out[i] = round4(v)
	}
	return out
}

func writeAblationReport(rep ablationReport, outPath string) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fatal(err)
	}
	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(outPath, append(out, '\n'), 0o644); err != nil {
		fatal(fmt.Errorf("write %s: %w", outPath, err))
	}
}

func printAblationReport(rep ablationReport, outPath string) {
	fmt.Printf("dictionary ablation — %d essays (%d skipped under %d words) from %s, %d draws/level, seed %d\n",
		rep.Essays, rep.Skipped, rep.MinWords, rep.Corpus, rep.Draws, rep.Seed)
	fmt.Printf("full report written to %s\n\n", outPath)
	fmt.Printf("%-16s %5s %9s", "level", "cats", "coverage")
	for _, t := range labelTraits {
		fmt.Printf(" %9s", shortTrait(t.dim))
	}
	fmt.Println()
	for _, p := range rep.Levels {
		label := fmt.Sprintf("%d%%", p.LevelPct)
		if p.Deterministic {
			label += " (base)"
		}
		fmt.Printf("%-16s %5d %9.3f", label, p.Categories, p.MeanCoverage)
		for _, t := range labelTraits {
			fmt.Printf(" %9.4f", p.Traits[t.dim].MeanAUC)
		}
		fmt.Println()
	}
	for _, s := range rep.Splits {
		fmt.Printf("%-16s %5d %9.3f", s.Name, len(s.Categories), s.MeanCoverage)
		for _, t := range labelTraits {
			fmt.Printf(" %9.4f", s.AUC[t.dim])
		}
		fmt.Println()
	}
	if len(rep.ProjectedAUC2x) > 0 {
		fmt.Printf("%-16s %5s %9s", "proj 2x (lin)", "-", "-")
		for _, t := range labelTraits {
			fmt.Printf(" %9.4f", rep.ProjectedAUC2x[t.dim])
		}
		fmt.Println()
	}
	fmt.Println("\ndecision rule: projected 2x AUC >= 0.60 on the traits -> PASS (keep growing the dictionary); flat curve (< 0.01 gain lowest -> 100%) -> FAIL (pivot the quality strategy)")
}

// parseLevels parses a comma-separated list of dictionary size percentages,
// sorted descending, duplicates removed, 100 dropped (it is the baseline).
func parseLevels(s string) ([]int, error) {
	var levels []int
	seen := map[int]bool{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 100 {
			return nil, fmt.Errorf("invalid ablation level %q (want 1–100)", p)
		}
		if n < 100 && !seen[n] {
			seen[n] = true
			levels = append(levels, n)
		}
	}
	if len(levels) == 0 {
		return nil, fmt.Errorf("no ablation levels below 100%% in %q", s)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(levels)))
	return levels, nil
}

func shortTrait(dim string) string { return dim[:3] }
