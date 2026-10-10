// Package pipeline composes the full analysis flow (normalize, extract
// features, infer all dimensions, aggregate, narrate) into one unit that the
// browser build and the tests wire up once instead of duplicating the
// orchestration at every call site.
package pipeline

import (
	"context"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
)

// Pipeline runs text through every stage of the analysis. It is the single
// owner of stage ordering.
type Pipeline struct {
	extractor   *analyze.FeatureExtractor
	model       analyze.TraitModel
	aggregator  *profile.ScoreAggregator
	narrative   profile.NarrativeGenerator
	calibration *analyze.Calibration
}

func New(
	extractor *analyze.FeatureExtractor,
	model analyze.TraitModel,
	aggregator *profile.ScoreAggregator,
	narrative profile.NarrativeGenerator,
	calibration *analyze.Calibration,
) *Pipeline {
	return &Pipeline{
		extractor:   extractor,
		model:       model,
		aggregator:  aggregator,
		narrative:   narrative,
		calibration: calibration,
	}
}

// Run analyzes text end-to-end. Nothing is stored; the caller keeps the
// result. The context is checked between stages so a cancelled caller stops
// before more work is done.
func (p *Pipeline) Run(ctx context.Context, text string) (ingest.AnalysisOutput, error) {
	if err := ctx.Err(); err != nil {
		return ingest.AnalysisOutput{}, err
	}

	doc := ingest.NewNormalizer().Normalize(text)
	if err := ingest.ValidateDocument(doc); err != nil {
		return ingest.AnalysisOutput{}, err
	}
	features, coverage := p.extractor.Extract(doc)

	scores := analyze.ScoreFeatures(p.model, features)

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

	traits := make(map[string]any, len(prof.Traits))
	for k, v := range prof.Traits {
		traits[k] = v
	}

	return ingest.AnalysisOutput{
		AnalysisID:          prof.AnalysisID,
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
