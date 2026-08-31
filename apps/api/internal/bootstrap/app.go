// Package bootstrap is the composition root and owns process lifecycle.
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	claimexport "github.com/mattwebhub/micro1-template/apps/api/internal/adapters/export"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/postgres"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/validation"
	"github.com/mattwebhub/micro1-template/apps/api/internal/config"
	"github.com/mattwebhub/micro1-template/apps/api/internal/observability"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi"
)

type Module struct {
	Name      string
	Routes    httpapi.RouteRegistrar
	Readiness httpapi.ReadinessCheck
	Start     func(context.Context) error
	Shutdown  func(context.Context) error
}

type Application struct {
	config    config.Config
	logger    *slog.Logger
	modules   []Module
	readiness *httpapi.ReadinessRegistry
	handler   http.Handler
}

func NewApplication(cfg config.Config, logger *slog.Logger, modules ...Module) (*Application, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("bootstrap: invalid config: %w", err)
	}
	if logger == nil {
		return nil, errors.New("bootstrap: logger is required")
	}
	readiness := httpapi.NewReadinessRegistry()
	registrars := make([]httpapi.RouteRegistrar, 0, len(modules))
	allowedOrigins := append([]string(nil), cfg.HTTP.AllowedOrigins...)
	seen := make(map[string]struct{}, len(modules))
	for _, module := range modules {
		if module.Name == "" {
			return nil, errors.New("bootstrap: module name is required")
		}
		if _, exists := seen[module.Name]; exists {
			return nil, fmt.Errorf("bootstrap: duplicate module %q", module.Name)
		}
		seen[module.Name] = struct{}{}
		if module.Routes != nil {
			registrars = append(registrars, module.Routes)
			if provider, ok := module.Routes.(interface{ AllowedOrigins() []string }); ok {
				for _, origin := range provider.AllowedOrigins() {
					if !containsString(allowedOrigins, origin) {
						allowedOrigins = append(allowedOrigins, origin)
					}
				}
			}
		}
		if module.Readiness != nil {
			if err := readiness.Register(module.Name, module.Readiness); err != nil {
				return nil, fmt.Errorf("bootstrap: register %s readiness: %w", module.Name, err)
			}
		}
	}
	handler := httpapi.NewRouter(httpapi.RouterOptions{
		Logger: logger, Readiness: readiness,
		ReadinessTimeout: cfg.Server.ReadinessTimeout,
		AllowedOrigins:   allowedOrigins,
		Registrars:       registrars,
	})
	return &Application{config: cfg, logger: logger, modules: modules, readiness: readiness, handler: handler}, nil
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func (application *Application) Handler() http.Handler { return application.handler }

// Serve starts modules in dependency order and drains them in reverse order.
func (application *Application) Serve(ctx context.Context) error {
	started := 0
	for index := range application.modules {
		if application.modules[index].Start != nil {
			if err := application.modules[index].Start(ctx); err != nil {
				shutdownErr := application.shutdownModules(index)
				return errors.Join(fmt.Errorf("bootstrap: start module %s: %w", application.modules[index].Name, err), shutdownErr)
			}
		}
		started = index + 1
	}

	listener, err := net.Listen("tcp", application.config.Server.Address())
	if err != nil {
		shutdownErr := application.shutdownModules(started)
		return errors.Join(fmt.Errorf("bootstrap: listen: %w", err), shutdownErr)
	}
	server := &http.Server{
		Addr:              application.config.Server.Address(),
		Handler:           application.handler,
		ReadHeaderTimeout: application.config.Server.ReadHeaderTimeout,
		ReadTimeout:       application.config.Server.ReadTimeout,
		WriteTimeout:      application.config.Server.WriteTimeout,
		IdleTimeout:       application.config.Server.IdleTimeout,
		MaxHeaderBytes:    application.config.Server.MaxHeaderBytes,
	}
	application.readiness.SetAccepting(true)
	application.logger.InfoContext(ctx, "HTTP server listening", "address", application.config.Server.Address())

	serveError := make(chan error, 1)
	go func() { serveError <- server.Serve(listener) }()

	select {
	case err := <-serveError:
		application.readiness.SetAccepting(false)
		serverErr := application.shutdownServer(server)
		shutdownErr := application.shutdownModules(started)
		if errors.Is(err, http.ErrServerClosed) {
			return errors.Join(serverErr, shutdownErr)
		}
		return errors.Join(fmt.Errorf("bootstrap: serve HTTP: %w", err), serverErr, shutdownErr)
	case <-ctx.Done():
		application.readiness.SetAccepting(false)
		serverErr := application.shutdownServer(server)
		modulesErr := application.shutdownModules(started)
		if serverErr != nil || modulesErr != nil {
			return errors.Join(serverErr, modulesErr)
		}
		return nil
	}
}

func (application *Application) shutdownServer(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), application.config.Server.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		// Shutdown leaves active connections open when its deadline expires.
		// Force closure before infrastructure modules are drained.
		return errors.Join(err, server.Close())
	}
	return nil
}

