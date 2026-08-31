package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	if got, err := parseCommand([]string{"healthcheck"}); err != nil || got != "healthcheck" {
		t.Fatalf("parseCommand(healthcheck) = %q, %v", got, err)
	}
	if _, err := parseCommand([]string{"unknown"}); err == nil {
		t.Fatal("parseCommand accepted unknown command")
	}
}

func TestRunVerifyExportChecksExpectedWholeArchiveDigestBeforeZIP(t *testing.T) {
	t.Parallel()

	contents := []byte("not a ZIP archive")
	archive := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(archive, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "verified")
	if err := Run(context.Background(), []string{"verify-export", archive, strings.Repeat("0", 64), destination}); err == nil || !strings.Contains(err.Error(), "expected archive SHA-256 mismatch") {
		t.Fatalf("Run() error = %v, want whole-archive digest mismatch", err)
	}
	sum := sha256.Sum256(contents)
	if err := Run(context.Background(), []string{"verify-export", archive, hex.EncodeToString(sum[:]), destination}); err == nil || !strings.Contains(err.Error(), "open ZIP") {
		t.Fatalf("Run() error = %v, want ZIP error after matching digest", err)
	}
}

func TestRunHealthcheckUsesReadinessStatus(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		wantError bool
	}{
		{name: "ready", status: http.StatusOK},
		{name: "not ready", status: http.StatusServiceUnavailable, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health/ready" {
					t.Errorf("healthcheck path = %q", r.URL.Path)
				}
				w.WriteHeader(test.status)
			}))
			defer server.Close()

			cfg := testConfig(t)
			cfg.Server.Port = server.Listener.Addr().(*net.TCPAddr).Port
			err := runHealthcheck(context.Background(), cfg)
			if (err != nil) != test.wantError {
				t.Fatalf("runHealthcheck() error = %v, wantError %v", err, test.wantError)
			}
		})
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
