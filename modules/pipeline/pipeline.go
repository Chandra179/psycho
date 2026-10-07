// Package pipeline composes the full analysis flow — normalize, extract
// features, infer all dimensions, aggregate, narrate, and persist — into one
// unit that the HTTP server and the tests wire up once instead of
// duplicating the orchestration at every call site.
package pipeline

import (
	"context"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
)

// Pipeline runs text through every stage of the analysis and persists the
// result. It is the single owner of stage ordering; the HTTP handlers only
// deal with transport.
type Pipeline struct {
	extractor   *analyze.FeatureExtractor
	model       analyze.TraitModel
	aggregator  *profile.ScoreAggregator
	narrative   profile.NarrativeGenerator
	storage     *profile.Storage
	calibration *analyze.Calibration
}

func New(
	extractor *analyze.FeatureExtractor,
	model analyze.TraitModel,
	aggregator *profile.ScoreAggregator,
	narrative profile.NarrativeGenerator,
	storage *profile.Storage,
	calibration *analyze.Calibration,
) *Pipeline {
	return &Pipeline{
		extractor:   extractor,
		model:       model,
		aggregator:  aggregator,
		narrative:   narrative,
		storage:     storage,
		calibration: calibration,
	}
}

// Run analyzes text end-to-end and persists the result. The context is
// checked between stages so a request whose deadline has expired
// (middleware.Timeout) stops before doing more work.
func (p *Pipeline) Run(ctx context.Context, text string) (ingest.AnalysisOutput, error) {
	if err := ctx.Err(); err != nil {
		return ingest.AnalysisOutput{}, err
	}

	doc := ingest.NewNormalizer().Normalize(text)
	if err := ingest.ValidateDocument(doc); err != nil {
		return ingest.AnalysisOutput{}, err
	}
	features, coverage := p.extractor.Extract(doc)

	scores := p.model.Infer(features)
	if scores.Calculations == nil {
		scores.Calculations = make(map[string]*analyze.ScoreCalculation)
	}
	scores.Calculations["regulatory_focus"] = analyze.ComputeRegulatoryFocusCalculation(features)
	scores.Calculations["need_for_cognition"] = analyze.ComputeNeedForCognitionCalculation(features)
	scores.Calculations["cognitive_style"] = analyze.ComputeCognitiveStyleCalculation(features)
	scores.Calculations["need_for_closure"] = analyze.ComputeNeedForClosureCalculation(features)
	scores.RegulatoryFocus = scores.Calculations["regulatory_focus"].FinalScore
	scores.NeedForCognition = scores.Calculations["need_for_cognition"].FinalScore
	scores.CognitiveStyle = scores.Calculations["cognitive_style"].FinalScore
	scores.NeedForClosure = scores.Calculations["need_for_closure"].FinalScore

	scores.Values = analyze.ComputeSchwartzValues(features)

	// Recenter scores against the calibration corpus before aggregation so
	// absolute scores and percentiles share the same reference population.
	// Nil calibration leaves the raw scores untouched.
	if p.calibration != nil {
		p.calibration.AdjustScores(&scores)
	}

	prof := p.aggregator.Aggregate(scores, features, doc.WordCount, coverage)
	prof.PercentileReference = percentileReference(p.calibration)
	prof.Narrative = p.narrative.GenerateSynthesis(prof)

	if err := ctx.Err(); err != nil {
		return ingest.AnalysisOutput{}, err
	}

	analysisID, err := p.storage.SaveAnalysis(doc.WordCount, coverage, features, prof)
	if err != nil {
		return ingest.AnalysisOutput{}, err
	}

	traits := make(map[string]any, len(prof.Traits))
	for k, v := range prof.Traits {
		traits[k] = v
	}

	return ingest.AnalysisOutput{
		AnalysisID:          analysisID,
		WordCount:           doc.WordCount,
		DictionaryCoverage:  coverage,
		ConfidenceFlag:      prof.ConfidenceFlag,
		Traits:              traits,
		Values:              prof.Values,
		ValueEvidence:       prof.ValueEvidence,
		ValueExcerpts:       prof.ValueExcerpts,
		PercentileReference: prof.PercentileReference,
		CalculationDetails:  prof.CalculationDetails,
		Summary:             prof.Summary,
		Narrative:           prof.Narrative,
	}, nil
}

func percentileReference(calibration *analyze.Calibration) *ingest.PercentileReference {
	if calibration == nil {
		return &ingest.PercentileReference{Method: ingest.PercentileMethodNormalApproximation}
	}
	return &ingest.PercentileReference{
		Method:     ingest.PercentileMethodEmpirical,
		Corpus:     calibration.Corpus,
		SampleSize: calibration.SampleSize(),
	}
}
