package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
	"github.com/mattwebhub/micro1-template/apps/api/internal/services"
)

const projectIDString = "123e4567-e89b-12d3-a456-426614174000"

var fixedTime = time.Date(2026, time.August, 27, 9, 0, 0, 0, time.UTC)

func TestCreateProjectPersistsProjectAndWorkspaceInOneTransaction(t *testing.T) {
	t.Parallel()

	id := mustProjectID(t)
	projects := &fakeTransactionProjects{}
	workspaces := &fakeTransactionWorkspaces{}
	transactionContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	transactions := &fakeTransactionManager{repositories: fakeTransactionRepositories{
		projects: projects, workspaces: workspaces,
	}, transactionContext: transactionContext}
	service, err := services.NewProjectCommandService(
		transactions,
		fixedIDGenerator{id: id},
		fixedClock{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("NewProjectCommandService() error = %v", err)
	}

	result, err := service.CreateProject(context.Background(), services.CreateProjectCommand{Name: " New project "})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if transactions.calls != 1 {
		t.Fatalf("WithinTransaction() calls = %d, want 1", transactions.calls)
	}
	if projects.created.ID() != id || workspaces.created.ProjectID() != id {
		t.Fatalf("created aggregate IDs do not match generated ID %q", id)
	}
	if projects.ctx != transactionContext || workspaces.ctx != transactionContext {
		t.Fatal("transaction-bound repositories did not receive the callback transaction context")
	}
	if got, want := result.Project.Name(), "New project"; got != want {
		t.Fatalf("project name = %q, want %q", got, want)
	}
	if got, want := workspaces.created.Version(), uint64(1); got != want {
		t.Fatalf("persisted workspace version = %d, want %d", got, want)
	}
}

func TestCreateProjectDoesNotStartTransactionForInvalidDomainInput(t *testing.T) {
	t.Parallel()

	transactions := &fakeTransactionManager{}
	service, err := services.NewProjectCommandService(
		transactions,
		fixedIDGenerator{id: mustProjectID(t)},
		fixedClock{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("NewProjectCommandService() error = %v", err)
	}

	_, err = service.CreateProject(context.Background(), services.CreateProjectCommand{Name: "\n"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("CreateProject() error = %v, want validation error", err)
	}
	if transactions.calls != 0 {
		t.Fatalf("WithinTransaction() calls = %d, want 0", transactions.calls)
	}
}

func TestCreateProjectPreservesRepositoryErrorCategory(t *testing.T) {
	t.Parallel()

	projects := &fakeTransactionProjects{err: domain.ErrProjectExists}
	transactions := &fakeTransactionManager{repositories: fakeTransactionRepositories{
		projects: projects, workspaces: &fakeTransactionWorkspaces{},
	}}
	service, err := services.NewProjectCommandService(
		transactions,
		fixedIDGenerator{id: mustProjectID(t)},
		fixedClock{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("NewProjectCommandService() error = %v", err)
	}

	_, err = service.CreateProject(context.Background(), services.CreateProjectCommand{Name: "Project"})
	if !errors.Is(err, domain.ErrProjectExists) {
		t.Fatalf("CreateProject() error = %v, want project-exists category", err)
	}
}

func TestListProjectsAppliesDefaultPageSizeAndCopiesResults(t *testing.T) {
	t.Parallel()

	project := mustProject(t)
	repository := &fakeProjectRepository{page: ports.ProjectPage{Projects: []domain.Project{project}, NextCursor: "next"}}
	service, err := services.NewProjectQueryService(repository)
	if err != nil {
		t.Fatalf("NewProjectQueryService() error = %v", err)
	}

	result, err := service.ListProjects(context.Background(), services.ListProjectsQuery{Cursor: " cursor "})
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if repository.listRequest.Limit != 20 || repository.listRequest.Cursor != " cursor " {
		t.Fatalf("List() request = %#v, want limit 20 and unchanged opaque cursor", repository.listRequest)
	}
	if len(result.Projects) != 1 || result.NextCursor != "next" {
		t.Fatalf("ListProjects() result = %#v", result)
	}
	result.Projects[0] = domain.Project{}
	if repository.page.Projects[0].ID() == (domain.ProjectID{}) {
		t.Fatal("result aliases repository project slice")
	}
}

func TestListProjectsRejectsOutOfRangeLimitBeforeIO(t *testing.T) {
	t.Parallel()

	repository := &fakeProjectRepository{}
	service, err := services.NewProjectQueryService(repository)
	if err != nil {
		t.Fatalf("NewProjectQueryService() error = %v", err)
	}

	_, err = service.ListProjects(context.Background(), services.ListProjectsQuery{Limit: 101})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ListProjects() error = %v, want validation error", err)
	}
	if repository.listCalls != 0 {
		t.Fatalf("List() calls = %d, want 0", repository.listCalls)
	}
}

func TestServiceConstructorsRejectMissingDependencies(t *testing.T) {
	t.Parallel()

	if _, err := services.NewProjectCommandService(nil, nil, nil); !errors.Is(err, services.ErrInvalidDependencies) {
		t.Fatalf("NewProjectCommandService() error = %v, want invalid dependencies", err)
	}
	if _, err := services.NewProjectQueryService(nil); !errors.Is(err, services.ErrInvalidDependencies) {
		t.Fatalf("NewProjectQueryService() error = %v, want invalid dependencies", err)
	}
}

type fixedIDGenerator struct {
	id  domain.ProjectID
	err error
}

func (generator fixedIDGenerator) NewProjectID(context.Context) (domain.ProjectID, error) {
	return generator.id, generator.err
}

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

type fakeTransactionManager struct {
	repositories       ports.TransactionRepositories
	transactionContext context.Context
	err                error
	calls              int
}

func (manager *fakeTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(context.Context, ports.TransactionRepositories) error,
) error {
	manager.calls++
	if manager.err != nil {
		return manager.err
	}
	transactionContext := manager.transactionContext
	if transactionContext == nil {
		transactionContext = ctx
	}
	return fn(transactionContext, manager.repositories)
}

type fakeTransactionRepositories struct {
	projects   ports.TransactionProjectRepository
	workspaces ports.TransactionWorkspaceRepository
}

func (repositories fakeTransactionRepositories) Projects() ports.TransactionProjectRepository {
	return repositories.projects
}

func (repositories fakeTransactionRepositories) Workspaces() ports.TransactionWorkspaceRepository {
	return repositories.workspaces
}

type fakeTransactionProjects struct {
	created domain.Project
	ctx     context.Context
	err     error
}

func (repository *fakeTransactionProjects) Create(ctx context.Context, project domain.Project) error {
	repository.ctx = ctx
	repository.created = project
	return repository.err
}

type fakeTransactionWorkspaces struct {
	created domain.Workspace
	ctx     context.Context
	err     error
}

func (repository *fakeTransactionWorkspaces) Create(ctx context.Context, workspace domain.Workspace) error {
	repository.ctx = ctx
	repository.created = workspace
	return repository.err
}

type fakeProjectRepository struct {
	project     domain.Project
	getErr      error
	page        ports.ProjectPage
	listErr     error
	listRequest ports.ProjectPageRequest
	listCalls   int
}

func (repository *fakeProjectRepository) Get(context.Context, domain.ProjectID) (domain.Project, error) {
	return repository.project, repository.getErr
}

func (repository *fakeProjectRepository) List(_ context.Context, request ports.ProjectPageRequest) (ports.ProjectPage, error) {
	repository.listCalls++
	repository.listRequest = request
	return repository.page, repository.listErr
}

func mustProjectID(t *testing.T) domain.ProjectID {
	t.Helper()
	id, err := domain.NewProjectID(projectIDString)
	if err != nil {
		t.Fatalf("NewProjectID() error = %v", err)
	}
	return id
}

func mustProject(t *testing.T) domain.Project {
	t.Helper()
	project, err := domain.NewProject(mustProjectID(t), "Project", fixedTime)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	return project
}
