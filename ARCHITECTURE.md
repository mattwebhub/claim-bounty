# Architecture

The monorepo has two deployable applications and one shared contract. It deliberately has no generic shared-code package.

```text
browser -> apps/web -------- HTTP --------> apps/api -> PostgreSQL
             ^                                  ^
             +---- generated DTOs   handlers ---+
                        ^
                contracts/openapi.yaml
```

The contract is a build-time authority, not a runtime intermediary. It drives generated web DTOs and reviewable handler expectations while both deployables remain independently buildable.

## API dependency direction

`domain <- ports <- services <- transport/adapters <- bootstrap <- cmd`

Domain types enforce invariants. Services coordinate use cases through narrow ports. PostgreSQL and HTTP translate at the edges. Bootstrap is the only composition root. Executable rules live in `apps/api/tools/archlint` and `architecture/rules.yaml`.

## Web dependency direction

`shared <- features <- routes <- app`

Feature internals are private; cross-feature imports use public `index.ts` files. Routes compose features but own no server adapters. Shared code cannot import features, routes, or app composition. `dependency-cruiser.cjs` enforces these directions.

## State ownership

- URL: navigation and shareable selection.
- TanStack Query: remote data, cache invalidation, and request lifecycle.
- Zustand: local workspace interaction, history, and save workflow.
- React Hook Form: transient form state and validation presentation.
- Component state: ephemeral visual state with no cross-component owner.

Duplicating the same authoritative value across owners is an architecture defect.

## Contract boundary

`contracts/openapi.yaml` is canonical. The web application generates compile-time DTOs from it and validates untrusted responses with Zod. Go handlers remain explicit and are verified with transport tests plus the real full-stack system test.

The default browser client uses `credentials: same-origin`. Vite and the production web server proxy `/api` to the API, so local development and deployments share the same browser origin without exposing an API credential policy to the bundle. A future cross-origin cookie session requires a separate threat model plus coordinated credentialed-CORS policy; it is not an implicit template default.

Architecture exceptions require a dated ADR with scope, owner, expiry condition, and an executable regression check whenever practical.

## Repository control plane

The root Makefile composes existing app gates; it does not reimplement them. The pnpm workspace owns one JavaScript lockfile, while `apps/api` remains one independent Go module without `go.work`. Root scripts may inspect repository structure, contract drift, and release hygiene but never contain product decisions.

Local infrastructure uses explicit Compose project names. Development state and disposable integration/system-test state are separate ownership domains, so a test cleanup cannot delete a developer's active database.
