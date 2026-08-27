package domain_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

func TestWorkspaceDocumentRejectsDuplicateObjectIDs(t *testing.T) {
	t.Parallel()

	object := mustWorkspaceObject(t, "object-1", 10, 20)
	_, err := domain.NewWorkspaceDocument(domain.CurrentWorkspaceSchemaVersion, []domain.WorkspaceObject{object, object})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("NewWorkspaceDocument() error = %v, want validation error", err)
	}
}

func TestWorkspaceObjectRejectsUnsafeGeometry(t *testing.T) {
	t.Parallel()

	_, err := domain.NewWorkspaceObject("object-1", "note", "Label", math.NaN(), 0, -1, 20)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("NewWorkspaceObject() error = %v, want validation error", err)
	}
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if !hasIssue(validation.Issues(), "objects[].x") || !hasIssue(validation.Issues(), "objects[].width") {
		t.Fatalf("issues = %#v, want x and width issues", validation.Issues())
	}
}

func TestWorkspaceReplaceDocumentChecksAndIncrementsVersion(t *testing.T) {
	t.Parallel()

	id, _ := domain.NewProjectID(validProjectID)
	createdAt := time.Date(2026, time.August, 27, 9, 0, 0, 0, time.UTC)
	workspace, err := domain.NewWorkspace(id, domain.EmptyWorkspaceDocument(), createdAt)
	if err != nil {
		t.Fatalf("NewWorkspace() error = %v", err)
	}
	document, err := domain.NewWorkspaceDocument(
		domain.CurrentWorkspaceSchemaVersion,
		[]domain.WorkspaceObject{mustWorkspaceObject(t, "object-1", 10, 20)},
	)
	if err != nil {
		t.Fatalf("NewWorkspaceDocument() error = %v", err)
	}

	updatedAt := createdAt.Add(time.Minute)
	updated, err := workspace.ReplaceDocument(document, 1, updatedAt)
	if err != nil {
		t.Fatalf("ReplaceDocument() error = %v", err)
	}
	if got, want := updated.Version(), uint64(2); got != want {
		t.Fatalf("Version() = %d, want %d", got, want)
	}
	if !updated.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("UpdatedAt() = %v, want %v", updated.UpdatedAt(), updatedAt)
	}

	// Returned collections are defensive copies.
	objects := updated.Document().Objects()
	objects[0] = mustWorkspaceObject(t, "replacement", 0, 0)
	if got := updated.Document().Objects()[0].ID(); got != "object-1" {
		t.Fatalf("workspace document mutated through accessor: ID = %q", got)
	}
}

func TestWorkspaceReplaceDocumentReportsVersionConflict(t *testing.T) {
	t.Parallel()

	id, _ := domain.NewProjectID(validProjectID)
	now := time.Now().UTC()
	workspace, err := domain.NewWorkspace(id, domain.EmptyWorkspaceDocument(), now)
	if err != nil {
		t.Fatalf("NewWorkspace() error = %v", err)
	}

	_, err = workspace.ReplaceDocument(domain.EmptyWorkspaceDocument(), 7, now)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("ReplaceDocument() error = %v, want version conflict", err)
	}
	var conflict *domain.VersionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error type = %T, want *VersionConflictError", err)
	}
	if conflict.Expected != 7 || conflict.Actual != 1 {
		t.Fatalf("conflict = %#v, want expected 7 actual 1", conflict)
	}
}

func TestWorkspaceVersionRemainsExactlyRepresentableByJSONClients(t *testing.T) {
	t.Parallel()
	id, _ := domain.NewProjectID(validProjectID)
	now := time.Now().UTC()
	workspace, err := domain.RestoreWorkspace(
		id, domain.EmptyWorkspaceDocument(), domain.MaxWorkspaceVersion, now, now,
	)
	if err != nil {
		t.Fatalf("RestoreWorkspace(max version) error = %v", err)
	}
	_, err = workspace.ReplaceDocument(domain.EmptyWorkspaceDocument(), domain.MaxWorkspaceVersion, now)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("ReplaceDocument(max version) error = %v, want version conflict", err)
	}
	_, err = domain.RestoreWorkspace(
		id, domain.EmptyWorkspaceDocument(), domain.MaxWorkspaceVersion+1, now, now,
	)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("RestoreWorkspace(unsafe version) error = %v, want validation", err)
	}
}

func mustWorkspaceObject(t *testing.T, id string, x, y float64) domain.WorkspaceObject {
	t.Helper()
	object, err := domain.NewWorkspaceObject(id, "note", "Example", x, y, 120, 80)
	if err != nil {
		t.Fatalf("NewWorkspaceObject() error = %v", err)
	}
	return object
}
