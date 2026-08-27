# Agent Instructions

## Before editing

1. Read `ARCHITECTURE.md`.
2. Identify the sole owner for every state value.
3. Search for an existing shared primitive or feature public entrypoint before creating one.
4. Keep changes within the requested ownership boundary; concurrent edits belong to other agents.

## Dependency rules

- Allowed direction: `app/routes -> features -> shared`.
- `shared` never imports app, routes, or features.
- Import another feature only through `features/<name>/index.ts`.
- Raw network access belongs only in `shared/api`.
- Components do not call services or generated clients when React Query should own the lifecycle.
- Zustand stores contain no API resource collections, query clients, router objects, credentials, or network calls.
- Generated files under `src/shared/api/generated/` are never edited manually.

## Code rules

- No explicit `any`, ignored type errors, floating promises, or unexplained lint disables.
- Parse public environment values once and fail with actionable errors.
- Runtime-validate data at external trust boundaries.
- Use semantic tokens; preserve keyboard behavior, visible focus, labels, and live-region contracts.
- Keep providers narrow and route-heavy code lazy.
- Prefer cohesive files; investigate splitting near 300 lines and require a concrete reason beyond 500.

## Tests and handoff

- Add or update the nearest colocated behavior test.
- Run `pnpm check:fast` for normal changes and `pnpm check` before handoff when practical.
- Report commands run, failures, and anything deliberately skipped.
- Do not commit unless the user explicitly requests it.

## Git branches

- Never use the `agent/` prefix.
- Use a plain descriptive name such as `frontend-template-foundation`.
