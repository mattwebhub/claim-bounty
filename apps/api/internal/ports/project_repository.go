package ports

import (
	"context"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

// ProjectRepository is the non-transactional read capability consumed by
// project queries. Creation is intentionally available only through
// TransactionProjectRepository.
type ProjectRepository interface {
	Get(ctx context.Context, id domain.ProjectID) (domain.Project, error)
	List(ctx context.Context, page ProjectPageRequest) (ProjectPage, error)
}

// ProjectPageRequest is adapter-neutral. Cursor contents remain opaque outside
// the persistence adapter that issued them.
type ProjectPageRequest struct {
	Limit  uint32
	Cursor string
}

type ProjectPage struct {
	Projects   []domain.Project
	NextCursor string
}

// TransactionProjectRepository is a transaction-bound write capability. Its
// deliberately restricted method set makes transaction participation visible
// at the service call site.
type TransactionProjectRepository interface {
	Create(ctx context.Context, project domain.Project) error
}
