// Package postgres implements persistence ports with pgx and sqlc-generated
// queries. SQL errors are translated here; neither services nor HTTP know
// PostgreSQL error codes.
package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/postgres/sqlc"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

const defaultQueryTimeout = 5 * time.Second

var (
	_ ports.ProjectRepository              = (*Store)(nil)
	_ ports.WorkspaceReader                = (*Store)(nil)
	_ ports.WorkspaceSaver                 = (*Store)(nil)
	_ ports.TransactionManager             = (*Store)(nil)
	_ ports.TransactionProjectRepository   = (*transactionRepositories)(nil)
	_ ports.TransactionWorkspaceRepository = (*workspaceTransactionRepository)(nil)
)

type Store struct {
	pool         *pgxpool.Pool
	queries      *sqlc.Queries
	queryTimeout time.Duration
	emails       ports.EmailProtector
}

func NewStore(pool *pgxpool.Pool, queryTimeout time.Duration, emails ...ports.EmailProtector) (*Store, error) {
	if pool == nil {
		return nil, errors.New("postgres: pool is required")
	}
	if queryTimeout <= 0 {
		queryTimeout = defaultQueryTimeout
	}
	var protector ports.EmailProtector
	if len(emails) > 0 {
		protector = emails[0]
	}
	return &Store{pool: pool, queries: sqlc.New(pool), queryTimeout: queryTimeout, emails: protector}, nil
}

func Open(ctx context.Context, databaseURL string, maxConnections int32, queryTimeout time.Duration, emails ...ports.EmailProtector) (*Store, error) {
	if databaseURL == "" {
		return nil, errors.New("postgres: database URL is required")
	}
	configuration, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("postgres: database configuration is invalid")
	}
	if maxConnections > 0 {
		configuration.MaxConns = maxConnections
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuration)
	if err != nil {
		return nil, databaseFailure("open pool", err)
	}
	store, err := NewStore(pool, queryTimeout, emails...)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if err := store.Check(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) protectEmail(ctx context.Context, email string) ([]byte, [32]byte, error) {
	if s.emails == nil {
		return nil, [32]byte{}, errors.New("postgres: email protector is required for ClaimBounty")
	}
	return s.emails.EncryptEmail(ctx, email)
}

