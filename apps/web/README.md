# Web application

The React + Vite application demonstrates feature-first boundaries, lazy routing, TanStack Query server state, Zustand workspace state, runtime-validated HTTP, accessible components, deterministic browser tests, and performance budgets.

Run focused commands from the repository root:

```bash
pnpm --filter @micro1/web dev
pnpm --filter @micro1/web verify:fast
pnpm --filter @micro1/web verify
```

Build the production container from the root context:

```bash
docker build -f apps/web/Dockerfile -t micro1-web .
```

The editable API contract is `contracts/openapi.yaml` at the repository root. `pnpm contract:generate` updates the generated TypeScript schema. Feature services still validate untrusted responses and map DTOs into feature models.

Read `AGENTS.md` and `ARCHITECTURE.md` before changing feature imports or state ownership.
