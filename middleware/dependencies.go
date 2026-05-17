package middleware

import (
	"brook/zlogger"
)

type Dependencies struct {
	logger *zlogger.Logger
}

func NewDependencies(logger *zlogger.Logger) *Dependencies {
	return &Dependencies{logger: logger}
}
