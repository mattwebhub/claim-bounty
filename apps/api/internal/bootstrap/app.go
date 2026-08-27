// Package bootstrap is the composition root and owns process lifecycle.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattwebhub/micro1-go-template/internal/config"
	"github.com/mattwebhub/micro1-go-template/internal/observability"
	"github.com/mattwebhub/micro1-go-template/internal/transport/httpapi"
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
		AllowedOrigins:   cfg.HTTP.AllowedOrigins,
		Registrars:       registrars,
	})
	return &Application{config: cfg, logger: logger, modules: modules, readiness: readiness, handler: handler}, nil
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

// Run is the process entry point. Supported commands are serve (default) and
// validate-config, which performs an offline deployment preflight.
func Run(parent context.Context, arguments []string) error {
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
	logger, err := observability.NewLogger(os.Stdout, observability.LoggerOptions{
		Level: cfg.Log.Level, Format: cfg.Log.Format,
		Service: "api", Environment: string(cfg.Environment),
	})
	if err != nil {
		return fmt.Errorf("bootstrap: initialize logger: %w", err)
	}
	projectModule, err := buildProjectModule(parent, cfg, logger)
	if err != nil {
		return err
	}
	application, err := NewApplication(cfg, logger, projectModule)
	if err != nil {
		_ = projectModule.Shutdown(context.Background())
		return err
	}
	runtimeContext, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	return application.Serve(runtimeContext)
}

func parseCommand(arguments []string) (string, error) {
	if len(arguments) == 0 {
		return "serve", nil
	}
	if len(arguments) == 1 && (arguments[0] == "serve" || arguments[0] == "validate-config") {
		return arguments[0], nil
	}
	return "", errors.New("bootstrap: usage: api [serve|validate-config]")
}
