// Package logger wraps zap for structured, request-ID-aware logging.
package logger

import (
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ctxKey is unexported to prevent collisions in context.
type ctxKey int

const (
	requestIDKey ctxKey = iota
)

// New returns a zap.Logger configured per the level and format in cfg.
func New(level, format string) (*zap.Logger, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		lvl = zapcore.InfoLevel
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.MessageKey = "msg"
	encCfg.LevelKey = "level"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder

	core := zapcore.NewCore(
		encoderFromFormat(format, encCfg),
		zapcore.Lock(os.Stdout),
		lvl,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger, nil
}

func encoderFromFormat(format string, cfg zapcore.EncoderConfig) zapcore.Encoder {
	if strings.ToLower(format) == "console" {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(cfg)
}

// WithRequestID stores a request ID in the context for downstream loggers.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFrom returns the request ID stored in ctx, or empty string.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// FromContext returns a zap.Logger with the request_id field prefilled if present.
func FromContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	if id := RequestIDFrom(ctx); id != "" {
		return base.With(zap.String("request_id", id))
	}
	return base
}
