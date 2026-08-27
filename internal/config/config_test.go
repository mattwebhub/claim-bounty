package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Address() != "127.0.0.1:8080" {
		t.Fatalf("Address() = %q", cfg.Server.Address())
	}
	if cfg.Server.ReadHeaderTimeout != 5*time.Second || cfg.HTTP.MaxBodyBytes != 1<<20 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, key, value, want string
	}{
		{"environment", "APP_ENV", "staging", "APP_ENV"},
		{"port", "SERVER_PORT", "70000", "SERVER_PORT"},
		{"duration", "SERVER_READ_TIMEOUT", "soon", "SERVER_READ_TIMEOUT"},
		{"body", "HTTP_MAX_BODY_BYTES", "0", "HTTP_MAX_BODY_BYTES"},
		{"level", "LOG_LEVEL", "verbose", "LOG_LEVEL"},
		{"origin", "CORS_ALLOWED_ORIGINS", "file:///tmp/x", "CORS origin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(env(map[string]string{tt.key: tt.value}))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	t.Parallel()
	_, err := Load(env(map[string]string{
		"APP_ENV": "production", "DATABASE_URL": "postgres://app@db.example/app?sslmode=verify-full", "CORS_ALLOWED_ORIGINS": "*",
	}))
	if err == nil || !strings.Contains(err.Error(), "wildcard") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestProductionDefaultsToJSONLogs(t *testing.T) {
	t.Parallel()
	cfg, err := Load(env(map[string]string{
		"APP_ENV": "production", "DATABASE_URL": "postgres://app@db.example/app?sslmode=verify-full",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("log format = %q", cfg.Log.Format)
	}
}

func TestProductionRequiresExplicitTLSDatabaseURL(t *testing.T) {
	t.Parallel()
	for _, values := range []map[string]string{
		{"APP_ENV": "production"},
		{"APP_ENV": "production", "DATABASE_URL": "postgres://app@db.example/app?sslmode=disable"},
	} {
		if _, err := Load(env(values)); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Fatalf("Load(%v) error = %v, want production database error", values, err)
		}
	}
}

func TestDatabaseConnectionCountCannotOverflowInt32Validation(t *testing.T) {
	t.Parallel()
	_, err := Load(env(map[string]string{"DATABASE_MAX_CONNECTIONS": "4294967306"}))
	if err == nil || !strings.Contains(err.Error(), "DATABASE_MAX_CONNECTIONS") {
		t.Fatalf("Load() error = %v, want max connections error", err)
	}
}

func env(values map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
