.DEFAULT_GOAL := help

API_DIR := apps/api
WEB_FILTER := @micro1/web
ACTIONLINT_VERSION := v1.7.12
COMPOSE_PROJECT_NAME ?= micro1-template
POSTGRES_PORT ?= 5432
DATABASE_URL ?= postgres://postgres:postgres@127.0.0.1:$(POSTGRES_PORT)/app?sslmode=disable
INTEGRATION_COMPOSE_PROJECT_NAME ?= $(COMPOSE_PROJECT_NAME)-integration
INTEGRATION_POSTGRES_PORT ?= 55431
TEST_DATABASE_URL ?= postgres://postgres:postgres@127.0.0.1:$(INTEGRATION_POSTGRES_PORT)/app?sslmode=disable
SYSTEM_COMPOSE_PROJECT_NAME ?= $(COMPOSE_PROJECT_NAME)-system
SYSTEM_POSTGRES_PORT ?= 55432
SYSTEM_TEST_DATABASE_URL ?= postgres://postgres:postgres@127.0.0.1:$(SYSTEM_POSTGRES_PORT)/app?sslmode=disable
COMPOSE := docker compose --project-name $(COMPOSE_PROJECT_NAME) --file infra/compose.yaml

.PHONY: help install doctor dev db-up db-down contract-generate contract-check \
	architecture-registry arch arch-explain api-check-fast api-precommit api-check web-check-fast web-check check-fast check \
	test-integration test-system vulnerability docker-build public-release check-ci hooks clean

help: ## Show the monorepo command contract.
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install the pinned frontend workspace dependencies.
	pnpm install --frozen-lockfile

doctor: ## Validate all local toolchain and repository prerequisites.
	@./scripts/doctor

dev: db-up ## Start PostgreSQL, the API, and the Vite application.
	DATABASE_URL='$(DATABASE_URL)' pnpm dev

db-up: ## Start the pinned PostgreSQL service and wait for readiness.
	POSTGRES_PORT='$(POSTGRES_PORT)' $(COMPOSE) up -d --wait postgres

db-down: ## Remove only this monorepo's disposable PostgreSQL service and volume.
	$(COMPOSE) down --volumes --remove-orphans

contract-generate: ## Regenerate the frontend types from the canonical OpenAPI contract.
	pnpm contract:generate

contract-check: ## Fail when generated API types drift from the root contract.
	pnpm contract:check

architecture-registry: ## Validate stable rule metadata and local documentation links.
	./scripts/check-architecture

arch: architecture-registry ## Validate the registry and both executable dependency graphs.
	$(MAKE) -C $(API_DIR) arch
	pnpm --filter $(WEB_FILTER) architecture:check

arch-explain: ## Print one stable architecture rule (RULE=GO-ARCH-001).
	@test -n "$(RULE)" || (echo "usage: make arch-explain RULE=GO-ARCH-001" && exit 2)
	@./scripts/explain-rule "$(RULE)"

api-check-fast: ## Run fast deterministic Go architecture, vet, and unit gates.
	$(MAKE) -C $(API_DIR) check-fast

api-precommit: ## Run backend architecture, tests, and lint for newly changed code.
	$(MAKE) -C $(API_DIR) precommit

api-check: ## Run the complete Go handoff gate.
	$(MAKE) -C $(API_DIR) check

web-check-fast: ## Run fast frontend formatting, lint, type, boundary, contract, and unit gates.
	pnpm --filter $(WEB_FILTER) verify:fast

web-check: ## Run the complete frontend handoff gate, including browser smoke tests.
	pnpm --filter $(WEB_FILTER) verify

check-fast: contract-check architecture-registry api-check-fast web-check-fast ## Run the warm-cache monorepo edit-loop gate.

check: contract-check architecture-registry api-check web-check ## Run all deterministic local release gates.
	GOCACHE="$${GOCACHE:-$$(go env GOCACHE)}" go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)
	./scripts/check-public-release

test-integration: ## Replay migrations and run PostgreSQL tests in an isolated disposable stack.
	@set -eu; \
	  compose='docker compose --project-name $(INTEGRATION_COMPOSE_PROJECT_NAME) --file infra/compose.yaml'; \
	  POSTGRES_PORT='$(INTEGRATION_POSTGRES_PORT)' $$compose up -d --wait postgres; \
	  trap 'POSTGRES_PORT="$(INTEGRATION_POSTGRES_PORT)" $$compose down --volumes --remove-orphans' EXIT INT TERM; \
	  $(MAKE) -C $(API_DIR) TEST_DATABASE_URL='$(TEST_DATABASE_URL)' migration-check test-integration

test-system: ## Exercise the real browser-to-API-to-PostgreSQL slice in an isolated stack.
	@set -eu; \
	  compose='docker compose --project-name $(SYSTEM_COMPOSE_PROJECT_NAME) --file infra/compose.yaml'; \
	  POSTGRES_PORT='$(SYSTEM_POSTGRES_PORT)' $$compose up -d --wait postgres; \
	  trap 'POSTGRES_PORT="$(SYSTEM_POSTGRES_PORT)" $$compose down --volumes --remove-orphans' EXIT INT TERM; \
	  SYSTEM_TEST_DATABASE_URL='$(SYSTEM_TEST_DATABASE_URL)' pnpm test:system

vulnerability: ## Scan Go and JavaScript dependency graphs for known vulnerabilities.
	$(MAKE) -C $(API_DIR) vulnerability
	pnpm audit --audit-level high

docker-build: ## Build both production images from the monorepo root context.
	docker build --file apps/api/Dockerfile --tag micro1-api:local .
	docker build --file apps/web/Dockerfile --tag micro1-web:local .

public-release: contract-check architecture-registry ## Reject secrets, private references, generated drift, and unsafe files.
	./scripts/check-public-release

check-ci: check test-integration vulnerability docker-build test-system ## Reproduce the complete infrastructure and security gate.

hooks: ## Install the single repository-root hook configuration.
	pnpm hooks:install

clean: ## Remove app build outputs through their scoped clean commands.
	$(MAKE) -C $(API_DIR) clean
	pnpm --filter $(WEB_FILTER) exec node -e "for (const path of ['dist','coverage','playwright-report','test-results']) require('node:fs').rmSync(path,{recursive:true,force:true})"
