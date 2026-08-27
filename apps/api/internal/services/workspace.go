package services

import (
	"context"
	"fmt"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

type GetWorkspaceQuery struct {
	ProjectID domain.ProjectID
}

type GetWorkspaceResult struct {
	Workspace domain.Workspace
}

type SaveWorkspaceCommand struct {
	ProjectID       domain.ProjectID
	ExpectedVersion uint64
	Document        domain.WorkspaceDocument
}

type SaveWorkspaceResult struct {
	Workspace domain.Workspace
}

// WorkspaceQueryService owns workspace reads and has no write capability.
type WorkspaceQueryService struct {
	reader ports.WorkspaceReader
}

func NewWorkspaceQueryService(reader ports.WorkspaceReader) (*WorkspaceQueryService, error) {
	if reader == nil {
		return nil, ErrInvalidDependencies
	}
	return &WorkspaceQueryService{reader: reader}, nil
}

func (s *WorkspaceQueryService) GetWorkspace(
	ctx context.Context,
	query GetWorkspaceQuery,
) (GetWorkspaceResult, error) {
	if _, err := domain.NewProjectID(query.ProjectID.String()); err != nil {
		return GetWorkspaceResult{}, err
	}
	workspace, err := s.reader.GetByProjectID(ctx, query.ProjectID)
	if err != nil {
		return GetWorkspaceResult{}, fmt.Errorf("get workspace: %w", err)
	}
	return GetWorkspaceResult{Workspace: workspace}, nil
}

// WorkspaceCommandService owns workspace mutations. Its reader and saver are
// separate capabilities so a save use case cannot acquire unrelated methods.
type WorkspaceCommandService struct {
	reader ports.WorkspaceReader
	saver  ports.WorkspaceSaver
	clock  ports.Clock
}

func NewWorkspaceCommandService(
	reader ports.WorkspaceReader,
	saver ports.WorkspaceSaver,
	clock ports.Clock,
) (*WorkspaceCommandService, error) {
	if reader == nil || saver == nil || clock == nil {
		return nil, ErrInvalidDependencies
	}
	return &WorkspaceCommandService{reader: reader, saver: saver, clock: clock}, nil
}

func (s *WorkspaceCommandService) SaveWorkspace(
	ctx context.Context,
	command SaveWorkspaceCommand,
) (SaveWorkspaceResult, error) {
	var issues []domain.FieldIssue
	if _, err := domain.NewProjectID(command.ProjectID.String()); err != nil {
		issues = append(issues, domain.FieldIssue{
			Field: "projectId", Code: "invalid_format", Message: "must be a UUID",
		})
	}
	if command.ExpectedVersion == 0 {
		issues = append(issues, domain.FieldIssue{
			Field: "expectedVersion", Code: "required", Message: "must be greater than zero",
		})
	} else if command.ExpectedVersion > domain.MaxWorkspaceVersion {
		issues = append(issues, domain.FieldIssue{
			Field: "expectedVersion", Code: "out_of_range", Message: "must be a JSON-safe integer",
		})
	}
	if err := domain.NewValidationError(issues...); err != nil {
		return SaveWorkspaceResult{}, err
	}

	current, err := s.reader.GetByProjectID(ctx, command.ProjectID)
	if err != nil {
		return SaveWorkspaceResult{}, fmt.Errorf("load workspace for save: %w", err)
	}
	updated, err := current.ReplaceDocument(command.Document, command.ExpectedVersion, s.clock.Now())
	if err != nil {
		return SaveWorkspaceResult{}, fmt.Errorf("apply workspace document: %w", err)
	}

	saved, err := s.saver.Save(ctx, updated, command.ExpectedVersion)
	if err != nil {
		return SaveWorkspaceResult{}, fmt.Errorf("save workspace: %w", err)
	}
	return SaveWorkspaceResult{Workspace: saved}, nil
}
