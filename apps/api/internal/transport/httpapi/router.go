// Package httpapi owns HTTP protocol composition and process health endpoints.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/middleware"
	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/response"
)

type RouteRegistrar interface {
	RegisterRoutes(*http.ServeMux)
}

type RouterOptions struct {
	Logger           *slog.Logger
	Readiness        *ReadinessRegistry
	ReadinessTimeout time.Duration
	AllowedOrigins   []string
	Registrars       []RouteRegistrar
}

func NewRouter(options RouterOptions) http.Handler {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	readiness := options.Readiness
	if readiness == nil {
		readiness = NewReadinessRegistry()
		readiness.SetAccepting(true)
	}
	if options.ReadinessTimeout <= 0 {
		options.ReadinessTimeout = 2 * time.Second
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		_ = response.WriteData(w, http.StatusOK, map[string]string{"status": "live"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), options.ReadinessTimeout)
		defer cancel()
		if err := readiness.Check(ctx); err != nil {
			_ = response.WriteError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready", middleware.RequestIDFromContext(r.Context()), nil)
			return
		}
		_ = response.WriteData(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	for _, registrar := range options.Registrars {
		if registrar != nil {
			registrar.RegisterRoutes(mux)
		}
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = response.WriteError(w, http.StatusNotFound, "not_found", "resource not found", middleware.RequestIDFromContext(r.Context()), nil)
	})

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Recovery(logger),
		middleware.Security,
		middleware.CORS(options.AllowedOrigins),
		middleware.Logging(logger),
	)
}
