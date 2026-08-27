package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDPreservesOnlyValidInput(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, supplied, want string }{
		{"valid", "caller-12345678", "caller-12345678"},
		{"invalid", "bad\nvalue", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if RequestIDFromContext(r.Context()) == "" {
					t.Error("request ID missing from context")
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(RequestIDHeader, test.supplied)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			got := recorder.Header().Get(RequestIDHeader)
			if got == "" || (test.want != "" && got != test.want) || (test.want == "" && got == test.supplied) {
				t.Fatalf("request ID = %q", got)
			}
		})
	}
}

func TestRecoveryReturnsSafeError(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("private value") }), RequestID, Recovery(logger))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "private value") {
		t.Fatalf("unsafe response = %d %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(logs.String(), "private value") {
		t.Fatalf("panic value leaked to logs: %s", logs.String())
	}
}

func TestSecurityAndCORS(t *testing.T) {
	t.Parallel()
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), Security, CORS([]string{"https://app.example.com"}))
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent || recorder.Header().Get("Access-Control-Allow-Origin") == "" || recorder.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("headers = %#v", recorder.Header())
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentialed cross-origin requests must require an explicit future policy")
	}
	if exposed := recorder.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(exposed, "ETag") || !strings.Contains(exposed, "Location") {
		t.Fatalf("public response headers are not exposed: %q", exposed)
	}
}

func TestCORSRejectsDisallowedStateChangingOriginWithStableEnvelope(t *testing.T) {
	t.Parallel()
	called := false
	handler := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}), RequestID, Security, CORS([]string{"https://app.example.com"}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"unsafe"}`))
	req.Header.Set("Origin", "https://attacker.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden || called {
		t.Fatalf("response = %d %s, next called = %v", recorder.Code, recorder.Body.String(), called)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"cors_origin_denied"`) || recorder.Header().Get(RequestIDHeader) == "" {
		t.Fatalf("response is not the stable error envelope: %s", recorder.Body.String())
	}
}
