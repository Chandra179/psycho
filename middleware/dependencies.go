package middleware

import (
	"psycho/zlogger"
)

type Dependencies struct {
	logger *zlogger.Logger
}

func NewDependencies(logger *zlogger.Logger) *Dependencies {
	return &Dependencies{logger: logger}
}
