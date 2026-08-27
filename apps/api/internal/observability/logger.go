// Package observability constructs vendor-neutral process telemetry.
package observability

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type LoggerOptions struct {
	Level       string
	Format      string
	Service     string
	Environment string
}

func NewLogger(output io.Writer, options LoggerOptions) (*slog.Logger, error) {
	if output == nil {
		return nil, fmt.Errorf("observability: logger output is required")
	}
	level, err := parseLevel(options.Level)
	if err != nil {
		return nil, err
	}
	handlerOptions := &slog.HandlerOptions{Level: level, ReplaceAttr: redactAttribute}
	var handler slog.Handler
	switch options.Format {
	case "json":
		handler = slog.NewJSONHandler(output, handlerOptions)
	case "text":
		handler = slog.NewTextHandler(output, handlerOptions)
	default:
		return nil, fmt.Errorf("observability: log format must be text or json")
	}
	logger := slog.New(handler)
	if options.Service != "" {
		logger = logger.With("service", options.Service)
	}
	if options.Environment != "" {
		logger = logger.With("environment", options.Environment)
	}
	return logger, nil
}

func parseLevel(value string) (slog.Level, error) {
	switch value {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("observability: log level must be debug, info, warn, or error")
	}
}

func redactAttribute(_ []string, attribute slog.Attr) slog.Attr {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(attribute.Key))
	for _, fragment := range []string{"authorization", "cookie", "password", "secret", "token", "apikey", "databaseurl"} {
		if strings.Contains(normalized, fragment) {
			return slog.String(attribute.Key, "[REDACTED]")
		}
	}
	return attribute
}
