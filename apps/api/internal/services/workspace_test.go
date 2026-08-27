package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
)

func TestSaveWorkspaceAppliesDomainVersionAndAtomicRepositoryCheck(t *testing.T) {
	t.Parallel()

	current := mustWorkspace(t)
	document := mustDocument(t, "object-1")
	reader := &fakeWorkspaceReader{workspace: current}
	saver := &fakeWorkspaceSaver{}
	service, err := services.NewWorkspaceCommandService(reader, saver, fixedClock{now: fixedTime.Add(1)})
	if err != nil {
		t.Fatalf("NewWorkspaceCommandService() error = %v", err)
	}

	result, err := service.SaveWorkspace(context.Background(), services.SaveWorkspaceCommand{
		ProjectID: mustProjectID(t), ExpectedVersion: 1, Document: document,
	})
	if err != nil {
		t.Fatalf("SaveWorkspace() error = %v", err)
	}
	if saver.saveCalls != 1 || saver.expectedVersion != 1 {
		t.Fatalf("Save() calls/version = %d/%d, want 1/1", saver.saveCalls, saver.expectedVersion)
	}
	if got, want := saver.saved.Version(), uint64(2); got != want {
		t.Fatalf("saved version = %d, want %d", got, want)
	}
	if got, want := result.Workspace.Version(), uint64(2); got != want {
		t.Fatalf("result version = %d, want %d", got, want)
	}
}

func TestSaveWorkspaceRejectsStaleVersionBeforeWrite(t *testing.T) {
	t.Parallel()

	reader := &fakeWorkspaceReader{workspace: mustWorkspace(t)}
	saver := &fakeWorkspaceSaver{}
	service, err := services.NewWorkspaceCommandService(reader, saver, fixedClock{now: fixedTime.Add(1)})
	if err != nil {
		t.Fatalf("NewWorkspaceCommandService() error = %v", err)
	}

	_, err = service.SaveWorkspace(context.Background(), services.SaveWorkspaceCommand{
		ProjectID: mustProjectID(t), ExpectedVersion: 9, Document: domain.EmptyWorkspaceDocument(),
	})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("SaveWorkspace() error = %v, want version conflict", err)
	}
	if saver.saveCalls != 0 {
		t.Fatalf("Save() calls = %d, want 0", saver.saveCalls)
	}
}

func TestSaveWorkspacePreservesConcurrentRepositoryConflict(t *testing.T) {
	t.Parallel()

	reader := &fakeWorkspaceReader{workspace: mustWorkspace(t)}
	saver := &fakeWorkspaceSaver{saveErr: domain.NewVersionConflictError(1, 2)}
	service, err := services.NewWorkspaceCommandService(reader, saver, fixedClock{now: fixedTime.Add(1)})
	if err != nil {
		t.Fatalf("NewWorkspaceCommandService() error = %v", err)
	}

	_, err = service.SaveWorkspace(context.Background(), services.SaveWorkspaceCommand{
		ProjectID: mustProjectID(t), ExpectedVersion: 1, Document: domain.EmptyWorkspaceDocument(),
	})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("SaveWorkspace() error = %v, want version conflict", err)
	}
}

func TestSaveWorkspaceRejectsMissingVersionBeforeIO(t *testing.T) {
	t.Parallel()

	reader := &fakeWorkspaceReader{}
	saver := &fakeWorkspaceSaver{}
	service, err := services.NewWorkspaceCommandService(reader, saver, fixedClock{now: fixedTime})
	if err != nil {
		t.Fatalf("NewWorkspaceCommandService() error = %v", err)
	}

	_, err = service.SaveWorkspace(context.Background(), services.SaveWorkspaceCommand{})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("SaveWorkspace() error = %v, want validation error", err)
	}
	if reader.getCalls != 0 {
		t.Fatalf("GetByProjectID() calls = %d, want 0", reader.getCalls)
	}
}

func TestGetWorkspaceUsesReadCapabilityOnly(t *testing.T) {
	t.Parallel()

	workspace := mustWorkspace(t)
	reader := &fakeWorkspaceReader{workspace: workspace}
	service, err := services.NewWorkspaceQueryService(reader)
	if err != nil {
		t.Fatalf("NewWorkspaceQueryService() error = %v", err)
	}

	result, err := service.GetWorkspace(context.Background(), services.GetWorkspaceQuery{ProjectID: mustProjectID(t)})
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	if reader.getCalls != 1 || result.Workspace.ProjectID() != workspace.ProjectID() {
		t.Fatalf("GetWorkspace() calls/result = %d/%v", reader.getCalls, result.Workspace.ProjectID())
	}
}

func TestWorkspaceServiceConstructorsRejectMissingCapabilities(t *testing.T) {
	t.Parallel()

	reader := &fakeWorkspaceReader{}
	saver := &fakeWorkspaceSaver{}
	clock := fixedClock{now: fixedTime}
	if _, err := services.NewWorkspaceQueryService(nil); !errors.Is(err, services.ErrInvalidDependencies) {
		t.Fatalf("NewWorkspaceQueryService() error = %v, want invalid dependencies", err)
	}
	for name, build := range map[string]func() error{
		"reader": func() error { _, err := services.NewWorkspaceCommandService(nil, saver, clock); return err },
		"saver":  func() error { _, err := services.NewWorkspaceCommandService(reader, nil, clock); return err },
		"clock":  func() error { _, err := services.NewWorkspaceCommandService(reader, saver, nil); return err },
	} {
		if err := build(); !errors.Is(err, services.ErrInvalidDependencies) {
			t.Errorf("missing %s error = %v, want invalid dependencies", name, err)
		}
	}
}

type fakeWorkspaceReader struct {
	workspace domain.Workspace
	getErr    error
	getCalls  int
}

func (reader *fakeWorkspaceReader) GetByProjectID(context.Context, domain.ProjectID) (domain.Workspace, error) {
	reader.getCalls++
	return reader.workspace, reader.getErr
}

type fakeWorkspaceSaver struct {
	saved           domain.Workspace
	expectedVersion uint64
	saveErr         error
	saveCalls       int
}

func (saver *fakeWorkspaceSaver) Save(
	_ context.Context,
	workspace domain.Workspace,
	expectedVersion uint64,
) (domain.Workspace, error) {
	saver.saveCalls++
	saver.saved = workspace
	saver.expectedVersion = expectedVersion
	if saver.saveErr != nil {
		return domain.Workspace{}, saver.saveErr
	}
	return workspace, nil
}

func mustWorkspace(t *testing.T) domain.Workspace {
	t.Helper()
	workspace, err := domain.NewWorkspace(mustProjectID(t), domain.EmptyWorkspaceDocument(), fixedTime)
	if err != nil {
		t.Fatalf("NewWorkspace() error = %v", err)
	}
	return workspace
}

func mustDocument(t *testing.T, objectID string) domain.WorkspaceDocument {
	t.Helper()
	object, err := domain.NewWorkspaceObject(objectID, "note", "Example", 10, 20, 100, 80)
	if err != nil {
		t.Fatalf("NewWorkspaceObject() error = %v", err)
	}
	document, err := domain.NewWorkspaceDocument(domain.CurrentWorkspaceSchemaVersion, []domain.WorkspaceObject{object})
	if err != nil {
		t.Fatalf("NewWorkspaceDocument() error = %v", err)
	}
	return document
}
