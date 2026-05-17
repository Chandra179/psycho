package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"psycho/config"
	"psycho/middleware"
	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := zlogger.New(cfg.Middleware.Logger.Level)

	profileDeps, err := profile.NewDependencies(profile.Config{DBPath: cfg.Profile.DBPath}, logger)
	if err != nil {
		log.Fatalf("init profile: %v", err)
	}

	analyzeDeps, err := analyze.NewDependencies(analyze.Config{DictionaryPath: cfg.Analyze.DictionaryPath}, logger)
	if err != nil {
		log.Fatalf("init analyze: %v", err)
	}

	ingestDeps := ingest.NewDependencies(ingest.Config{MaxTextSize: cfg.Ingest.MaxTextSize}, logger)

	mwDeps := middleware.NewDependencies(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /analyze", makeHandleAnalyze(ingestDeps.Config, logger, analyzeDeps, profileDeps))

	chain := middleware.Chain(
		mux,
		mwDeps.Recovery(),
		middleware.RequestID,
		middleware.Timeout(middleware.TimeoutConfig{Duration: time.Duration(cfg.Middleware.TimeoutInSec) * time.Second}),
	)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.HTTP.Port),
		Handler:      chain,
		ReadTimeout:  time.Duration(cfg.App.HTTP.ReadTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(cfg.App.HTTP.WriteTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.App.HTTP.IdleTimeoutInSec) * time.Second,
	}

	logger.Info(context.Background(), "starting HTTP server", zlogger.Field{Key: "addr", Value: srv.Addr})
	if err := srv.ListenAndServe(); err != nil {
		logger.Error(context.Background(), "server error", zlogger.Field{Key: "error", Value: err.Error()})
	}
}
