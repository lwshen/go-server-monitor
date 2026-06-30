// Package logger provides a thin wrapper around zap for structured logging.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init builds a production-ish *zap.Logger at the given level.
// level accepts "debug", "info", "warn", "error" (defaults to info on parse failure).
//
// The encoder uses ISO8601 timestamps so log lines read like:
//
//	2025-01-20T14:30:45.000Z  INFO  服务启动在 :8080
func Init(level string) *zap.Logger {
	lvl := zapcore.InfoLevel
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = zapcore.InfoLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	l, err := cfg.Build()
	if err != nil {
		// Fall back to a no-frills logger; Init must never fail the boot.
		l = zap.NewNop()
	}
	return l
}
