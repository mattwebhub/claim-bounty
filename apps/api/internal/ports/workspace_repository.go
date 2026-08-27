package ports

import (
	"context"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

// WorkspaceReader is the read capability consumed by workspace queries and
// by the save command before it applies the domain transition.
type WorkspaceReader interface {
	GetByProjectID(ctx context.Context, projectID domain.ProjectID) (domain.Workspace, error)
}

// WorkspaceSaver performs the single-statement optimistic write. Save must
// compare expectedVersion atomically and return a
// domain.ErrVersionConflict-compatible error when another writer wins.
type WorkspaceSaver interface {
	Save(ctx context.Context, workspace domain.Workspace, expectedVersion uint64) (domain.Workspace, error)
}

// TransactionWorkspaceRepository is used only while creating the initial
// workspace in the same transaction as its project.
type TransactionWorkspaceRepository interface {
	Create(ctx context.Context, workspace domain.Workspace) error
}
