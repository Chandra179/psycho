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
	extractor  *analyze.FeatureExtractor
	model      analyze.TraitModel
	aggregator *profile.ScoreAggregator
	narrative  profile.NarrativeGenerator
	storage    *profile.Storage
}

func New(
	extractor *analyze.FeatureExtractor,
	model analyze.TraitModel,
	aggregator *profile.ScoreAggregator,
	narrative profile.NarrativeGenerator,
	storage *profile.Storage,
) *Pipeline {
	return &Pipeline{
		extractor:  extractor,
		model:      model,
		aggregator: aggregator,
		narrative:  narrative,
		storage:    storage,
	}
}

// Run analyzes text end-to-end and persists the result. The context is
// checked between stages so a request whose deadline has expired
// (middleware.Timeout) stops before doing more work.
func (p *Pipeline) Run(ctx context.Context, sourceType, sourceDate, text string) (ingest.AnalysisOutput, error) {
	if err := ctx.Err(); err != nil {
		return ingest.AnalysisOutput{}, err
	}

	doc := ingest.NewNormalizer().Normalize(text)
	features, coverage := p.extractor.Extract(doc)

	scores := p.model.Infer(features)
	scores.RegulatoryFocus = analyze.ComputeRegulatoryFocus(features)
	scores.NeedForCognition = analyze.ComputeNeedForCognition(features)
	scores.CognitiveStyle = analyze.ComputeCognitiveStyle(features)
	scores.NeedForClosure = analyze.ComputeNeedForClosure(features)
	scores.Values = analyze.ComputeSchwartzValues(features)

	prof := p.aggregator.Aggregate(scores, features, doc.WordCount, coverage)
	prof.Narrative = p.narrative.GenerateSynthesis(prof)

	if err := ctx.Err(); err != nil {
		return ingest.AnalysisOutput{}, err
	}

	analysisID, err := p.storage.SaveAnalysis(sourceType, sourceDate, doc.WordCount, coverage, features, prof)
	if err != nil {
		return ingest.AnalysisOutput{}, err
	}

	traits := make(map[string]any, len(prof.Traits))
	for k, v := range prof.Traits {
		traits[k] = v
	}

	return ingest.AnalysisOutput{
		AnalysisID:         analysisID,
		WordCount:          doc.WordCount,
		DictionaryCoverage: coverage,
		ConfidenceFlag:     prof.ConfidenceFlag,
		Traits:             traits,
		Values:             prof.Values,
		Summary:            prof.Summary,
		Narrative:          prof.Narrative,
	}, nil
}
