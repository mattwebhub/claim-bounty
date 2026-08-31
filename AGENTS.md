# Agent Instructions

## Start here

1. Run `./scripts/agent-context <path>` before editing.
2. Read the nearest `AGENTS.md` and the relevant architecture rule.
3. Use `make check-fast` during the edit loop and `make review` before handoff.
4. Change the root OpenAPI contract and generated frontend types together.

`./scripts/agent-context` prints the stable rules and smallest useful gate for the target. Use `make arch-explain RULE=<ID>` for local remediation without searching external documentation.

## Boundaries

- `apps/api/internal/domain` is pure business logic and imports no adapters, transport, configuration, or infrastructure.
- `apps/api/internal/services` orchestrates ports; it does not know HTTP or PostgreSQL.
- `apps/web/src/features/<feature>` owns that feature's server adapters, schemas, components, and local workflow state.
- Routes compose features; features do not import routes or another feature's internals.
- TanStack Query owns server state. Zustand owns synchronous client workflow state. Forms own transient input state.
- `contracts/openapi.yaml` is the only API contract source of truth.
- Root files own orchestration and policy only. Product behavior belongs in an application; do not hide business logic in scripts, Make targets, or a generic shared package.

## Change discipline

- Prefer the smallest complete vertical change over a speculative abstraction.
- Do not add a shared package until two real consumers require it.
- Add or update tests at the boundary where behavior changes.
- Never suppress a quality gate without documenting a narrow architecture exception.
- Never persist secrets in browser storage or expose secrets through `VITE_` variables.
- Never use the `agent/` branch prefix; use a plain descriptive branch name.
- Do not run destructive commands from a parent workspace. Repository infrastructure commands use explicit Compose projects; preserve that isolation when adding a service.

## Handoff

- Application-only change: run the scoped fast gate, then its full gate.
- Contract or cross-stack change: run `make contract-check` and `make test-system`.
- Root policy or release change: run `make check` and `make public-release`.
- Persistence, dependency, container, or infrastructure change: run the relevant portion of `make check-ci`.
- Always report exact commands, failures, and deliberately skipped infrastructure checks.
- `git push` runs the complete deterministic suite and a read-only Codex review. Fix
  actionable findings rather than suppressing the hook; use `make ai-review` to rerun
  only the semantic review while iterating.
