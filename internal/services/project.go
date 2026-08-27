package services

import (
	"context"
	"fmt"

	"github.com/mattwebhub/micro1-go-template/internal/domain"
	"github.com/mattwebhub/micro1-go-template/internal/ports"
)

const (
	defaultProjectPageSize uint32 = 20
	maxProjectPageSize     uint32 = 100
	maxCursorLength               = 1024
)

type CreateProjectCommand struct {
	Name string
}

type CreateProjectResult struct {
	Project   domain.Project
	Workspace domain.Workspace
}

// ProjectCommandService owns the one project mutation. It intentionally does
// not depend on the non-transactional project repository: project and initial
// workspace creation are one atomic invariant.
type ProjectCommandService struct {
	transactions ports.TransactionManager
	ids          ports.ProjectIDGenerator
	clock        ports.Clock
}

func NewProjectCommandService(
	transactions ports.TransactionManager,
	ids ports.ProjectIDGenerator,
	clock ports.Clock,
) (*ProjectCommandService, error) {
	if transactions == nil || ids == nil || clock == nil {
		return nil, ErrInvalidDependencies
	}

	return &ProjectCommandService{transactions: transactions, ids: ids, clock: clock}, nil
}

func (s *ProjectCommandService) CreateProject(
	ctx context.Context,
	command CreateProjectCommand,
) (CreateProjectResult, error) {
	id, err := s.ids.NewProjectID(ctx)
	if err != nil {
		return CreateProjectResult{}, fmt.Errorf("generate project ID: %w", err)
	}

	now := s.clock.Now()
	project, err := domain.NewProject(id, command.Name, now)
	if err != nil {
		return CreateProjectResult{}, fmt.Errorf("create project entity: %w", err)
	}
	workspace, err := domain.NewWorkspace(id, domain.EmptyWorkspaceDocument(), now)
	if err != nil {
		return CreateProjectResult{}, fmt.Errorf("create initial workspace: %w", err)
	}

	err = s.transactions.WithinTransaction(ctx, func(repositories ports.TransactionRepositories) error {
		if repositories == nil {
			return ErrInvalidDependencies
		}
		projects := repositories.Projects()
		workspaces := repositories.Workspaces()
		if projects == nil || workspaces == nil {
			return ErrInvalidDependencies
		}
		if err := projects.Create(ctx, project); err != nil {
			return fmt.Errorf("persist project: %w", err)
		}
		if err := workspaces.Create(ctx, workspace); err != nil {
			return fmt.Errorf("persist initial workspace: %w", err)
		}
		return nil
	})
	if err != nil {
		return CreateProjectResult{}, fmt.Errorf("create project transaction: %w", err)
	}

	return CreateProjectResult{Project: project, Workspace: workspace}, nil
}

type GetProjectQuery struct {
	ProjectID domain.ProjectID
}

type GetProjectResult struct {
	Project domain.Project
}

type ListProjectsQuery struct {
	Limit  uint32
	Cursor string
}

type ListProjectsResult struct {
	Projects   []domain.Project
	NextCursor string
}

// ProjectQueryService is kept separate from commands because its only
// dependency and consistency guarantees are read-oriented.
type ProjectQueryService struct {
	repository ports.ProjectRepository
}

func NewProjectQueryService(repository ports.ProjectRepository) (*ProjectQueryService, error) {
	if repository == nil {
		return nil, ErrInvalidDependencies
	}
	return &ProjectQueryService{repository: repository}, nil
}

func (s *ProjectQueryService) GetProject(
	ctx context.Context,
	query GetProjectQuery,
) (GetProjectResult, error) {
	if _, err := domain.NewProjectID(query.ProjectID.String()); err != nil {
		return GetProjectResult{}, err
	}
	project, err := s.repository.Get(ctx, query.ProjectID)
	if err != nil {
		return GetProjectResult{}, fmt.Errorf("get project: %w", err)
	}
	return GetProjectResult{Project: project}, nil
}

func (s *ProjectQueryService) ListProjects(
	ctx context.Context,
	query ListProjectsQuery,
) (ListProjectsResult, error) {
	limit := query.Limit
	if limit == 0 {
		limit = defaultProjectPageSize
	}

	cursor := query.Cursor
	var issues []domain.FieldIssue
	if limit > maxProjectPageSize {
		issues = append(issues, domain.FieldIssue{
			Field: "limit", Code: "out_of_range", Message: "must be between 1 and 100",
		})
	}
	if len(cursor) > maxCursorLength {
		issues = append(issues, domain.FieldIssue{
			Field: "cursor", Code: "too_long", Message: "must be at most 1024 bytes",
		})
	}
	if err := domain.NewValidationError(issues...); err != nil {
		return ListProjectsResult{}, err
	}

	page, err := s.repository.List(ctx, ports.ProjectPageRequest{Limit: limit, Cursor: cursor})
	if err != nil {
		return ListProjectsResult{}, fmt.Errorf("list projects: %w", err)
	}
	return ListProjectsResult{
		Projects:   append([]domain.Project(nil), page.Projects...),
		NextCursor: page.NextCursor,
	}, nil
}
