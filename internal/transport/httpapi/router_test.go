package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthEndpoints(t *testing.T) {
	t.Parallel()
	registry := NewReadinessRegistry()
	registry.SetAccepting(true)
	router := NewRouter(RouterOptions{Logger: slog.New(slog.DiscardHandler), Readiness: registry})
	for _, test := range []struct {
		path   string
		status int
		body   string
	}{
		{"/health/live", http.StatusOK, `"live"`},
		{"/health/ready", http.StatusOK, `"ready"`},
		{"/missing", http.StatusNotFound, `"not_found"`},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), test.body) {
			t.Errorf("GET %s = %d %s", test.path, recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("X-Request-ID") == "" {
			t.Errorf("GET %s omitted request ID", test.path)
		}
	}
}

func TestReadinessFailureIsSafeAndBounded(t *testing.T) {
	t.Parallel()
	registry := NewReadinessRegistry()
	if err := registry.Register("database", func(context.Context) error { return errors.New("secret host") }); err != nil {
		t.Fatal(err)
	}
	registry.SetAccepting(true)
	router := NewRouter(RouterOptions{Readiness: registry, ReadinessTimeout: time.Second})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable || strings.Contains(recorder.Body.String(), "secret host") {
		t.Fatalf("readiness response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestReadinessCheckPanicDoesNotEscape(t *testing.T) {
	t.Parallel()
	registry := NewReadinessRegistry()
	if err := registry.Register("broken", func(context.Context) error { panic("private value") }); err != nil {
		t.Fatal(err)
	}
	registry.SetAccepting(true)
	if err := registry.Check(context.Background()); err == nil || strings.Contains(err.Error(), "private value") {
		t.Fatalf("Check() error = %v", err)
	}
}
