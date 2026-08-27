package observability

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerRedactsSensitiveAttributes(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	logger, err := NewLogger(&output, LoggerOptions{Level: "info", Format: "json", Service: "api"})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	logger.Info("configured", "api_token", "never-print-this", "count", 2)
	if strings.Contains(output.String(), "never-print-this") || !strings.Contains(output.String(), "[REDACTED]") {
		t.Fatalf("log output = %s", output.String())
	}
}

func TestLoggerRejectsUnknownOptions(t *testing.T) {
	t.Parallel()
	if _, err := NewLogger(&bytes.Buffer{}, LoggerOptions{Level: "trace", Format: "json"}); err == nil {
		t.Fatal("NewLogger() accepted unknown level")
	}
	if _, err := NewLogger(&bytes.Buffer{}, LoggerOptions{Level: "info", Format: "xml"}); err == nil {
		t.Fatal("NewLogger() accepted unknown format")
	}
}
