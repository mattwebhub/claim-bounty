package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattwebhub/micro1-go-template/internal/adapters/postgres"
	"github.com/mattwebhub/micro1-go-template/internal/adapters/system"
	"github.com/mattwebhub/micro1-go-template/internal/config"
	"github.com/mattwebhub/micro1-go-template/internal/services"
	"github.com/mattwebhub/micro1-go-template/internal/transport/httpapi"
)

// buildProjectModule is the only place that chooses concrete implementations
// for the reference feature. Its route consumes service-facing interfaces;
// services consume ports; only this composition root sees all three.
func buildProjectModule(ctx context.Context, cfg config.Config, logger *slog.Logger) (Module, error) {
	if cfg.Database.AutoMigrate {
		migrationContext, cancel := context.WithTimeout(ctx, cfg.Database.MigrationTimeout)
		err := postgres.Migrate(migrationContext, cfg.Database.URL)
		cancel()
		if err != nil {
			return Module{}, fmt.Errorf("bootstrap: migrate database: %w", err)
		}
	}
	startupContext, cancel := context.WithTimeout(ctx, cfg.Database.QueryTimeout)
	defer cancel()
	store, err := postgres.Open(startupContext, cfg.Database.URL, cfg.Database.MaxConnections, cfg.Database.QueryTimeout)
	if err != nil {
		return Module{}, fmt.Errorf("bootstrap: open database: %w", err)
	}
	clock := system.Clock{}
	commands, err := services.NewProjectCommandService(store, system.ProjectIDGenerator{}, clock)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct project commands: %w", err)
	}
	queries, err := services.NewProjectQueryService(store)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct project queries: %w", err)
	}
	workspaces, err := services.NewWorkspaceService(store, clock)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct workspace service: %w", err)
	}
	routes, err := httpapi.NewProjectRoutes(commands, queries, workspaces, logger, cfg.HTTP.MaxBodyBytes)
	if err != nil {
		store.Close()
		return Module{}, fmt.Errorf("bootstrap: construct project routes: %w", err)
	}
	return Module{
		Name:      "projects",
		Routes:    routes,
		Readiness: store.Check,
		Shutdown: func(context.Context) error {
			store.Close()
			return nil
		},
	}, nil
}
