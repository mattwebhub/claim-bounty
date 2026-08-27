.DEFAULT_GOAL := help

GOVULNCHECK_VERSION := v1.7.0
LEFTHOOK_VERSION := v1.13.6
ACTIONLINT_VERSION := v1.7.12
TEST_DATABASE_URL ?= postgres://postgres:postgres@127.0.0.1:5432/app?sslmode=disable
CONTEXT_PATH ?= .

.PHONY: help doctor context arch arch-explain dev db-up db-down validate-config \
	fmt fmt-check vet lint lint-new lint-actions test test-race test-integration migration-check vulnerability \
	build docker-build public-release check-fast precommit check check-ci hooks clean

help: ## Show available commands.
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

doctor: ## Validate the local toolchain and runtime prerequisites.
	@../../scripts/doctor

context: ## Show boundaries and focused checks (CONTEXT_PATH=internal/domain).
	@cd ../.. && ./scripts/agent-context "apps/api/$(CONTEXT_PATH)"

arch: ## Run executable architecture rules and their fixture tests.
	go run ./tools/archlint -root .
	go test ./tools/archlint

arch-explain: ## Explain one rule (RULE=GO-ARCH-001).
	@test -n "$(RULE)" || (echo "usage: make arch-explain RULE=GO-ARCH-001" && exit 2)
	go run ./tools/archlint -explain "$(RULE)"

dev: ## Run PostgreSQL and then the API.
	$(MAKE) -C ../.. db-up
	go run ./cmd/api

db-up: ## Start the pinned local PostgreSQL service and wait for health.
	docker compose -f ../../infra/compose.yaml up -d --wait postgres

db-down: ## Stop and remove the template's ephemeral PostgreSQL service.
	docker compose -f ../../infra/compose.yaml down --volumes --remove-orphans

validate-config: ## Validate environment configuration without serving traffic.
	go run ./cmd/api validate-config

fmt: ## Format Go source.
	gofmt -w .

fmt-check: ## Fail if Go source is not formatted.
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run: make fmt" && exit 1)

vet: ## Run the standard static analyzer.
	go vet ./...

lint: ## Run the pinned high-signal lint suite.
	../../scripts/golangci-lint run
	$(MAKE) lint-actions

lint-new: ## Reject lint findings introduced by the current worktree change.
	../../scripts/golangci-lint run --new-from-rev=HEAD --whole-files

lint-actions: ## Validate GitHub workflow syntax and expressions.
	go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION) ../../.github/workflows/*.yml

test: ## Run the deterministic unit and HTTP test suite.
	go test ./...

test-race: ## Run all non-integration tests with the race detector.
	go test -race ./...

test-integration: ## Run PostgreSQL adapter tests against TEST_DATABASE_URL.
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -tags=integration -count=1 ./internal/adapters/postgres

migration-check: ## Replay migrations up/down/up against the disposable test database.
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -tags=integration -count=1 -run TestMigrationsReplayUpDownUp ./internal/adapters/postgres

vulnerability: ## Scan reachable dependencies with the pinned Go vulnerability database client.
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

build: ## Build the production API binary.
	go build -trimpath -o bin/api ./cmd/api

docker-build: ## Build the pinned, non-root runtime image.
	docker build --pull --file apps/api/Dockerfile --tag micro1-api:local ../..

public-release: ## Scan for secrets, private files, and unsafe release defaults.
	@cd ../.. && ./scripts/check-public-release

check-fast: fmt-check arch vet test ## Run the warm-cache edit-loop gate.

precommit: check-fast lint-new ## Run backend checks required before a commit.

check: check-fast lint test-race build public-release ## Run the complete local handoff gate.

check-ci: ## Run database, security, and container gates in an ephemeral local stack.
	@set -eu; \
	  $(MAKE) db-up; \
	  trap '$(MAKE) db-down' EXIT INT TERM; \
	  $(MAKE) check migration-check test-integration vulnerability docker-build

hooks: ## Install the pinned Lefthook Git hooks.
	@cd ../.. && pnpm hooks:install

clean: ## Remove local build artifacts.
	@set -eu; target="$(CURDIR)/bin"; [ "$${target}" != "$(CURDIR)" ]; rm -rf "$${target}"
