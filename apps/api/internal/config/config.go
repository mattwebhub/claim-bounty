// Package config parses and validates process configuration exactly once.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHost              = "127.0.0.1"
	defaultPort              = 8080
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 15 * time.Second
	defaultReadinessTimeout  = 2 * time.Second
	defaultMaxHeaderBytes    = 1 << 20
	defaultMaxBodyBytes      = 1 << 20
	maxBodyBytes             = 256 << 20
)

// LookupEnv makes configuration loading deterministic and testable.
type LookupEnv func(string) (string, bool)

// Config is the immutable application configuration assembled at startup.
type Config struct {
	Environment Environment
	Server      Server
	HTTP        HTTP
	Log         Log
	Database    Database
	ClaimBounty ClaimBounty
}

type ClaimBounty struct {
	Enabled                    bool
	CanonicalOrigin            string
	SessionPepper              string
	AdminEmails                []string
	AuthorizationVersion       string
	AdminAllowlistVersion      string
	RetentionPolicyVersion     string
	SourceRetentionMaxDuration time.Duration
	PIIRetentionMaxDuration    time.Duration
	WorkerInterval             time.Duration
	RetentionBatch             int
	RetentionTimeout           time.Duration
	AbandonedAfter             time.Duration
	EmailEncryptionKey         string
	EmailLookupKey             string
	TrustedRoutine             TrustedRoutine
	S3                         S3
	ClamAV                     ClamAV
	SMTP                       SMTP
}
type TrustedRoutine struct {
	Revision       string
	ValidatedAt    time.Time
	EvidenceSHA256 string
}
type S3 struct {
	Endpoint, Region, Bucket, AccessKey, SecretKey string
	Secure, CreateBucket                           bool
}
type ClamAV struct {
	Address string
	Timeout time.Duration
}
type SMTP struct {
	Address, Username, Password, From string
	TLSMode, TLSServerName, TLSCAFile string
	DevelopmentLog                    bool
}

type Environment string

const (
	Development Environment = "development"
	Test        Environment = "test"
	Production  Environment = "production"
)

type Server struct {
	Host              string
	Port              int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	ReadinessTimeout  time.Duration
	MaxHeaderBytes    int
}

func (s Server) Address() string { return net.JoinHostPort(s.Host, strconv.Itoa(s.Port)) }

type HTTP struct {
	MaxBodyBytes      int64
	AllowedOrigins    []string
	TrustedProxyCIDRs []string
}

type Log struct {
	Level  string
	Format string
}

type Database struct {
	URL              string
	MaxConnections   int32
	QueryTimeout     time.Duration
	MigrationTimeout time.Duration
	AutoMigrate      bool
}

