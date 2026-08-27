package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mattwebhub/micro1-go-template/internal/domain"
	"github.com/mattwebhub/micro1-go-template/internal/services"
)

func TestSaveWorkspaceAppliesDomainVersionAndAtomicRepositoryCheck(t *testing.T) {
	t.Parallel()

	current := mustWorkspace(t)
	document := mustDocument(t, "object-1")
	repository := &fakeWorkspaceRepository{workspace: current}
	service, err := services.NewWorkspaceService(repository, fixedClock{now: fixedTime.Add(1)})
	if err != nil {
		t.Fatalf("NewWorkspaceService() error = %v", err)
	}

	result, err := service.SaveWorkspace(context.Background(), services.SaveWorkspaceCommand{
		ProjectID: mustProjectID(t), ExpectedVersion: 1, Document: document,
	})
	if err != nil {
		t.Fatalf("SaveWorkspace() error = %v", err)
	}
	if repository.saveCalls != 1 || repository.expectedVersion != 1 {
		t.Fatalf("Save() calls/version = %d/%d, want 1/1", repository.saveCalls, repository.expectedVersion)
	}
	if got, want := repository.saved.Version(), uint64(2); got != want {
		t.Fatalf("saved version = %d, want %d", got, want)
	}
	if got, want := result.Workspace.Version(), uint64(2); got != want {
		t.Fatalf("result version = %d, want %d", got, want)
	}
}

func TestSaveWorkspaceRejectsStaleVersionBeforeWrite(t *testing.T) {
	t.Parallel()

	repository := &fakeWorkspaceRepository{workspace: mustWorkspace(t)}
	service, err := services.NewWorkspaceService(repository, fixedClock{now: fixedTime.Add(1)})
	if err != nil {
		t.Fatalf("NewWorkspaceService() error = %v", err)
	}

	_, err = service.SaveWorkspace(context.Background(), services.SaveWorkspaceCommand{
		ProjectID: mustProjectID(t), ExpectedVersion: 9, Document: domain.EmptyWorkspaceDocument(),
	})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("SaveWorkspace() error = %v, want version conflict", err)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("Save() calls = %d, want 0", repository.saveCalls)
	}
}

func TestSaveWorkspacePreservesConcurrentRepositoryConflict(t *testing.T) {
	t.Parallel()

	repository := &fakeWorkspaceRepository{
		workspace: mustWorkspace(t),
		saveErr:   domain.NewVersionConflictError(1, 2),
	}
	service, err := services.NewWorkspaceService(repository, fixedClock{now: fixedTime.Add(1)})
	if err != nil {
		t.Fatalf("NewWorkspaceService() error = %v", err)
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

	repository := &fakeWorkspaceRepository{}
	service, err := services.NewWorkspaceService(repository, fixedClock{now: fixedTime})
	if err != nil {
		t.Fatalf("NewWorkspaceService() error = %v", err)
	}

	_, err = service.SaveWorkspace(context.Background(), services.SaveWorkspaceCommand{})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("SaveWorkspace() error = %v, want validation error", err)
	}
	if repository.getCalls != 0 {
		t.Fatalf("GetByProjectID() calls = %d, want 0", repository.getCalls)
	}
}

type fakeWorkspaceRepository struct {
	workspace       domain.Workspace
	getErr          error
	getCalls        int
	saved           domain.Workspace
	expectedVersion uint64
	saveErr         error
	saveCalls       int
}

func (repository *fakeWorkspaceRepository) GetByProjectID(context.Context, domain.ProjectID) (domain.Workspace, error) {
	repository.getCalls++
	return repository.workspace, repository.getErr
}

func (repository *fakeWorkspaceRepository) Save(
	_ context.Context,
	workspace domain.Workspace,
	expectedVersion uint64,
) (domain.Workspace, error) {
	repository.saveCalls++
	repository.saved = workspace
	repository.expectedVersion = expectedVersion
	if repository.saveErr != nil {
		return domain.Workspace{}, repository.saveErr
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
