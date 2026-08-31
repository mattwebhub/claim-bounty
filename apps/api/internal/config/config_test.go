package config

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/system"
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

func TestLoadAcceptsMultipartGatewayLimitAndRejectsLargerBodies(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{"HTTP_MAX_BODY_BYTES": "263192576"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.MaxBodyBytes != 263192576 {
		t.Fatalf("HTTP max body bytes = %d", cfg.HTTP.MaxBodyBytes)
	}

	_, err = Load(env(map[string]string{"HTTP_MAX_BODY_BYTES": "268435457"}))
	if err == nil || !strings.Contains(err.Error(), "HTTP_MAX_BODY_BYTES") {
		t.Fatalf("Load() error = %v, want body limit error", err)
	}
}

func TestLoadPinsTrustedRoutineRegistryEntry(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"CLAIMBOUNTY_ROUTINE_REVISION":        "sha256:" + strings.Repeat("c", 64),
		"CLAIMBOUNTY_ROUTINE_VALIDATED_AT":    "2026-08-30T10:00:00Z",
		"CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256": strings.Repeat("d", 64),
	}
	cfg, err := Load(env(values))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ClaimBounty.TrustedRoutine.Revision != values["CLAIMBOUNTY_ROUTINE_REVISION"] || cfg.ClaimBounty.TrustedRoutine.EvidenceSHA256 != values["CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256"] {
		t.Fatalf("trusted routine = %+v", cfg.ClaimBounty.TrustedRoutine)
	}
}

func TestLoadRetentionPolicyCeilings(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"PII_RETENTION_POLICY_VERSION":  "intake-21d-v2",
		"SOURCE_RETENTION_MAX_DURATION": "336h",
		"PII_RETENTION_MAX_DURATION":    "504h",
	}
	cfg, err := Load(env(values))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ClaimBounty.RetentionPolicyVersion != values["PII_RETENTION_POLICY_VERSION"] || cfg.ClaimBounty.SourceRetentionMaxDuration != 14*24*time.Hour || cfg.ClaimBounty.PIIRetentionMaxDuration != 21*24*time.Hour {
		t.Fatalf("retention policy = %+v", cfg.ClaimBounty)
	}
}

func TestLoadRejectsInvalidRetentionPolicyCeilings(t *testing.T) {
	t.Parallel()
	for name, values := range map[string]map[string]string{
		"version": {"PII_RETENTION_POLICY_VERSION": "NOT-A-VERSION"},
		"source after pii": {
			"SOURCE_RETENTION_MAX_DURATION": "31h",
			"PII_RETENTION_MAX_DURATION":    "30h",
		},
		"non-positive": {"SOURCE_RETENTION_MAX_DURATION": "0s"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(env(values)); err == nil {
				t.Fatal("Load() accepted invalid retention policy ceilings")
			}
		})
	}
}

func TestLoadRejectsInvalidTrustedRoutineRegistryEntry(t *testing.T) {
	t.Parallel()

	for name, values := range map[string]map[string]string{
		"revision":  {"CLAIMBOUNTY_ROUTINE_REVISION": "main"},
		"timestamp": {"CLAIMBOUNTY_ROUTINE_VALIDATED_AT": "not-a-time"},
		"future":    {"CLAIMBOUNTY_ROUTINE_VALIDATED_AT": "2099-01-01T00:00:00Z"},
		"evidence":  {"CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256": strings.Repeat("z", 64)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(env(values)); err == nil {
				t.Fatal("Load() accepted an invalid trusted routine registry entry")
			}
		})
	}
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	t.Parallel()
	values := validProductionValues()
	values["CORS_ALLOWED_ORIGINS"] = "*"
	_, err := Load(env(values))
	if err == nil || !strings.Contains(err.Error(), "wildcard") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestProductionDefaultsToJSONLogs(t *testing.T) {
	t.Parallel()
	values := validProductionValues()
	cfg, err := Load(env(values))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("log format = %q", cfg.Log.Format)
	}
}

func TestDeploymentEmailKeyNamesInitializeProtector(t *testing.T) {
	t.Parallel()
	values := validProductionValues()
	cfg, err := Load(env(values))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.NewEmailProtector(cfg.ClaimBounty.EmailEncryptionKey, cfg.ClaimBounty.EmailLookupKey); err != nil {
		t.Fatalf("NewEmailProtector() error = %v", err)
	}
}

