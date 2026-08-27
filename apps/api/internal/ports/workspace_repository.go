package ports

import (
	"context"

	"github.com/mattwebhub/micro1-go-template/internal/domain"
)

// WorkspaceRepository reads workspaces and performs the single-statement,
// optimistic save. Save must compare expectedVersion atomically and return a
// domain.ErrVersionConflict-compatible error when another writer wins.
type WorkspaceRepository interface {
	GetByProjectID(ctx context.Context, projectID domain.ProjectID) (domain.Workspace, error)
	Save(ctx context.Context, workspace domain.Workspace, expectedVersion uint64) (domain.Workspace, error)
}

// TransactionWorkspaceRepository is used only while creating the initial
// workspace in the same transaction as its project.
type TransactionWorkspaceRepository interface {
	Create(ctx context.Context, workspace domain.Workspace) error
}