// Load reads configuration without loading dotenv files or mutating the process.
func Load(lookup LookupEnv) (Config, error) {
	if lookup == nil {
		return Config{}, errors.New("config: environment lookup is required")
	}

	cfg := Config{
		Environment: Environment(value(lookup, "APP_ENV", string(Development))),
		Server: Server{
			Host:              value(lookup, "SERVER_HOST", defaultHost),
			Port:              defaultPort,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
			ReadTimeout:       defaultReadTimeout,
			WriteTimeout:      defaultWriteTimeout,
			IdleTimeout:       defaultIdleTimeout,
			ShutdownTimeout:   defaultShutdownTimeout,
			ReadinessTimeout:  defaultReadinessTimeout,
			MaxHeaderBytes:    defaultMaxHeaderBytes,
		},
		HTTP: HTTP{MaxBodyBytes: defaultMaxBodyBytes},
		Log:  Log{Level: value(lookup, "LOG_LEVEL", "info"), Format: value(lookup, "LOG_FORMAT", "text")},
		Database: Database{
			URL:              value(lookup, "DATABASE_URL", "postgres://postgres:postgres@localhost:5432/app?sslmode=disable"),
			MaxConnections:   10,
			QueryTimeout:     5 * time.Second,
			MigrationTimeout: time.Minute,
			AutoMigrate:      true,
		},
		ClaimBounty: ClaimBounty{Enabled: true, CanonicalOrigin: "http://127.0.0.1:5173", SessionPepper: "development-only-identity-token-pepper", AdminEmails: []string{"admin@example.test"}, AuthorizationVersion: "admin-policy-v1", AdminAllowlistVersion: "admin-allowlist-v1", RetentionPolicyVersion: "intake-30d-v1", SourceRetentionMaxDuration: 30 * 24 * time.Hour, PIIRetentionMaxDuration: 30 * 24 * time.Hour, WorkerInterval: 2 * time.Second, RetentionBatch: 25, RetentionTimeout: 10 * time.Minute, AbandonedAfter: 7 * 24 * time.Hour, EmailEncryptionKey: base64.StdEncoding.EncodeToString([]byte("development-email-encryption-key")), EmailLookupKey: base64.StdEncoding.EncodeToString([]byte("development-email-lookup-key-0001")), TrustedRoutine: TrustedRoutine{Revision: "sha256:" + strings.Repeat("a", 64), ValidatedAt: time.Date(2026, 8, 30, 11, 50, 0, 0, time.UTC), EvidenceSHA256: strings.Repeat("b", 64)}, S3: S3{Endpoint: "http://127.0.0.1:9000", Region: "us-east-1", Bucket: "claimbounty-private", AccessKey: "claimbounty-demo", SecretKey: "development-only-object-storage-key", CreateBucket: true}, ClamAV: ClamAV{Address: "127.0.0.1:3310", Timeout: 2 * time.Minute}, SMTP: SMTP{Address: "127.0.0.1:1025", From: "no-reply@claimbounty.test", TLSMode: "none"}},
	}
	if cfg.Environment == Production {
		cfg.Log.Format = value(lookup, "LOG_FORMAT", "json")
		cfg.Database.AutoMigrate = false
	}

	var err error
	if cfg.Server.Port, err = integer(lookup, "SERVER_PORT", defaultPort); err != nil {
		return Config{}, err
	}
	if cfg.Server.ReadHeaderTimeout, err = duration(lookup, "SERVER_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.ReadTimeout, err = duration(lookup, "SERVER_READ_TIMEOUT", defaultReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.WriteTimeout, err = duration(lookup, "SERVER_WRITE_TIMEOUT", defaultWriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.IdleTimeout, err = duration(lookup, "SERVER_IDLE_TIMEOUT", defaultIdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.ShutdownTimeout, err = duration(lookup, "SERVER_SHUTDOWN_TIMEOUT", defaultShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.ReadinessTimeout, err = duration(lookup, "SERVER_READINESS_TIMEOUT", defaultReadinessTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.MaxHeaderBytes, err = integer(lookup, "HTTP_MAX_HEADER_BYTES", defaultMaxHeaderBytes); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.MaxBodyBytes, err = integer64(lookup, "HTTP_MAX_BODY_BYTES", defaultMaxBodyBytes); err != nil {
		return Config{}, err
	}
	if raw, ok := lookup("CORS_ALLOWED_ORIGINS"); ok {
		cfg.HTTP.AllowedOrigins = splitList(raw)
	}
	if raw, ok := lookup("HTTP_TRUSTED_PROXY_CIDRS"); ok {
		cfg.HTTP.TrustedProxyCIDRs = splitList(raw)
	}
	if maxConnections, err := integer(lookup, "DATABASE_MAX_CONNECTIONS", int(cfg.Database.MaxConnections)); err != nil {
		return Config{}, err
	} else {
		if maxConnections < 1 || maxConnections > 100 {
			return Config{}, errors.New("config: DATABASE_MAX_CONNECTIONS must be between 1 and 100")
		}
		cfg.Database.MaxConnections = int32(maxConnections)
	}
	if cfg.Database.QueryTimeout, err = duration(lookup, "DATABASE_QUERY_TIMEOUT", cfg.Database.QueryTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Database.MigrationTimeout, err = duration(lookup, "DATABASE_MIGRATION_TIMEOUT", cfg.Database.MigrationTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Database.AutoMigrate, err = boolean(lookup, "DATABASE_AUTO_MIGRATE", cfg.Database.AutoMigrate); err != nil {
		return Config{}, err
	}
	if cfg.ClaimBounty.Enabled, err = boolean(lookup, "CLAIMBOUNTY_ENABLED", cfg.ClaimBounty.Enabled); err != nil {
		return Config{}, err
	}
	if cfg.Environment == Production && cfg.ClaimBounty.Enabled {
		for _, name := range []string{"PII_RETENTION_POLICY_VERSION", "SOURCE_RETENTION_MAX_DURATION", "PII_RETENTION_MAX_DURATION"} {
			if raw, ok := lookup(name); !ok || strings.TrimSpace(raw) == "" {
				return Config{}, fmt.Errorf("config: %s is required in production", name)
			}
		}
	}
	cfg.ClaimBounty.CanonicalOrigin = value(lookup, "CLAIMBOUNTY_CANONICAL_ORIGIN", cfg.ClaimBounty.CanonicalOrigin)
	cfg.ClaimBounty.SessionPepper = value(lookup, "CLAIMBOUNTY_SESSION_PEPPER", cfg.ClaimBounty.SessionPepper)
	if raw, ok := lookup("CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64"); ok && strings.TrimSpace(raw) != "" {
		cfg.ClaimBounty.EmailEncryptionKey = strings.TrimSpace(raw)
	} else if cfg.Environment == Production && cfg.ClaimBounty.Enabled {
		return Config{}, errors.New("config: CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64 is required in production")
	}
	if raw, ok := lookup("CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64"); ok && strings.TrimSpace(raw) != "" {
		cfg.ClaimBounty.EmailLookupKey = strings.TrimSpace(raw)
	} else if cfg.Environment == Production && cfg.ClaimBounty.Enabled {
		return Config{}, errors.New("config: CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64 is required in production")
	}
	if raw, ok := lookup("CLAIMBOUNTY_ADMIN_EMAILS"); ok {
		cfg.ClaimBounty.AdminEmails = splitList(raw)
	}
	cfg.ClaimBounty.AuthorizationVersion = value(lookup, "CLAIMBOUNTY_AUTHORIZATION_VERSION", cfg.ClaimBounty.AuthorizationVersion)
	cfg.ClaimBounty.AdminAllowlistVersion = value(lookup, "CLAIMBOUNTY_ADMIN_ALLOWLIST_VERSION", cfg.ClaimBounty.AdminAllowlistVersion)
	cfg.ClaimBounty.RetentionPolicyVersion = value(lookup, "PII_RETENTION_POLICY_VERSION", cfg.ClaimBounty.RetentionPolicyVersion)
	if cfg.ClaimBounty.SourceRetentionMaxDuration, err = duration(lookup, "SOURCE_RETENTION_MAX_DURATION", cfg.ClaimBounty.SourceRetentionMaxDuration); err != nil {
		return Config{}, err
	}
	if cfg.ClaimBounty.PIIRetentionMaxDuration, err = duration(lookup, "PII_RETENTION_MAX_DURATION", cfg.ClaimBounty.PIIRetentionMaxDuration); err != nil {
		return Config{}, err
	}
	if cfg.ClaimBounty.WorkerInterval, err = duration(lookup, "CLAIMBOUNTY_WORKER_INTERVAL", cfg.ClaimBounty.WorkerInterval); err != nil {
		return Config{}, err
	}
	if cfg.ClaimBounty.RetentionBatch, err = integer(lookup, "RETENTION_BATCH_SIZE", cfg.ClaimBounty.RetentionBatch); err != nil {
		return Config{}, err
	}
	if cfg.ClaimBounty.RetentionTimeout, err = duration(lookup, "RETENTION_COMMAND_TIMEOUT", cfg.ClaimBounty.RetentionTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ClaimBounty.AbandonedAfter, err = duration(lookup, "CLAIMBOUNTY_ABANDONED_AFTER", cfg.ClaimBounty.AbandonedAfter); err != nil {
		return Config{}, err
	}
	cfg.ClaimBounty.TrustedRoutine.Revision = value(lookup, "CLAIMBOUNTY_ROUTINE_REVISION", cfg.ClaimBounty.TrustedRoutine.Revision)
	if cfg.ClaimBounty.TrustedRoutine.ValidatedAt, err = timestamp(lookup, "CLAIMBOUNTY_ROUTINE_VALIDATED_AT", cfg.ClaimBounty.TrustedRoutine.ValidatedAt); err != nil {
		return Config{}, err
	}
	cfg.ClaimBounty.TrustedRoutine.EvidenceSHA256 = value(lookup, "CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256", cfg.ClaimBounty.TrustedRoutine.EvidenceSHA256)
	cfg.ClaimBounty.S3.Endpoint = value(lookup, "CLAIMBOUNTY_S3_ENDPOINT", cfg.ClaimBounty.S3.Endpoint)
	cfg.ClaimBounty.S3.Region = value(lookup, "CLAIMBOUNTY_S3_REGION", cfg.ClaimBounty.S3.Region)
	cfg.ClaimBounty.S3.Bucket = value(lookup, "CLAIMBOUNTY_S3_BUCKET", cfg.ClaimBounty.S3.Bucket)
	cfg.ClaimBounty.S3.AccessKey = value(lookup, "CLAIMBOUNTY_S3_ACCESS_KEY", cfg.ClaimBounty.S3.AccessKey)
	cfg.ClaimBounty.S3.SecretKey = value(lookup, "CLAIMBOUNTY_S3_SECRET_KEY", cfg.ClaimBounty.S3.SecretKey)
	if cfg.ClaimBounty.S3.Secure, err = boolean(lookup, "CLAIMBOUNTY_S3_SECURE", false); err != nil {
		return Config{}, err
	}
	if cfg.ClaimBounty.S3.CreateBucket, err = boolean(lookup, "CLAIMBOUNTY_S3_CREATE_BUCKET", cfg.ClaimBounty.S3.CreateBucket); err != nil {
		return Config{}, err
	}
	cfg.ClaimBounty.ClamAV.Address = value(lookup, "CLAIMBOUNTY_CLAMAV_ADDRESS", cfg.ClaimBounty.ClamAV.Address)
	if cfg.ClaimBounty.ClamAV.Timeout, err = duration(lookup, "CLAIMBOUNTY_CLAMAV_TIMEOUT", cfg.ClaimBounty.ClamAV.Timeout); err != nil {
		return Config{}, err
	}
	cfg.ClaimBounty.SMTP.Address = value(lookup, "CLAIMBOUNTY_SMTP_ADDRESS", cfg.ClaimBounty.SMTP.Address)
	cfg.ClaimBounty.SMTP.Username = value(lookup, "CLAIMBOUNTY_SMTP_USERNAME", "")
	cfg.ClaimBounty.SMTP.Password = value(lookup, "CLAIMBOUNTY_SMTP_PASSWORD", "")
	cfg.ClaimBounty.SMTP.From = value(lookup, "CLAIMBOUNTY_SMTP_FROM", cfg.ClaimBounty.SMTP.From)
	cfg.ClaimBounty.SMTP.TLSMode = strings.ToLower(value(lookup, "CLAIMBOUNTY_SMTP_TLS_MODE", cfg.ClaimBounty.SMTP.TLSMode))
	cfg.ClaimBounty.SMTP.TLSServerName = value(lookup, "CLAIMBOUNTY_SMTP_TLS_SERVER_NAME", "")
	cfg.ClaimBounty.SMTP.TLSCAFile = value(lookup, "CLAIMBOUNTY_SMTP_TLS_CA_FILE", "")
	if cfg.ClaimBounty.SMTP.DevelopmentLog, err = boolean(lookup, "CLAIMBOUNTY_SMTP_DEVELOPMENT_LOG", cfg.ClaimBounty.SMTP.DevelopmentLog); err != nil {
		return Config{}, err
	}
	if cfg.Environment == Production {
		if err := requireExplicitProductionSettings(lookup, cfg.ClaimBounty.Enabled); err != nil {
			return Config{}, err
		}
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Environment != Development && c.Environment != Test && c.Environment != Production {
		return fmt.Errorf("config: APP_ENV must be development, test, or production")
	}
	if net.ParseIP(c.Server.Host) == nil && !validHostname(c.Server.Host) {
		return errors.New("config: SERVER_HOST must be an IP address or hostname")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("config: SERVER_PORT must be between 1 and 65535")
	}
	for name, timeout := range map[string]time.Duration{
		"SERVER_READ_HEADER_TIMEOUT": c.Server.ReadHeaderTimeout,
		"SERVER_READ_TIMEOUT":        c.Server.ReadTimeout,
		"SERVER_WRITE_TIMEOUT":       c.Server.WriteTimeout,
		"SERVER_IDLE_TIMEOUT":        c.Server.IdleTimeout,
		"SERVER_SHUTDOWN_TIMEOUT":    c.Server.ShutdownTimeout,
		"SERVER_READINESS_TIMEOUT":   c.Server.ReadinessTimeout,
	} {
		if timeout <= 0 {
			return fmt.Errorf("config: %s must be positive", name)
		}
	}
	if c.Server.MaxHeaderBytes < 1024 || c.Server.MaxHeaderBytes > 16<<20 {
		return errors.New("config: HTTP_MAX_HEADER_BYTES must be between 1KiB and 16MiB")
	}
	if c.HTTP.MaxBodyBytes < 1 || c.HTTP.MaxBodyBytes > maxBodyBytes {
		return errors.New("config: HTTP_MAX_BODY_BYTES must be between 1 byte and 256MiB")
	}
	if c.Log.Level != "debug" && c.Log.Level != "info" && c.Log.Level != "warn" && c.Log.Level != "error" {
		return errors.New("config: LOG_LEVEL must be debug, info, warn, or error")
	}
	if c.Log.Format != "text" && c.Log.Format != "json" {
		return errors.New("config: LOG_FORMAT must be text or json")
	}
	parsedDatabaseURL, err := url.Parse(c.Database.URL)
	if err != nil || (parsedDatabaseURL.Scheme != "postgres" && parsedDatabaseURL.Scheme != "postgresql") || parsedDatabaseURL.Host == "" || strings.Trim(parsedDatabaseURL.Path, "/") == "" || parsedDatabaseURL.Fragment != "" {
		return errors.New("config: DATABASE_URL must be a PostgreSQL URL with a database name")
	}
	if c.Environment == Production {
		switch parsedDatabaseURL.Query().Get("sslmode") {
		case "require", "verify-ca", "verify-full":
		default:
			return errors.New("config: production DATABASE_URL must require TLS with sslmode=require, verify-ca, or verify-full")
		}
		if c.Database.AutoMigrate {
			return errors.New("config: DATABASE_AUTO_MIGRATE must be false in production")
		}
	}
	if c.Database.MaxConnections < 1 || c.Database.MaxConnections > 100 {
		return errors.New("config: DATABASE_MAX_CONNECTIONS must be between 1 and 100")
	}
	if c.Database.QueryTimeout <= 0 || c.Database.QueryTimeout > time.Minute {
		return errors.New("config: DATABASE_QUERY_TIMEOUT must be between 1ns and 1m")
	}
	if c.Database.MigrationTimeout <= 0 || c.Database.MigrationTimeout > 15*time.Minute {
		return errors.New("config: DATABASE_MIGRATION_TIMEOUT must be between 1ns and 15m")
	}
	for _, origin := range c.HTTP.AllowedOrigins {
		if origin == "*" {
			if c.Environment == Production {
				return errors.New("config: wildcard CORS origin is forbidden in production")
			}
			continue
		}
		parsed, err := url.ParseRequestURI(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("config: invalid CORS origin %q", origin)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("config: CORS origin %q must use http or https", origin)
		}
	}
	for _, cidr := range c.HTTP.TrustedProxyCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("config: invalid trusted proxy CIDR %q", cidr)
		}
	}
	if c.ClaimBounty.Enabled {
		origin, err := url.ParseRequestURI(c.ClaimBounty.CanonicalOrigin)
		if err != nil || origin.Scheme == "" || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
			return errors.New("config: CLAIMBOUNTY_CANONICAL_ORIGIN must be an origin URL")
		}
		if len(c.ClaimBounty.SessionPepper) < 32 {
			return errors.New("config: CLAIMBOUNTY_SESSION_PEPPER must contain at least 32 characters")
		}
		encryptionKey, encryptionErr := base64.StdEncoding.DecodeString(c.ClaimBounty.EmailEncryptionKey)
		lookupKey, lookupErr := base64.StdEncoding.DecodeString(c.ClaimBounty.EmailLookupKey)
		if encryptionErr != nil || len(encryptionKey) != 32 || lookupErr != nil || len(lookupKey) < 32 || string(encryptionKey) == string(lookupKey) {
			return errors.New("config: ClaimBounty email encryption and lookup keys must be distinct base64-encoded keys of at least 32 bytes")
		}
		if c.Environment == Production && (strings.HasPrefix(string(encryptionKey), "development-") || strings.HasPrefix(string(lookupKey), "development-")) {
			return errors.New("config: production ClaimBounty requires non-development email protection keys")
		}
		if len(c.ClaimBounty.AdminEmails) == 0 {
			return errors.New("config: CLAIMBOUNTY_ADMIN_EMAILS must contain at least one address")
		}
		if !validPolicyVersion(c.ClaimBounty.RetentionPolicyVersion) {
			return errors.New("config: PII_RETENTION_POLICY_VERSION must be a version identifier")
		}
		if c.ClaimBounty.SourceRetentionMaxDuration <= 0 || c.ClaimBounty.PIIRetentionMaxDuration <= 0 || c.ClaimBounty.SourceRetentionMaxDuration > c.ClaimBounty.PIIRetentionMaxDuration {
			return errors.New("config: SOURCE_RETENTION_MAX_DURATION and PII_RETENTION_MAX_DURATION must be positive and source retention must not exceed PII retention")
		}
		if c.ClaimBounty.WorkerInterval <= 0 || c.ClaimBounty.ClamAV.Timeout <= 0 || c.ClaimBounty.RetentionBatch < 1 || c.ClaimBounty.RetentionBatch > 1000 || c.ClaimBounty.RetentionTimeout <= 0 || c.ClaimBounty.AbandonedAfter < 24*time.Hour {
			return errors.New("config: ClaimBounty worker and scanner timeouts must be positive")
		}
		if !validRevision(c.ClaimBounty.TrustedRoutine.Revision) || c.ClaimBounty.TrustedRoutine.ValidatedAt.IsZero() || c.ClaimBounty.TrustedRoutine.ValidatedAt.After(time.Now().UTC()) || !validSHA256(c.ClaimBounty.TrustedRoutine.EvidenceSHA256) {
			return errors.New("config: trusted ClaimBounty routine revision, validation timestamp, and evidence hash are required")
		}
		if c.ClaimBounty.S3.Endpoint == "" || c.ClaimBounty.S3.Bucket == "" || c.ClaimBounty.S3.AccessKey == "" || c.ClaimBounty.S3.SecretKey == "" {
			return errors.New("config: ClaimBounty S3 configuration is incomplete")
		}
		storageEndpoint, storageErr := url.Parse(c.ClaimBounty.S3.Endpoint)
		if storageErr != nil || storageEndpoint.Host == "" || storageEndpoint.Path != "" || (storageEndpoint.Scheme != "http" && storageEndpoint.Scheme != "https") || (storageEndpoint.Scheme == "https") != c.ClaimBounty.S3.Secure {
			return errors.New("config: ClaimBounty S3 endpoint and TLS setting are inconsistent")
		}
		if c.ClaimBounty.SMTP.Address == "" || c.ClaimBounty.SMTP.From == "" {
			return errors.New("config: ClaimBounty SMTP configuration is incomplete")
		}
		switch c.ClaimBounty.SMTP.TLSMode {
		case "none":
		case "starttls", "implicit":
			if c.ClaimBounty.SMTP.TLSServerName == "" {
				return errors.New("config: CLAIMBOUNTY_SMTP_TLS_SERVER_NAME is required for encrypted SMTP")
			}
		default:
			return errors.New("config: CLAIMBOUNTY_SMTP_TLS_MODE must be none, starttls, or implicit")
		}
		if c.Environment == Production && (!c.ClaimBounty.S3.Secure || c.ClaimBounty.S3.CreateBucket) {
			return errors.New("config: production ClaimBounty requires TLS object storage and a pre-created bucket")
		}
		if c.Environment == Production {
			if err := validateProductionSecurity(c); err != nil {
				return err
			}
		}
	}
	return nil
}

func boolean(lookup LookupEnv, name string, fallback bool) (bool, error) {
	raw, ok := lookup(name)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("config: %s must be a boolean: %w", name, err)
	}
	return parsed, nil
}