func validProductionValues() map[string]string {
	return map[string]string{
		"APP_ENV": "production", "DATABASE_URL": "postgres://app@db.acme.org/app?sslmode=verify-full", "DATABASE_AUTO_MIGRATE": "false", "CLAIMBOUNTY_ENABLED": "true",
		"CORS_ALLOWED_ORIGINS": "https://claimbounty.acme.org", "HTTP_TRUSTED_PROXY_CIDRS": "10.20.30.0/24",
		"CLAIMBOUNTY_CANONICAL_ORIGIN": "https://claimbounty.acme.org", "CLAIMBOUNTY_SESSION_PEPPER": "synthetic-session-pepper-value-000001",
		"CLAIMBOUNTY_ADMIN_EMAILS": "admin@acme.org", "CLAIMBOUNTY_AUTHORIZATION_VERSION": "admin-policy-2026-08-31", "CLAIMBOUNTY_ADMIN_ALLOWLIST_VERSION": "admin-allowlist-2026-08-31",
		"PII_RETENTION_POLICY_VERSION": "intake-30d-v1", "SOURCE_RETENTION_MAX_DURATION": "720h", "PII_RETENTION_MAX_DURATION": "720h",
		"CLAIMBOUNTY_ROUTINE_REVISION": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "CLAIMBOUNTY_ROUTINE_VALIDATED_AT": "2026-08-30T10:00:00Z", "CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256": "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		"CLAIMBOUNTY_S3_ENDPOINT": "https://objects.acme.org", "CLAIMBOUNTY_S3_REGION": "eu-west-1", "CLAIMBOUNTY_S3_BUCKET": "claimbounty-production", "CLAIMBOUNTY_S3_ACCESS_KEY": "api-service-access-key", "CLAIMBOUNTY_S3_SECRET_KEY": "synthetic-object-storage-secret-0001", "CLAIMBOUNTY_S3_SECURE": "true", "CLAIMBOUNTY_S3_CREATE_BUCKET": "false",
		"CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64": encodedTestKey(1), "CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64": encodedTestKey(65),
		"CLAIMBOUNTY_CLAMAV_ADDRESS": "scanner.acme.org:3310", "CLAIMBOUNTY_CLAMAV_TIMEOUT": "2m", "CLAIMBOUNTY_SMTP_ADDRESS": "smtp.acme.org:587", "CLAIMBOUNTY_SMTP_FROM": "no-reply@acme.org", "CLAIMBOUNTY_SMTP_TLS_MODE": "starttls", "CLAIMBOUNTY_SMTP_TLS_SERVER_NAME": "smtp.acme.org", "CLAIMBOUNTY_SMTP_DEVELOPMENT_LOG": "false",
		"CLAIMBOUNTY_WORKER_INTERVAL": "2s", "RETENTION_BATCH_SIZE": "25", "RETENTION_COMMAND_TIMEOUT": "10m", "CLAIMBOUNTY_ABANDONED_AFTER": "168h",
	}
}

func TestSMTPModesRequireValidExplicitTLSConfiguration(t *testing.T) {
	t.Parallel()

	for name, values := range map[string]map[string]string{
		"unknown mode":        {"CLAIMBOUNTY_SMTP_TLS_MODE": "opportunistic"},
		"missing server name": {"CLAIMBOUNTY_SMTP_TLS_MODE": "starttls"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(env(values)); err == nil || !strings.Contains(err.Error(), "CLAIMBOUNTY_SMTP_TLS") {
				t.Fatalf("Load() error = %v, want SMTP TLS configuration error", err)
			}
		})
	}

	values := map[string]string{
		"CLAIMBOUNTY_SMTP_TLS_MODE":        "implicit",
		"CLAIMBOUNTY_SMTP_TLS_SERVER_NAME": "smtp.example.test",
		"CLAIMBOUNTY_SMTP_TLS_CA_FILE":     "/run/secrets/smtp-ca.pem",
	}
	cfg, err := Load(env(values))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ClaimBounty.SMTP.TLSMode != "implicit" || cfg.ClaimBounty.SMTP.TLSServerName != "smtp.example.test" || cfg.ClaimBounty.SMTP.TLSCAFile != "/run/secrets/smtp-ca.pem" {
		t.Fatalf("SMTP TLS configuration = %+v", cfg.ClaimBounty.SMTP)
	}
}

func encodedTestKey(start byte) string {
	value := make([]byte, 32)
	for index := range value {
		value[index] = start + byte(index)
	}
	return base64.StdEncoding.EncodeToString(value)
}

func TestProductionRequiresExplicitRetentionPolicy(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"PII_RETENTION_POLICY_VERSION", "SOURCE_RETENTION_MAX_DURATION", "PII_RETENTION_MAX_DURATION"} {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			values := validProductionValues()
			delete(values, name)
			if _, err := Load(env(values)); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("Load() error = %v, want missing %s", err, name)
			}
		})
	}
}

func TestProductionRequiresExplicitTLSDatabaseURL(t *testing.T) {
	t.Parallel()
	missingDatabase := validProductionValues()
	delete(missingDatabase, "DATABASE_URL")
	insecureDatabase := validProductionValues()
	insecureDatabase["DATABASE_URL"] = "postgres://app@db.example/app?sslmode=disable"
	for _, values := range []map[string]string{missingDatabase, insecureDatabase} {
		if _, err := Load(env(values)); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Fatalf("Load() error = %v, want production database error", err)
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
