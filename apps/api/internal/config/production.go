package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
)

var productionClaimBountySettings = []string{
	"CORS_ALLOWED_ORIGINS",
	"HTTP_TRUSTED_PROXY_CIDRS",
	"CLAIMBOUNTY_CANONICAL_ORIGIN",
	"CLAIMBOUNTY_SESSION_PEPPER",
	"CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64",
	"CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64",
	"CLAIMBOUNTY_ADMIN_EMAILS",
	"CLAIMBOUNTY_AUTHORIZATION_VERSION",
	"CLAIMBOUNTY_ADMIN_ALLOWLIST_VERSION",
	"PII_RETENTION_POLICY_VERSION",
	"SOURCE_RETENTION_MAX_DURATION",
	"PII_RETENTION_MAX_DURATION",
	"CLAIMBOUNTY_ROUTINE_REVISION",
	"CLAIMBOUNTY_ROUTINE_VALIDATED_AT",
	"CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256",
	"CLAIMBOUNTY_S3_ENDPOINT",
	"CLAIMBOUNTY_S3_REGION",
	"CLAIMBOUNTY_S3_BUCKET",
	"CLAIMBOUNTY_S3_ACCESS_KEY",
	"CLAIMBOUNTY_S3_SECRET_KEY",
	"CLAIMBOUNTY_S3_SECURE",
	"CLAIMBOUNTY_S3_CREATE_BUCKET",
	"CLAIMBOUNTY_CLAMAV_ADDRESS",
	"CLAIMBOUNTY_CLAMAV_TIMEOUT",
	"CLAIMBOUNTY_SMTP_ADDRESS",
	"CLAIMBOUNTY_SMTP_FROM",
	"CLAIMBOUNTY_SMTP_TLS_MODE",
	"CLAIMBOUNTY_SMTP_TLS_SERVER_NAME",
	"CLAIMBOUNTY_SMTP_DEVELOPMENT_LOG",
	"CLAIMBOUNTY_WORKER_INTERVAL",
	"RETENTION_BATCH_SIZE",
	"RETENTION_COMMAND_TIMEOUT",
	"CLAIMBOUNTY_ABANDONED_AFTER",
}

func requireExplicitProductionSettings(lookup LookupEnv, claimBountyEnabled bool) error {
	required := []string{"DATABASE_URL", "DATABASE_AUTO_MIGRATE", "CLAIMBOUNTY_ENABLED"}
	if claimBountyEnabled {
		required = append(required, productionClaimBountySettings...)
	}
	for _, name := range required {
		if raw, ok := lookup(name); !ok || strings.TrimSpace(raw) == "" {
			return fmt.Errorf("config: %s is required in production", name)
		}
	}
	return nil
}

