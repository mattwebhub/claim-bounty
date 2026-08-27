// Package response owns the API response envelope and bounded JSON decoding.
package response

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

const defaultMaxBodyBytes int64 = 1 << 20

type FieldIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorDocument struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	RequestID string       `json:"requestId,omitempty"`
	Details   []FieldIssue `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorDocument `json:"error"`
}

type DataEnvelope struct {
	Data any `json:"data"`
}

// ClientError represents a safe syntactic transport failure.
type ClientError struct {
	Status  int
	Code    string
	Message string
	Cause   error
}

func (e *ClientError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *ClientError) Unwrap() error { return e.Cause }

func WriteData(w http.ResponseWriter, status int, data any) error {
	return WriteJSON(w, status, DataEnvelope{Data: data})
}

func WriteError(w http.ResponseWriter, status int, code, message, requestID string, details []FieldIssue) error {
	return WriteJSON(w, status, ErrorEnvelope{Error: ErrorDocument{
		Code: code, Message: message, RequestID: requestID, Details: details,
	}})
}

func WriteJSON(w http.ResponseWriter, status int, value any) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(value); err != nil {
		return fmt.Errorf("response: encode JSON: %w", err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if _, err := w.Write(body.Bytes()); err != nil {
		return fmt.Errorf("response: write JSON: %w", err)
	}
	return nil
}

// DecodeJSON accepts exactly one JSON value, rejects unknown fields, and caps
// bytes read. maxBytes <= 0 uses a conservative 1 MiB default.
func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any, maxBytes int64) error {
	if destination == nil {
		return errors.New("response: JSON destination is required")
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &ClientError{Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Message: "Content-Type must be application/json"}
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxBodyBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return &ClientError{Status: http.StatusRequestEntityTooLarge, Code: "body_too_large", Message: "request body exceeds the allowed size", Cause: err}
		}
		if errors.Is(err, io.EOF) {
			return &ClientError{Status: http.StatusBadRequest, Code: "empty_body", Message: "request body must contain JSON", Cause: err}
		}
		return &ClientError{Status: http.StatusBadRequest, Code: "invalid_json", Message: "request body contains invalid JSON", Cause: err}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &ClientError{Status: http.StatusBadRequest, Code: "invalid_json", Message: "request body must contain one JSON value", Cause: err}
	}
	return nil
}
