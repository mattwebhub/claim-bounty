# React Frontend Template

A business-neutral React + Vite foundation for building production-shaped applications quickly. It provides strict TypeScript, lazy routing, TanStack Query, Zustand, runtime-validated HTTP, accessible UI primitives, semantic tokens, internationalization infrastructure, and behavior-focused tests.

## Requirements

- Node.js 20.19 or newer
- pnpm 9.15

## Start

```bash
cp .env.example .env.local
pnpm install --frozen-lockfile
pnpm hooks:install
pnpm dev
```

Open <http://localhost:5173>. The API defaults to <http://localhost:8080/api/v1>.

## Verify

```bash
pnpm verify:fast
pnpm verify
```

`verify:fast` runs formatting, linting, strict type checking, dependency rules, and colocated unit/component tests. `verify` adds unused-code analysis, the production build, compressed bundle budgets, and Chromium system/accessibility smoke tests. Install the local browser once with `pnpm e2e:install`. The legacy `check:*` commands remain aliases.

Bundle budgets default to 250 KiB total Brotli JavaScript, 175 KiB for the largest JavaScript chunk, and 50 KiB total Brotli CSS. Intentional changes must be reviewed; CI overrides are available through `BUNDLE_MAX_JS_KIB`, `BUNDLE_MAX_CHUNK_KIB`, and `BUNDLE_MAX_CSS_KIB`.

Coverage starts from an executable baseline of 70% lines, functions, and statements plus 55% branches. These thresholds are ratchet-only: raise them as coverage improves, never lower them to merge a change. New and changed behavior still requires focused tests regardless of the global percentage.

## Add a feature

Create `src/features/<feature>/` with:

```text
api/          framework-free service and React Query ownership
components/   feature presentation
model/        schemas, local workflow stores, reducers, selectors
index.ts      the only public cross-feature entrypoint
```

Compose it from a lazy module under `src/routes/`. Read [ARCHITECTURE.md](./ARCHITECTURE.md) before choosing a state owner or adding transport code.

## Environment

Only public, non-secret browser configuration may use the `VITE_` prefix. Values are parsed at startup by `src/shared/config/env.ts`; an invalid URL or timeout fails fast.

## Browser and deployment

Playwright starts the production preview automatically. Set `E2E_BASE_URL` to smoke-test an already deployed application. API requests in the system tests are intercepted at the public `/api/v1` boundary, so browser checks remain deterministic and require no seeded backend.

Build the static production container with:

```bash
docker build \
  --build-arg VITE_API_BASE_URL=https://api.example.com/api/v1 \
  -t react-frontend-template .
docker run --rm -p 8080:8080 react-frontend-template
```

The Caddy runtime supports SPA deep links, immutable hashed assets, compression, security headers, and `/health/live`. Add an environment-specific Content Security Policy at the deployment edge because allowed API origins vary by consumer.

## Monitoring

Core Web Vitals are collected in production through a vendor-neutral sink. The default adapter is deliberately a no-op and records no URLs, identifiers, or user data. Provide a reviewed sink at composition time when integrating telemetry; keep metric dimensions bounded.

## API contract

`contracts/openapi.yaml` is the reviewed API snapshot. `pnpm api:generate` creates the compile-time types in `src/shared/api/generated/`; generated files must never be edited manually. `pnpm api:check` regenerates them and fails on drift. Feature services bind their runtime Zod schemas and request DTOs to these generated types, then map validated DTOs into feature models.

When the backend contract changes, update the snapshot and generated types together, then run the complete verification gate.

## Scope

Authentication and 3D rendering are deliberately absent from the base. They require explicit security or performance decisions rather than insecure placeholders.
