package ingest

import "context"

const (
	PercentileMethodEmpirical           = "empirical"
	PercentileMethodNormalApproximation = "normal_approximation"
)

// PercentileReference describes how analysis percentiles were produced.
type PercentileReference struct {
	Method     string `json:"method"`
	Corpus     string `json:"corpus,omitempty"`
	SampleSize int    `json:"sample_size,omitempty"`
}

// AnalysisOutput is everything the HTTP layer needs to render a response
// after a successful analysis. Field types stay loose because ingest cannot
// import its sibling modules without an import cycle; the JSON shape is the
// contract at this seam, so the tags below are load-bearing — consumers
// (including the report renderer) decode this shape by its snake_case keys.
type AnalysisOutput struct {
	AnalysisID          string                   `json:"analysis_id"`
	WordCount           int                      `json:"word_count"`
	DictionaryCoverage  float64                  `json:"dictionary_coverage"`
	ConfidenceFlag      string                   `json:"confidence_flag"`
	Traits              map[string]any           `json:"traits"`
	Values              map[string]float64       `json:"values"`
	ValueEvidence       map[string][]string      `json:"value_evidence,omitempty"`
	ValueExcerpts       map[string][]TextExcerpt `json:"value_excerpts,omitempty"`
	PercentileReference *PercentileReference     `json:"percentile_reference,omitempty"`
	CalculationDetails  any                      `json:"calculation_details,omitempty"`
	Summary             any                      `json:"summary"`
	Narrative           string                   `json:"narrative,omitempty"`
}

// AnalyzeFunc is the seam the HTTP handlers call into. modules/server and
// the tests wire it to a *pipeline.Pipeline.
type AnalyzeFunc func(ctx context.Context, text string) (AnalysisOutput, error)
