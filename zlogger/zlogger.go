package zlogger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Field struct {
	Key   string
	Value string
}

type Logger struct {
	zap *zap.Logger
}

func New(level string) *Logger {
	var lvl zapcore.Level
	switch level {
	case "dev":
		lvl = zapcore.DebugLevel
	default:
		lvl = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	z, err := cfg.Build()
	if err != nil {
		z = zap.NewNop()
	}
	return &Logger{zap: z}
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...Field) {
	l.zap.Info(msg, toZapFields(fields)...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...Field) {
	l.zap.Error(msg, toZapFields(fields)...)
}

func toZapFields(fields []Field) []zap.Field {
	out := make([]zap.Field, len(fields))
	for i, f := range fields {
		out[i] = zap.String(f.Key, f.Value)
	}
	return out
}
