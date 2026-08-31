package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestProductionRequiresExplicitSecuritySettings(t *testing.T) {
	t.Parallel()
	required := append([]string{"DATABASE_URL", "DATABASE_AUTO_MIGRATE", "CLAIMBOUNTY_ENABLED"}, productionClaimBountySettings...)
	for _, name := range required {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			values := validProductionValues()
			delete(values, name)
			_, err := Load(env(values))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("Load() error = %v, want missing setting name", err)
			}
		})
	}
}

func TestProductionRejectsKnownNonProductionValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, key, value, want string
		sensitive              bool
	}{
		{"HTTP CORS origin", "CORS_ALLOWED_ORIGINS", "http://claimbounty.acme.org", "CORS_ALLOWED_ORIGINS", false},
		{"reserved CORS host", "CORS_ALLOWED_ORIGINS", "https://claimbounty.example.invalid", "CORS_ALLOWED_ORIGINS", false},
		{"documentation proxy range", "HTTP_TRUSTED_PROXY_CIDRS", "192.0.2.0/24", "HTTP_TRUSTED_PROXY_CIDRS", false},
		{"HTTP canonical origin", "CLAIMBOUNTY_CANONICAL_ORIGIN", "http://claimbounty.acme.org", "CLAIMBOUNTY_CANONICAL_ORIGIN", false},
		{"reserved canonical host", "CLAIMBOUNTY_CANONICAL_ORIGIN", "https://claimbounty.example.invalid", "CLAIMBOUNTY_CANONICAL_ORIGIN", false},
		{"demo session pepper", "CLAIMBOUNTY_SESSION_PEPPER", "demo-only-identity-token-pepper-change-me", "CLAIMBOUNTY_SESSION_PEPPER", true},
		{"demo encryption key", "CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")), "CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64", true},
		{"demo lookup key", "CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64", base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789")), "CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64", true},
		{"demo administrator", "CLAIMBOUNTY_ADMIN_EMAILS", "admin@example.test", "CLAIMBOUNTY_ADMIN_EMAILS", false},
		{"placeholder routine revision", "CLAIMBOUNTY_ROUTINE_REVISION", "sha256:" + strings.Repeat("a", 64), "CLAIMBOUNTY_ROUTINE_REVISION", false},
		{"placeholder routine evidence", "CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256", strings.Repeat("b", 64), "CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256", false},
		{"reserved storage host", "CLAIMBOUNTY_S3_ENDPOINT", "https://objects.example.invalid", "CLAIMBOUNTY_S3_ENDPOINT", false},
		{"demo storage access key", "CLAIMBOUNTY_S3_ACCESS_KEY", "claimbounty-api-demo", "CLAIMBOUNTY_S3_ACCESS_KEY", true},
		{"demo storage secret", "CLAIMBOUNTY_S3_SECRET_KEY", "demo-api-object-store-secret-key", "CLAIMBOUNTY_S3_SECRET_KEY", true},
		{"reserved scanner host", "CLAIMBOUNTY_CLAMAV_ADDRESS", "scanner.example.invalid:3310", "CLAIMBOUNTY_CLAMAV_ADDRESS", false},
		{"development SMTP host", "CLAIMBOUNTY_SMTP_ADDRESS", "mailpit:1025", "CLAIMBOUNTY_SMTP_ADDRESS", false},
		{"demo SMTP sender", "CLAIMBOUNTY_SMTP_FROM", "no-reply@claimbounty.test", "CLAIMBOUNTY_SMTP_FROM", false},
		{"plaintext SMTP", "CLAIMBOUNTY_SMTP_TLS_MODE", "none", "CLAIMBOUNTY_SMTP_TLS_MODE", false},
		{"reserved SMTP TLS name", "CLAIMBOUNTY_SMTP_TLS_SERVER_NAME", "smtp.example.invalid", "CLAIMBOUNTY_SMTP_TLS_SERVER_NAME", false},
		{"development mail logging", "CLAIMBOUNTY_SMTP_DEVELOPMENT_LOG", "true", "CLAIMBOUNTY_SMTP_DEVELOPMENT_LOG", false},
		{"demo database credentials", "DATABASE_URL", "postgres://postgres:postgres@db.acme.org/app?sslmode=verify-full", "DATABASE_URL", true},
		{"automatic production migration", "DATABASE_AUTO_MIGRATE", "true", "DATABASE_AUTO_MIGRATE", false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			values := validProductionValues()
			values[test.key] = test.value
			_, err := Load(env(values))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want setting name", err)
			}
			if test.sensitive && strings.Contains(err.Error(), test.value) {
				t.Fatal("Load() exposed a rejected credential value")
			}
		})
	}
}

func TestProductionRequiresCanonicalOriginInCORSAllowlist(t *testing.T) {
	t.Parallel()
	values := validProductionValues()
	values["CORS_ALLOWED_ORIGINS"] = "https://secondary.acme.org"
	_, err := Load(env(values))
	if err == nil || !strings.Contains(err.Error(), "exactly match") {
		t.Fatalf("Load() error = %v, want exact origin mismatch", err)
	}
}

func TestProductionAllowsClaimBountyToBeExplicitlyDisabled(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"APP_ENV":               "production",
		"DATABASE_URL":          "postgres://app@db.acme.org/app?sslmode=verify-full",
		"DATABASE_AUTO_MIGRATE": "false",
		"CLAIMBOUNTY_ENABLED":   "false",
	}
	if _, err := Load(env(values)); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}