func (application *Application) shutdownModules(count int) error {
	ctx, cancel := context.WithTimeout(context.Background(), application.config.Server.ShutdownTimeout)
	defer cancel()
	return application.shutdownModulesWithContext(ctx, count)
}

func (application *Application) shutdownModulesWithContext(ctx context.Context, count int) error {
	var result error
	for index := count - 1; index >= 0; index-- {
		module := application.modules[index]
		if module.Shutdown == nil {
			continue
		}
		if err := module.Shutdown(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("shutdown module %s: %w", module.Name, err))
		}
	}
	return result
}

// Run is the non-interactive process entry point used by the API, worker, and jobs.
func Run(parent context.Context, arguments []string) error {
	if (len(arguments) == 3 || len(arguments) == 4) && arguments[0] == "verify-export" {
		schemas, err := validation.New()
		if err != nil {
			return fmt.Errorf("bootstrap: compile ClaimBounty schemas: %w", err)
		}
		verifier, err := claimexport.NewVerifier(schemas)
		if err != nil {
			return err
		}
		destination := strings.TrimSuffix(arguments[1], filepath.Ext(arguments[1])) + "-verified"
		if len(arguments) == 4 {
			destination = arguments[3]
		}
		paths, err := verifier.VerifyAndExtract(arguments[1], arguments[2], destination)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(struct {
			Status           string `json:"status"`
			Destination      string `json:"destination"`
			CaseBundle       string `json:"caseBundle"`
			AuditRequest     string `json:"auditRequest"`
			ScientificPolicy string `json:"scientificPolicy"`
			ExecutionPolicy  string `json:"executionPolicy"`
		}{
			Status: "verified", Destination: paths.Destination, CaseBundle: paths.CaseBundle,
			AuditRequest: paths.AuditRequest, ScientificPolicy: paths.ScientificPolicy, ExecutionPolicy: paths.ExecutionPolicy,
		})
	}
	command, err := parseCommand(arguments)
	if err != nil {
		return err
	}
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("bootstrap: load config: %w", err)
	}
	if command == "validate-config" {
		return nil
	}
	if command == "healthcheck" {
		return runHealthcheck(parent, cfg)
	}
	if command == "migrate" {
		migrationContext, cancel := context.WithTimeout(parent, cfg.Database.MigrationTimeout)
		defer cancel()
		if err := postgres.Migrate(migrationContext, cfg.Database.URL); err != nil {
			return fmt.Errorf("bootstrap: migrate database: %w", err)
		}
		return nil
	}
	logger, err := observability.NewLogger(os.Stdout, observability.LoggerOptions{
		Level: cfg.Log.Level, Format: cfg.Log.Format,
		Service: "api", Environment: string(cfg.Environment),
	})
	if err != nil {
		return fmt.Errorf("bootstrap: initialize logger: %w", err)
	}
	runtimeContext, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if command == "worker" {
		workers, closeWorkers, err := buildClaimBountyWorkers(runtimeContext, cfg, logger)
		if err != nil {
			return err
		}
		defer closeWorkers()
		return workers.Run(runtimeContext)
	}
	if command == "retention" || command == "retention-cleanup" {
		cleanupContext, cancel := context.WithTimeout(runtimeContext, cfg.ClaimBounty.RetentionTimeout)
		defer cancel()
		return runRetentionCleanup(cleanupContext, cfg)
	}
	projectModule, err := buildProjectModule(parent, cfg, logger)
	if err != nil {
		return err
	}
	modules := []Module{projectModule}
	if cfg.ClaimBounty.Enabled {
		claimModule, claimErr := buildClaimBountyModule(parent, cfg, logger)
		if claimErr != nil {
			_ = projectModule.Shutdown(context.Background())
			return claimErr
		}
		modules = append(modules, claimModule)
	}
	application, err := NewApplication(cfg, logger, modules...)
	if err != nil {
		for index := len(modules) - 1; index >= 0; index-- {
			if modules[index].Shutdown != nil {
				_ = modules[index].Shutdown(context.Background())
			}
		}
		return err
	}
	return application.Serve(runtimeContext)
}

func runHealthcheck(parent context.Context, cfg config.Config) error {
	ctx, cancel := context.WithTimeout(parent, cfg.Server.ReadinessTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health/ready", cfg.Server.Port), nil)
	if err != nil {
		return fmt.Errorf("bootstrap: create readiness request: %w", err)
	}
	response, err := (&http.Client{Timeout: cfg.Server.ReadinessTimeout}).Do(request)
	if err != nil {
		return fmt.Errorf("bootstrap: readiness request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("bootstrap: readiness returned HTTP %d", response.StatusCode)
	}
	return nil
}

func parseCommand(arguments []string) (string, error) {
	if len(arguments) == 0 {
		return "serve", nil
	}
	if len(arguments) == 1 {
		switch arguments[0] {
		case "serve", "migrate", "worker", "retention", "retention-cleanup", "validate-config", "healthcheck":
			return arguments[0], nil
		}
	}
	return "", errors.New("bootstrap: usage: api [serve|migrate|worker|retention|retention-cleanup|validate-config|healthcheck] | api verify-export <archive.zip> <expected-sha256-hex> [new-destination]")
}
