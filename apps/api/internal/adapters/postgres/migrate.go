package postgres

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mattwebhub/micro1-go-template/migrations"
	"github.com/pressly/goose/v3"
)

// Migrate applies the embedded, monotonically versioned migrations. It is an
// explicit startup action so deployments may choose a separate migration job.
func Migrate(ctx context.Context, databaseURL string) error {
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return databaseFailure("open migration connection", err)
	}
	defer database.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.Files)
	if err != nil {
		return databaseFailure("initialize migrations", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return databaseFailure("apply migrations", err)
	}
	return nil
}
