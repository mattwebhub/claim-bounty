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
SYSTEM_API_PORT ?= 58080
SYSTEM_WEB_PORT ?= 58081
SYSTEM_MAILPIT_PORT ?= 58025
COMPOSE := docker compose --project-name $(COMPOSE_PROJECT_NAME) --file infra/compose.yaml

.PHONY: help install doctor dev db-up db-down contract-generate contract-check \
	architecture-registry arch arch-explain api-check-fast api-precommit api-check web-check-fast web-check check-fast check \
	review-hook-test reviewer-guide-check test-integration test-system vulnerability docker-build verify-export public-release check-ci review review-push ai-review hooks clean

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

review-hook-test: ## Verify exact pushed-range failures cannot fall through to another review.
	./scripts/test-review-hook

reviewer-guide-check: ## Check the 13-page reviewer guide with the Playwright-pinned Chromium browser.
	node submission/reviewer/guide/check.mjs

check-fast: contract-check architecture-registry api-check-fast web-check-fast review-hook-test ## Run the warm-cache monorepo edit-loop gate.

check: contract-check architecture-registry api-check web-check ## Run all deterministic local release gates.
	GOCACHE="$${GOCACHE:-$$(go env GOCACHE)}" go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)
	./scripts/check-public-release

test-integration: ## Replay migrations and run PostgreSQL tests in an isolated disposable stack.
	@set -eu; \
	  compose='docker compose --project-name $(INTEGRATION_COMPOSE_PROJECT_NAME) --file infra/compose.yaml'; \
	  POSTGRES_PORT='$(INTEGRATION_POSTGRES_PORT)' $$compose up -d --wait postgres; \
	  trap 'POSTGRES_PORT="$(INTEGRATION_POSTGRES_PORT)" $$compose down --volumes --remove-orphans' EXIT INT TERM; \
	  $(MAKE) -C $(API_DIR) TEST_DATABASE_URL='$(TEST_DATABASE_URL)' migration-check test-integration

test-system: ## Exercise the browser against the complete isolated ClaimBounty profile.
	@set -eu; \
	  compose='docker compose --project-name $(SYSTEM_COMPOSE_PROJECT_NAME) --file infra/compose.yaml'; \
	  trap 'POSTGRES_PORT="$(SYSTEM_POSTGRES_PORT)" CLAIMBOUNTY_API_PORT="$(SYSTEM_API_PORT)" CLAIMBOUNTY_WEB_PORT="$(SYSTEM_WEB_PORT)" MAILPIT_UI_PORT="$(SYSTEM_MAILPIT_PORT)" $$compose --profile claimbounty down --volumes --remove-orphans' EXIT INT TERM; \
	  POSTGRES_PORT='$(SYSTEM_POSTGRES_PORT)' CLAIMBOUNTY_API_PORT='$(SYSTEM_API_PORT)' CLAIMBOUNTY_WEB_PORT='$(SYSTEM_WEB_PORT)' MAILPIT_UI_PORT='$(SYSTEM_MAILPIT_PORT)' $$compose --profile claimbounty up --build -d --wait --wait-timeout 300; \
	  SYSTEM_TEST_WEB_ORIGIN='http://127.0.0.1:$(SYSTEM_WEB_PORT)' SYSTEM_TEST_MAILPIT_ORIGIN='http://127.0.0.1:$(SYSTEM_MAILPIT_PORT)' SYSTEM_TEST_COMPOSE_PROJECT_NAME='$(SYSTEM_COMPOSE_PROJECT_NAME)' SYSTEM_TEST_VERIFY_EXPORT='1' SYSTEM_TEST_REQUIRE_CLAIMBOUNTY='1' pnpm test:system

vulnerability: ## Scan Go and JavaScript dependency graphs for known vulnerabilities.
	$(MAKE) -C $(API_DIR) vulnerability
	pnpm audit --audit-level high

docker-build: ## Build both production images from the monorepo root context.
	docker build --file apps/api/Dockerfile --tag micro1-api:local .
	docker build --file apps/web/Dockerfile --tag micro1-web:local .

verify-export: ## Verify exports/claimbounty-export.zip in an offline container.
	@test -f exports/claimbounty-export.zip || (echo "missing exports/claimbounty-export.zip" >&2 && exit 2)
	@printf '%s' '$(EXPORT_SHA256)' | grep -Eq '^[a-f0-9]{64}$$' || (echo "EXPORT_SHA256 must be the export resource's 64-character lowercase hexadecimal sha256" >&2 && exit 2)
	@test ! -e verified-exports/claimbounty-export || (echo "verified-exports/claimbounty-export already exists; choose a new destination or move the previous result" >&2 && exit 2)
	@install -d -m 0700 verified-exports
	EXPORT_SHA256='$(EXPORT_SHA256)' $(COMPOSE) --profile operator run --rm verify-export

public-release: contract-check architecture-registry reviewer-guide-check ## Reject secrets, private references, generated drift, and unsafe files.
	pnpm policies:check
	./scripts/check-public-release

check-ci: check test-integration vulnerability docker-build test-system ## Reproduce the complete infrastructure and security gate.

review: ## Run the complete deterministic and Codex semantic review locally.
	./scripts/review

review-push: ## Review committed changes about to be pushed.
	./scripts/review --push

ai-review: ## Run only the Codex semantic review for current uncommitted changes.
	./scripts/ai-review --uncommitted

hooks: ## Install the single repository-root hook configuration.
	pnpm hooks:install

clean: ## Remove app build outputs through their scoped clean commands.
	$(MAKE) -C $(API_DIR) clean
	pnpm --filter $(WEB_FILTER) exec node -e "for (const path of ['dist','coverage','playwright-report','test-results']) require('node:fs').rmSync(path,{recursive:true,force:true})"
