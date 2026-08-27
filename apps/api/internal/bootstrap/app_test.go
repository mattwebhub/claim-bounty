package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mattwebhub/micro1-template/apps/api/internal/config"
)

type routeRegistrar func(*http.ServeMux)

func (registrar routeRegistrar) RegisterRoutes(mux *http.ServeMux) { registrar(mux) }

func TestApplicationWiresModuleRoutes(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	application, err := NewApplication(cfg, slog.New(slog.DiscardHandler), Module{
		Name: "example",
		Routes: routeRegistrar(func(mux *http.ServeMux) {
			mux.HandleFunc("GET /api/v1/example", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		}),
	})
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/example", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("route status = %d", recorder.Code)
	}
}

func TestApplicationDrainsModulesInReverseOrder(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	var calls []string
	modules := []Module{
		{Name: "first", Start: func(context.Context) error { calls = append(calls, "start:first"); return nil }, Shutdown: func(context.Context) error { calls = append(calls, "stop:first"); return nil }},
		{Name: "second", Start: func(context.Context) error { calls = append(calls, "start:second"); return nil }, Shutdown: func(context.Context) error { calls = append(calls, "stop:second"); return nil }},
	}
	application, err := NewApplication(cfg, slog.New(slog.DiscardHandler), modules...)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.modules[0].Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := application.modules[1].Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := application.shutdownModules(2); err != nil {
		t.Fatal(err)
	}
	want := []string{"start:first", "start:second", "stop:second", "stop:first"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("lifecycle = %v, want %v", calls, want)
	}
}

func TestParseCommand(t *testing.T) {
	t.Parallel()
	if got, err := parseCommand(nil); err != nil || got != "serve" {
		t.Fatalf("parseCommand(nil) = %q, %v", got, err)
	}
	if got, err := parseCommand([]string{"validate-config"}); err != nil || got != "validate-config" {
		t.Fatalf("parseCommand(validate) = %q, %v", got, err)
	}
	if _, err := parseCommand([]string{"unknown"}); err == nil {
		t.Fatal("parseCommand accepted unknown command")
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
