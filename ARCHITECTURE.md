# Frontend Architecture

## Dependency direction

```text
app/routes -> features -> shared
app/routes ------------> shared
```

- `app` owns bootstrap and narrow global providers.
- `routes` parse URL input and compose feature views.
- `features` own vertical product capabilities.
- `shared` owns business-neutral infrastructure and UI; it never imports app, routes, or features.
- Another feature is imported through `features/<name>/index.ts`, never an internal path.

Run `pnpm architecture:check` after changing imports.

## State ownership

| State                                                           | Sole owner                          |
| --------------------------------------------------------------- | ----------------------------------- |
| Backend resources and session profile                           | TanStack React Query                |
| Shareable resource ID, filters, tabs, selection                 | Router path/search params           |
| Local workflow, editor selection, tools, undo/redo, dirty draft | Feature-scoped Zustand store        |
| Validated form values/errors                                    | React Hook Form + Zod               |
| Private visual state                                            | Component state/reducer             |
| Durable UI preference                                           | Explicit, minimal persisted adapter |
| API collections or credentials                                  | Never Zustand/local storage         |

A Zustand store is synchronous and network-free. A feature controller may initialize a local draft from one query snapshot and persist it through a React Query mutation; it must not turn the store into a second server cache.

## Request pipeline

```text
route/component
  -> feature query/mutation hook
    -> feature query options and key factory
      -> framework-free feature service
        -> shared API/generated client
          -> HTTP/OpenAPI contract
```

- Raw `fetch` exists only in `src/shared/api`.
- Feature services select endpoints, validate/map DTOs, and have no React imports.
- Query modules own keys, cancellation, retries, invalidation, and optimistic rollback.
- Components decide toasts, navigation, and dialogs.
- The shared client normalizes transport failures into `ApiError`, retaining safe messages and request IDs.

## Routes and errors

Routes are lazy by default. Route modules export `Component`; route error boundaries retain structured request identifiers when available. Expensive features add a closer error boundary and loading surface without expanding the global provider tree.

## UI and accessibility

- Product UI uses semantic variables from `src/shared/ui/styles.css`, not raw palette values.
- Shared controls use native semantics or Radix primitives, visible focus, minimum 44px primary targets, and correct disabled/invalid state.
- Drag/drop always has a button and keyboard equivalent.
- A visual workspace must expose a synchronized semantic object list and inspector.
- Pending/saved messages use polite live regions; failures use alerts.
- Reduced-motion preferences are respected and scrollbars remain discoverable.

## Testing

Tests are colocated and discovered by `src/**/*.{test,spec}.{ts,tsx}`. Use `renderApplication` for isolated Query, Router, i18n, and theme providers. MSW owns network behavior. Reset handlers, local storage, Query clients, and feature stores between tests.

Test behavior through roles and user events. Avoid broad snapshots and implementation assertions.

## Generated code

`src/shared/api/generated/` is generator-owned. Generated files receive narrow lint/architecture exclusions; no other code does. Contract and generated output must change together.
