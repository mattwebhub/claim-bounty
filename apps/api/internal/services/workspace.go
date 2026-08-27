package services

import (
	"context"
	"fmt"

	"github.com/mattwebhub/micro1-go-template/internal/domain"
	"github.com/mattwebhub/micro1-go-template/internal/ports"
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

type WorkspaceService struct {
	repository ports.WorkspaceRepository
	clock      ports.Clock
}

func NewWorkspaceService(repository ports.WorkspaceRepository, clock ports.Clock) (*WorkspaceService, error) {
	if repository == nil || clock == nil {
		return nil, ErrInvalidDependencies
	}
	return &WorkspaceService{repository: repository, clock: clock}, nil
}

func (s *WorkspaceService) GetWorkspace(
	ctx context.Context,
	query GetWorkspaceQuery,
) (GetWorkspaceResult, error) {
	if _, err := domain.NewProjectID(query.ProjectID.String()); err != nil {
		return GetWorkspaceResult{}, err
	}
	workspace, err := s.repository.GetByProjectID(ctx, query.ProjectID)
	if err != nil {
		return GetWorkspaceResult{}, fmt.Errorf("get workspace: %w", err)
	}
	return GetWorkspaceResult{Workspace: workspace}, nil
}

func (s *WorkspaceService) SaveWorkspace(
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

	current, err := s.repository.GetByProjectID(ctx, command.ProjectID)
	if err != nil {
		return SaveWorkspaceResult{}, fmt.Errorf("load workspace for save: %w", err)
	}
	updated, err := current.ReplaceDocument(command.Document, command.ExpectedVersion, s.clock.Now())
	if err != nil {
		return SaveWorkspaceResult{}, fmt.Errorf("apply workspace document: %w", err)
	}

	saved, err := s.repository.Save(ctx, updated, command.ExpectedVersion)
	if err != nil {
		return SaveWorkspaceResult{}, fmt.Errorf("save workspace: %w", err)
	}
	return SaveWorkspaceResult{Workspace: saved}, nil
}
