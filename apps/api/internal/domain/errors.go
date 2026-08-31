package domain

import (
	"errors"
	"fmt"
)

// Stable error categories are intentionally small. Transports map these
// categories to protocol-specific errors without inspecting error strings.
var (
	ErrValidation        = errors.New("validation failed")
	ErrProjectNotFound   = errors.New("project not found")
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrProjectExists     = errors.New("project already exists")
	ErrVersionConflict   = errors.New("workspace version conflict")
	ErrUnauthorized      = errors.New("authentication required")
	ErrForbidden         = errors.New("operation forbidden")
	ErrOrderNotFound     = errors.New("order not found")
	ErrFileNotFound      = errors.New("file not found")
	ErrExportNotFound    = errors.New("export not found")
	ErrInvalidChallenge  = errors.New("email verification invalid")
	ErrRateLimited       = errors.New("rate limited")
	ErrStateConflict     = errors.New("resource state conflict")
	ErrFileNotClean      = errors.New("file not clean")
	ErrExportNotReady    = errors.New("export not ready")
	ErrIdempotency       = errors.New("idempotency conflict")
)

// FieldIssue is a stable, machine-readable description of invalid domain
// input. Message is safe to expose to a caller; Code and Field are the stable
// contract.
type FieldIssue struct {
	Field   string
	Code    string
	Message string
}

// ValidationError groups every invariant violation discovered in one pass.
type ValidationError struct {
	issues []FieldIssue
}

func NewValidationError(issues ...FieldIssue) error {
	if len(issues) == 0 {
		return nil
	}

	return &ValidationError{issues: append([]FieldIssue(nil), issues...)}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %d field issue(s)", ErrValidation, len(e.issues))
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

// Issues returns a copy so callers cannot mutate the error after construction.
func (e *ValidationError) Issues() []FieldIssue {
	return append([]FieldIssue(nil), e.issues...)
}

// VersionConflictError contains the versions needed to explain and recover
// from an optimistic-concurrency failure.
type VersionConflictError struct {
	Expected uint64
	Actual   uint64
}

func NewVersionConflictError(expected, actual uint64) error {
	return &VersionConflictError{Expected: expected, Actual: actual}
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("%s: expected %d, actual %d", ErrVersionConflict, e.Expected, e.Actual)
}

func (e *VersionConflictError) Unwrap() error { return ErrVersionConflict }
