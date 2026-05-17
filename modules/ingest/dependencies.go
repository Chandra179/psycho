package ingest

import "psycho/zlogger"

type Dependencies struct {
	Config Config
	Logger *zlogger.Logger
}

func NewDependencies(cfg Config, logger *zlogger.Logger) *Dependencies {
	return &Dependencies{
		Config: cfg,
		Logger: logger,
	}
}
