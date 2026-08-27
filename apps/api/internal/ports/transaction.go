package ports

import "context"

// TransactionManager owns infrastructure retries and supplies both a bounded
// transaction context and repositories bound to exactly one transaction. The
// callback must use transactionCtx for every operation on those repositories.
// Implementations must never smuggle a transaction through context values.
type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(transactionCtx context.Context, repositories TransactionRepositories) error,
	) error
}

type TransactionRepositories interface {
	Projects() TransactionProjectRepository
	Workspaces() TransactionWorkspaceRepository
}