func (s *Store) revealEmail(ctx context.Context, ciphertext []byte) (string, error) {
	if s.emails == nil {
		return "", errors.New("postgres: email protector is required for ClaimBounty")
	}
	return s.emails.DecryptEmail(ctx, ciphertext)
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		return databaseFailure("health check", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id domain.ProjectID) (domain.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	row, err := s.queries.GetProject(ctx, postgresUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if err != nil {
		return domain.Project{}, databaseFailure("get project", err)
	}
	return projectFromRow(row)
}

func (s *Store) List(ctx context.Context, request ports.ProjectPageRequest) (ports.ProjectPage, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	if request.Limit == 0 || request.Limit > 100 {
		return ports.ProjectPage{}, domain.NewValidationError(domain.FieldIssue{
			Field: "limit", Code: "out_of_range", Message: "must be between 1 and 100",
		})
	}
	hasCursor := request.Cursor != ""
	cursorTime := pgtype.Timestamptz{Time: time.Unix(0, 0).UTC(), Valid: true}
	cursorID := pgtype.UUID{Valid: true}
	if hasCursor {
		cursor, err := decodeCursor(request.Cursor)
		if err != nil {
			return ports.ProjectPage{}, domain.NewValidationError(domain.FieldIssue{
				Field: "cursor", Code: "invalid", Message: "must be a cursor returned by this endpoint",
			})
		}
		cursorTime.Time = cursor.CreatedAt
		cursorID = postgresUUID(cursor.ID)
	}
	rows, err := s.queries.ListProjects(ctx, hasCursor, cursorTime, cursorID, int32(request.Limit+1))
	if err != nil {
		return ports.ProjectPage{}, databaseFailure("list projects", err)
	}
	projects := make([]domain.Project, 0, min(len(rows), int(request.Limit)))
	for index, row := range rows {
		if index == int(request.Limit) {
			break
		}
		project, err := projectFromRow(row)
		if err != nil {
			return ports.ProjectPage{}, err
		}
		projects = append(projects, project)
	}
	page := ports.ProjectPage{Projects: projects}
	if len(rows) > int(request.Limit) && len(projects) > 0 {
		last := projects[len(projects)-1]
		page.NextCursor, err = encodeCursor(projectCursor{CreatedAt: last.CreatedAt(), ID: last.ID()})
		if err != nil {
			return ports.ProjectPage{}, fmt.Errorf("postgres: encode project cursor: %w", err)
		}
	}
	return page, nil
}

func (s *Store) GetByProjectID(ctx context.Context, projectID domain.ProjectID) (domain.Workspace, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	row, err := s.queries.GetWorkspace(ctx, postgresUUID(projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Workspace{}, domain.ErrWorkspaceNotFound
	}
	if err != nil {
		return domain.Workspace{}, databaseFailure("get workspace", err)
	}
	return workspaceFromRow(row)
}

func (s *Store) Save(ctx context.Context, workspace domain.Workspace, expectedVersion uint64) (domain.Workspace, error) {
	if workspace.Version() > domain.MaxWorkspaceVersion || expectedVersion > domain.MaxWorkspaceVersion {
		return domain.Workspace{}, domain.NewValidationError(domain.FieldIssue{Field: "expectedVersion", Code: "out_of_range", Message: "exceeds the persistence version limit"})
	}
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	document, err := marshalDocument(workspace.Document())
	if err != nil {
		return domain.Workspace{}, err
	}
	row, err := s.queries.UpdateWorkspace(ctx, sqlc.UpdateWorkspaceParams{
		ProjectID: postgresUUID(workspace.ProjectID()), Document: document,
		NewVersion: int64(workspace.Version()), UpdatedAt: postgresTime(workspace.UpdatedAt()),
		ExpectedVersion: int64(expectedVersion),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		current, getErr := s.queries.GetWorkspace(ctx, postgresUUID(workspace.ProjectID()))
		if errors.Is(getErr, pgx.ErrNoRows) {
			return domain.Workspace{}, domain.ErrWorkspaceNotFound
		}
		if getErr != nil {
			return domain.Workspace{}, databaseFailure("inspect workspace conflict", getErr)
		}
		return domain.Workspace{}, domain.NewVersionConflictError(expectedVersion, uint64(current.Version))
	}
	if err != nil {
		return domain.Workspace{}, databaseFailure("save workspace", err)
	}
	return workspaceFromRow(row)
}

func (s *Store) WithinTransaction(
	ctx context.Context,
	fn func(context.Context, ports.TransactionRepositories) error,
) error {
	if fn == nil {
		return errors.New("postgres: transaction callback is required")
	}
	transactionCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(transactionCtx, s.pool, func(tx pgx.Tx) error {
		repositories := &transactionRepositories{queries: s.queries.WithTx(tx)}
		if err := fn(transactionCtx, repositories); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return databaseFailure("transaction", err)
	}
	return nil
}

type transactionRepositories struct {
	queries *sqlc.Queries
}

func (r *transactionRepositories) Projects() ports.TransactionProjectRepository { return r }
func (r *transactionRepositories) Workspaces() ports.TransactionWorkspaceRepository {
	return &workspaceTransactionRepository{parent: r}
}

func (r *transactionRepositories) Create(ctx context.Context, project domain.Project) error {
	err := r.queries.CreateProject(ctx, postgresUUID(project.ID()), project.Name(), postgresTime(project.CreatedAt()), postgresTime(project.UpdatedAt()))
	if constraint(err, "projects_pkey") {
		return domain.ErrProjectExists
	}
	if err != nil {
		return databaseFailure("create project", err)
	}
	return nil
}

func (r *transactionRepositories) CreateWorkspace(ctx context.Context, workspace domain.Workspace) error {
	document, err := marshalDocument(workspace.Document())
	if err != nil {
		return err
	}
	err = r.queries.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{
		ProjectID: postgresUUID(workspace.ProjectID()), Document: document, Version: int64(workspace.Version()),
		CreatedAt: postgresTime(workspace.CreatedAt()), UpdatedAt: postgresTime(workspace.UpdatedAt()),
	})
	if constraint(err, "workspaces_project_id_fkey") {
		return domain.ErrProjectNotFound
	}
	if err != nil {
		return databaseFailure("create workspace", err)
	}
	return nil
}

// Go cannot overload Create for the project and workspace interfaces, so this
// narrow value exposes only the transaction-bound workspace capability.
type workspaceTransactionRepository struct{ parent *transactionRepositories }

func (r *workspaceTransactionRepository) Create(ctx context.Context, workspace domain.Workspace) error {
	return r.parent.CreateWorkspace(ctx, workspace)
}

type projectCursor struct {
	CreatedAt time.Time        `json:"createdAt"`
	ID        domain.ProjectID `json:"-"`
	IDString  string           `json:"id,omitempty"`
}

func encodeCursor(cursor projectCursor) (string, error) {
	cursor.IDString = cursor.ID.String()
	value, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func decodeCursor(raw string) (projectCursor, error) {
	value, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return projectCursor{}, err
	}
	var decoded projectCursor
	if err := json.Unmarshal(value, &decoded); err != nil {
		return projectCursor{}, err
	}
	id, err := domain.NewProjectID(decoded.IDString)
	if err != nil || decoded.CreatedAt.IsZero() {
		return projectCursor{}, errors.New("invalid cursor contents")
	}
	decoded.ID = id
	return decoded, nil
}

func projectFromRow(row sqlc.Project) (domain.Project, error) {
	id, err := domain.NewProjectID(uuidString(row.ID))
	if err != nil {
		return domain.Project{}, fmt.Errorf("postgres: restore project ID: %w", err)
	}
	project, err := domain.RestoreProject(id, row.Name, row.CreatedAt.Time, row.UpdatedAt.Time)
	if err != nil {
		return domain.Project{}, fmt.Errorf("postgres: restore project: %w", err)
	}
	return project, nil
}

func postgresUUID(id domain.ProjectID) pgtype.UUID {
	var value pgtype.UUID
	_ = value.Scan(id.String())
	return value
}

func uuidString(value pgtype.UUID) string {
	encoded, _ := value.Value()
	result, _ := encoded.(string)
	return result
}

func postgresTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func constraint(err error, name string) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.ConstraintName == name
}

func checkedVersion(value int64) (uint64, error) {
	if value <= 0 {
		return 0, errors.New("postgres: invalid persisted workspace version")
	}
	return uint64(value), nil
}

// databaseError preserves cancellation and typed error inspection without
// exposing SQL text, SQLSTATE details, topology, or provider messages through
// logs and outer error wrappers.
type databaseError struct {
	operation string
	cause     error
}

func databaseFailure(operation string, cause error) error {
	return &databaseError{operation: operation, cause: cause}
}

func (e *databaseError) Error() string { return "postgres: " + e.operation + " failed" }
func (e *databaseError) Unwrap() error { return e.cause }