func value(lookup LookupEnv, name, fallback string) string {
	if raw, ok := lookup(name); ok && strings.TrimSpace(raw) != "" {
		return strings.TrimSpace(raw)
	}
	return fallback
}

func integer(lookup LookupEnv, name string, fallback int) (int, error) {
	raw, ok := lookup(name)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", name, err)
	}
	return v, nil
}

func integer64(lookup LookupEnv, name string, fallback int64) (int64, error) {
	raw, ok := lookup(name)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", name, err)
	}
	return v, nil
}

func duration(lookup LookupEnv, name string, fallback time.Duration) (time.Duration, error) {
	raw, ok := lookup(name)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a Go duration such as 5s: %w", name, err)
	}
	return v, nil
}

func timestamp(lookup LookupEnv, name string, fallback time.Time) (time.Time, error) {
	raw, ok := lookup(name)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("config: %s must be an RFC 3339 timestamp: %w", name, err)
	}
	return parsed.UTC(), nil
}

func validRevision(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validSHA256(strings.TrimPrefix(value, "sha256:"))
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func validPolicyVersion(value string) bool {
	if len(value) == 0 || len(value) > 100 || ((value[0] < 'a' || value[0] > 'z') && (value[0] < '0' || value[0] > '9')) {
		return false
	}
	for _, character := range value[1:] {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '.' && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func splitList(raw string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func validHostname(value string) bool {
	if len(value) == 0 || len(value) > 253 || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}