func validateProductionSecurity(c Config) error {
	canonical, _ := url.Parse(c.ClaimBounty.CanonicalOrigin)
	if canonical.Scheme != "https" || nonProductionHost(canonical.Hostname()) {
		return errors.New("config: production CLAIMBOUNTY_CANONICAL_ORIGIN must use HTTPS and a deployment host")
	}
	canonicalAllowed := false
	for _, originValue := range c.HTTP.AllowedOrigins {
		origin, _ := url.Parse(originValue)
		if origin.Scheme != "https" || nonProductionHost(origin.Hostname()) {
			return errors.New("config: production CORS_ALLOWED_ORIGINS must use HTTPS deployment origins")
		}
		canonicalAllowed = canonicalAllowed || originValue == c.ClaimBounty.CanonicalOrigin
	}
	if !canonicalAllowed {
		return errors.New("config: CLAIMBOUNTY_CANONICAL_ORIGIN must exactly match an allowed CORS origin")
	}

	for _, cidr := range c.HTTP.TrustedProxyCIDRs {
		if knownDocumentationCIDR(cidr) {
			return errors.New("config: HTTP_TRUSTED_PROXY_CIDRS contains a documentation placeholder")
		}
	}
	if placeholderCredential(c.ClaimBounty.SessionPepper) {
		return errors.New("config: CLAIMBOUNTY_SESSION_PEPPER contains a known non-production value")
	}
	if knownDemoEmailKey(c.ClaimBounty.EmailEncryptionKey) {
		return errors.New("config: CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64 contains a known non-production value")
	}
	if knownDemoEmailKey(c.ClaimBounty.EmailLookupKey) {
		return errors.New("config: CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64 contains a known non-production value")
	}
	for _, address := range c.ClaimBounty.AdminEmails {
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Address != address || nonProductionEmail(address) {
			return errors.New("config: CLAIMBOUNTY_ADMIN_EMAILS must contain deployment email addresses")
		}
	}

	if repeatedDigest(strings.TrimPrefix(c.ClaimBounty.TrustedRoutine.Revision, "sha256:")) {
		return errors.New("config: CLAIMBOUNTY_ROUTINE_REVISION contains a known placeholder digest")
	}
	if repeatedDigest(c.ClaimBounty.TrustedRoutine.EvidenceSHA256) {
		return errors.New("config: CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256 contains a known placeholder digest")
	}

	storageEndpoint, _ := url.Parse(c.ClaimBounty.S3.Endpoint)
	if storageEndpoint.Scheme != "https" || nonProductionHost(storageEndpoint.Hostname()) {
		return errors.New("config: production CLAIMBOUNTY_S3_ENDPOINT must use HTTPS and a deployment host")
	}
	if placeholderCredential(c.ClaimBounty.S3.AccessKey) {
		return errors.New("config: CLAIMBOUNTY_S3_ACCESS_KEY contains a known non-production value")
	}
	if placeholderCredential(c.ClaimBounty.S3.SecretKey) {
		return errors.New("config: CLAIMBOUNTY_S3_SECRET_KEY contains a known non-production value")
	}

	if err := validateProductionServiceAddress("CLAIMBOUNTY_CLAMAV_ADDRESS", c.ClaimBounty.ClamAV.Address); err != nil {
		return err
	}
	if err := validateProductionServiceAddress("CLAIMBOUNTY_SMTP_ADDRESS", c.ClaimBounty.SMTP.Address); err != nil {
		return err
	}
	from, err := mail.ParseAddress(c.ClaimBounty.SMTP.From)
	if err != nil || from.Address != c.ClaimBounty.SMTP.From || nonProductionEmail(c.ClaimBounty.SMTP.From) {
		return errors.New("config: CLAIMBOUNTY_SMTP_FROM must be a deployment email address")
	}
	if c.ClaimBounty.SMTP.DevelopmentLog {
		return errors.New("config: CLAIMBOUNTY_SMTP_DEVELOPMENT_LOG must be false in production")
	}
	if c.ClaimBounty.SMTP.TLSMode != "starttls" && c.ClaimBounty.SMTP.TLSMode != "implicit" {
		return errors.New("config: production CLAIMBOUNTY_SMTP_TLS_MODE must be starttls or implicit")
	}
	if net.ParseIP(c.ClaimBounty.SMTP.TLSServerName) != nil || !validHostname(c.ClaimBounty.SMTP.TLSServerName) || nonProductionHost(c.ClaimBounty.SMTP.TLSServerName) {
		return errors.New("config: CLAIMBOUNTY_SMTP_TLS_SERVER_NAME must be a deployment hostname")
	}
	if (c.ClaimBounty.SMTP.Username == "") != (c.ClaimBounty.SMTP.Password == "") {
		return errors.New("config: CLAIMBOUNTY_SMTP_USERNAME and CLAIMBOUNTY_SMTP_PASSWORD must be configured together")
	}
	if placeholderCredential(c.ClaimBounty.SMTP.Username) || placeholderCredential(c.ClaimBounty.SMTP.Password) {
		return errors.New("config: ClaimBounty SMTP credentials contain a known non-production value")
	}

	databaseURL, _ := url.Parse(c.Database.URL)
	if databaseURL.User != nil {
		password, hasPassword := databaseURL.User.Password()
		if hasPassword && strings.EqualFold(databaseURL.User.Username(), "postgres") && password == "postgres" {
			return errors.New("config: DATABASE_URL contains known demonstration credentials")
		}
	}
	return nil
}

func validateProductionServiceAddress(name, address string) error {
	host, rawPort, err := net.SplitHostPort(address)
	port, portErr := strconv.Atoi(rawPort)
	if err != nil || portErr != nil || port < 1 || port > 65535 || nonProductionHost(host) {
		return fmt.Errorf("config: %s must be a deployment host and port", name)
	}
	return nil
}

func nonProductionHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" || host == "localhost" || host == "mailpit" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".test") || strings.HasSuffix(host, ".invalid") {
		return true
	}
	return net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

func nonProductionEmail(address string) bool {
	at := strings.LastIndexByte(address, '@')
	return at < 1 || nonProductionHost(address[at+1:])
}

func placeholderCredential(value string) bool {
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	return lower == "demo" || strings.HasPrefix(lower, "demo-") || strings.HasSuffix(lower, "-demo") || strings.HasPrefix(lower, "development-") || strings.Contains(lower, "change-me") || strings.Contains(lower, "changeme") || strings.Contains(lower, "placeholder")
}

func knownDemoEmailKey(value string) bool {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	plain := string(decoded)
	return strings.HasPrefix(plain, "development-") || plain == "0123456789abcdef0123456789abcdef" || plain == "abcdef0123456789abcdef0123456789"
}

func repeatedDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	return strings.Trim(value, value[:1]) == ""
}

func knownDocumentationCIDR(value string) bool {
	switch value {
	case "192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24":
		return true
	default:
		return false
	}
}
