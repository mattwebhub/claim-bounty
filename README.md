# Micro1 Full-Stack Template

A production-shaped, agent-ready monorepo for a Go API and React + Vite application. It ships one neutral project/workspace vertical slice through the browser, HTTP API, and PostgreSQL so architectural rules are demonstrated by working code rather than empty folders.

## Repository map

```text
apps/api/             Go, net/http, pgx, Goose, and sqlc
apps/web/             React, Vite, TanStack Query, Zustand, and Playwright
contracts/            Canonical OpenAPI contract
architecture/         Machine-readable dependency rules
docs/                 Agent workflow, testing, ADRs, and golden paths
infra/                Local PostgreSQL composition
scripts/              Repository-wide context and release checks
```

## Start

Requirements: Go 1.26.6, Node.js 20.19+ (Node 22 is pinned in `.node-version`), pnpm 9.15.0, and Docker with Compose v2.

```bash
pnpm install --frozen-lockfile
make doctor
make hooks
make dev
```

The web application runs at <http://localhost:5173> and the API at <http://localhost:8080>.

Safe development defaults work without a dotenv loader. To customize them, export values from [`.env.example`](./.env.example) through your shell or process runner. Browser variables prefixed with `VITE_` are public and must never contain secrets. `COMPOSE_PROJECT_NAME` and `POSTGRES_PORT` let concurrent clones avoid sharing local infrastructure.

## Verify

```bash
make check-fast   # deterministic edit-loop feedback
make check        # full unit, race, browser, architecture, contract, and release gates
make check-ci     # adds PostgreSQL, vulnerabilities, containers, and the real system flow
make test-system  # real browser -> API -> PostgreSQL flow
```

Infrastructure tests use isolated Compose projects and host ports (`55431` and `55432` by default), then delete only their own disposable volumes. They do not stop or erase the database started by `make dev`.

Read [ARCHITECTURE.md](./ARCHITECTURE.md) before introducing a boundary, state owner, adapter, or shared abstraction. Agents should begin with `./scripts/agent-context <path>`.

## Contract workflow

[`contracts/openapi.yaml`](./contracts/openapi.yaml) is the only editable HTTP contract. After a contract change, run `make contract-generate`, review the generated TypeScript diff, then run `make contract-check`. Do not edit `apps/web/src/shared/api/generated/schema.d.ts` by hand.

## Template customization

Before building a product from this template, change the Go module path, repository links, application title, image names, package name, and example domain vocabulary. Keep the root contract, scoped `AGENTS.md` files, architecture registry, and verification commands intact until an intentional architecture decision replaces them.

Useful entrypoints are `make help`, [the agent workflow](./docs/agent-workflow.md), [the testing strategy](./docs/testing.md), and [the cross-stack feature golden path](./docs/golden-paths/add-feature.md).
