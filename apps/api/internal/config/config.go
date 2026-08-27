// Package config parses and validates process configuration exactly once.
package config

import (
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
	MaxBodyBytes   int64
	AllowedOrigins []string
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
	}
	if cfg.Environment == Production {
		cfg.Log.Format = value(lookup, "LOG_FORMAT", "json")
		cfg.Database.AutoMigrate = false
		if raw, ok := lookup("DATABASE_URL"); !ok || strings.TrimSpace(raw) == "" {
			return Config{}, errors.New("config: DATABASE_URL is required in production")
		}
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
	if c.HTTP.MaxBodyBytes < 1 || c.HTTP.MaxBodyBytes > 64<<20 {
		return errors.New("config: HTTP_MAX_BODY_BYTES must be between 1 byte and 64MiB")
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
