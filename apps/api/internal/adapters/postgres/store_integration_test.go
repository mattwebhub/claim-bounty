//go:build integration

package postgres_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mattwebhub/micro1-template/apps/api/internal/adapters/postgres"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
	"github.com/mattwebhub/micro1-template/apps/api/migrations"
	"github.com/pressly/goose/v3"
)

func TestMigrationsReplayUpDownUp(t *testing.T) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseURL, cleanup := isolatedSchema(t, ctx, baseURL)
	defer cleanup()

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	if _, err := provider.Down(ctx); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("migrate up after rollback: %v", err)
	}
}

func TestStoreRoundTripTransactionPaginationAndConflict(t *testing.T) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	databaseURL, cleanup := isolatedSchema(t, ctx, baseURL)
	defer cleanup()
	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, databaseURL, 4, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	id := mustID(t, "0190cafe-7a4f-7000-8000-000000000001")
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject(id, "Integration", now)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := domain.NewWorkspace(id, domain.EmptyWorkspaceDocument(), now)
	if err != nil {
		t.Fatal(err)
	}
	// Use the exact port callback type: both writes must commit together.
	err = store.WithinTransaction(ctx, func(transactionCtx context.Context, repositories ports.TransactionRepositories) error {
		if err := repositories.Projects().Create(transactionCtx, project); err != nil {
			return err
		}
		return repositories.Workspaces().Create(transactionCtx, workspace)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, id)
	if err != nil || got.Name() != project.Name() {
		t.Fatalf("Get() = %q, %v", got.Name(), err)
	}
	page, err := store.List(ctx, ports.ProjectPageRequest{Limit: 1})
	if err != nil || len(page.Projects) != 1 || page.NextCursor != "" {
		t.Fatalf("single-item List() = %#v, %v", page, err)
	}

	secondID := mustID(t, "0190cafe-7a4f-7000-8000-000000000002")
	secondProject, err := domain.NewProject(secondID, "Newer integration project", now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	secondWorkspace, err := domain.NewWorkspace(secondID, domain.EmptyWorkspaceDocument(), now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WithinTransaction(ctx, func(transactionCtx context.Context, repositories ports.TransactionRepositories) error {
		if err := repositories.Projects().Create(transactionCtx, secondProject); err != nil {
			return err
		}
		return repositories.Workspaces().Create(transactionCtx, secondWorkspace)
	}); err != nil {
		t.Fatal(err)
	}
	firstPage, err := store.List(ctx, ports.ProjectPageRequest{Limit: 1})
	if err != nil || len(firstPage.Projects) != 1 || firstPage.Projects[0].ID() != secondID || firstPage.NextCursor == "" {
		t.Fatalf("first cursor page = %#v, %v", firstPage, err)
	}
	secondPage, err := store.List(ctx, ports.ProjectPageRequest{Limit: 1, Cursor: firstPage.NextCursor})
	if err != nil || len(secondPage.Projects) != 1 || secondPage.Projects[0].ID() != id {
		t.Fatalf("second cursor page = %#v, %v", secondPage, err)
	}

	loaded, err := store.GetByProjectID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := loaded.ReplaceDocument(loaded.Document(), 1, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.Save(ctx, updated, 1)
	if err != nil || saved.Version() != 2 {
		t.Fatalf("Save() version = %d, %v", saved.Version(), err)
	}
	_, err = store.Save(ctx, updated, 1)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale Save() error = %v", err)
	}

	rollbackID := mustID(t, "0190cafe-7a4f-7000-8000-000000000003")
	rollbackProject, err := domain.NewProject(rollbackID, "Rollback", now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	rollbackWorkspace, err := domain.NewWorkspace(rollbackID, domain.EmptyWorkspaceDocument(), now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	err = store.WithinTransaction(ctx, func(transactionCtx context.Context, repositories ports.TransactionRepositories) error {
		if err := repositories.Projects().Create(transactionCtx, rollbackProject); err != nil {
			return err
		}
		if err := repositories.Workspaces().Create(transactionCtx, rollbackWorkspace); err != nil {
			return err
		}
		return repositories.Workspaces().Create(transactionCtx, rollbackWorkspace)
	})
	if err == nil {
		t.Fatal("duplicate workspace unexpectedly committed")
	}
	if _, err := store.Get(ctx, rollbackID); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("partial transaction persisted project: %v", err)
	}
}

func isolatedSchema(t *testing.T, ctx context.Context, baseURL string) (string, func()) {
	t.Helper()
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	schema := "test_" + hex.EncodeToString(random)
	database, err := sql.Open("pgx", baseURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `CREATE SCHEMA "`+schema+`"`); err != nil {
		database.Close()
		t.Fatal(err)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String(), func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = database.ExecContext(cleanupCtx, `DROP SCHEMA "`+schema+`" CASCADE`)
		_ = database.Close()
	}
}

func mustID(t *testing.T, raw string) domain.ProjectID {
	t.Helper()
	id, err := domain.NewProjectID(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
