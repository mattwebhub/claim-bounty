package ports

import "context"

// TransactionManager owns infrastructure retries and supplies repositories
// bound to exactly one transaction. Implementations must never smuggle a
// transaction through context values.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(TransactionRepositories) error) error
}

type TransactionRepositories interface {
	Projects() TransactionProjectRepository
	Workspaces() TransactionWorkspaceRepository
}
