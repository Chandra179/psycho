package server

import (
	"net/http"
	"time"

	"psycho/config"
	"psycho/middleware"
	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

func NewHandler(cfg *config.Config, logger *zlogger.Logger) (http.Handler, error) {
	profileDeps, err := profile.NewDependencies(profile.Config{DBPath: cfg.Profile.DBPath}, logger)
	if err != nil {
		return nil, err
	}

	analyzeDeps, err := analyze.NewDependencies(analyze.Config{DictionaryPath: cfg.Analyze.DictionaryPath}, logger)
	if err != nil {
		return nil, err
	}

	ingestDeps := ingest.NewDependencies(ingest.Config{MaxTextSize: cfg.Ingest.MaxTextSize, DirPath: cfg.Ingest.DirPath}, logger)

	mwDeps := middleware.NewDependencies(logger)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /analyze", analyze.MakeHandleAnalyze(
		ingestDeps.Config,
		logger,
		analyzeDeps,
		func(sourceType string, wordCount int, coverage float64, features analyze.FeatureVector, scores analyze.BigFiveScores) (string, map[string]any, string, string, error) {
			prof := profileDeps.Aggregator.Aggregate(scores, features, wordCount, coverage)
			analysisID, err := profileDeps.Storage.SaveAnalysis(sourceType, wordCount, coverage, features, prof)
			if err != nil {
				return "", nil, "", "", err
			}
			traits := make(map[string]any, len(prof.Traits))
			for k, v := range prof.Traits {
				traits[k] = v
			}
			narrative := profileDeps.NarrativeGenerator.GenerateSynthesis(prof)
			return analysisID, traits, prof.ConfidenceFlag, narrative, nil
		},
	))

	mux.HandleFunc("POST /analyze-dir", ingest.MakeHandleAnalyzeDir(
		ingestDeps.Config,
		logger,
		func(text string, sourceType string) (string, int, float64, string, map[string]any, any, string, error) {
			normalizer := ingest.NewNormalizer()
			doc := normalizer.Normalize(text)
			features, coverage := analyzeDeps.Extractor.Extract(doc)
			scores := analyzeDeps.Model.Infer(features)
			scores.RegulatoryFocus = analyze.ComputeRegulatoryFocus(features)
			scores.NeedForCognition = analyze.ComputeNeedForCognition(features)
			prof := profileDeps.Aggregator.Aggregate(scores, features, doc.WordCount, coverage)
			analysisID, err := profileDeps.Storage.SaveAnalysis(sourceType, doc.WordCount, coverage, features, prof)
			if err != nil {
				return "", 0, 0, "", nil, nil, "", err
			}
			traits := make(map[string]any, len(prof.Traits))
			for k, v := range prof.Traits {
				traits[k] = v
			}
			narrative := profileDeps.NarrativeGenerator.GenerateSynthesis(prof)
			return analysisID, doc.WordCount, coverage, prof.ConfidenceFlag, traits, prof.Summary, narrative, nil
		},
	))

	chain := middleware.Chain(
		mux,
		mwDeps.Recovery(),
		middleware.RequestID,
		middleware.Timeout(middleware.TimeoutConfig{Duration: time.Duration(cfg.Middleware.TimeoutInSec) * time.Second}),
	)

	return chain, nil
}
