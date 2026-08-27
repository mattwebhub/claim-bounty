package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, contentType, body, code string
		max                           int64
	}{
		{"valid", "application/json; charset=utf-8", `{"name":"demo"}`, "", 128},
		{"content type", "text/plain", `{}`, "unsupported_media_type", 128},
		{"empty", "application/json", ``, "empty_body", 128},
		{"unknown", "application/json", `{"extra":true}`, "invalid_json", 128},
		{"multiple", "application/json", `{"name":"one"} {"name":"two"}`, "invalid_json", 128},
		{"large", "application/json", `{"name":"long value"}`, "body_too_large", 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			var got struct {
				Name string `json:"name"`
			}
			err := DecodeJSON(httptest.NewRecorder(), req, &got, tt.max)
			if tt.code == "" {
				if err != nil || got.Name != "demo" {
					t.Fatalf("DecodeJSON() = (%+v, %v)", got, err)
				}
				return
			}
			var clientErr *ClientError
			if !errors.As(err, &clientErr) || clientErr.Code != tt.code {
				t.Fatalf("DecodeJSON() error = %v, want code %q", err, tt.code)
			}
		})
	}
}

func TestWriteErrorUsesStableEnvelope(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	if err := WriteError(recorder, http.StatusBadRequest, "invalid_input", "invalid input", "req-1", nil); err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"requestId":"req-1"`) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestWriteJSONDoesNotCommitAnUnencodableValue(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	if err := WriteJSON(recorder, http.StatusOK, make(chan int)); err == nil {
		t.Fatal("WriteJSON() accepted an unencodable value")
	}
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 || recorder.Header().Get("Content-Type") != "" {
		t.Fatalf("response was committed: %d %q", recorder.Code, recorder.Body.String())
	}
}
