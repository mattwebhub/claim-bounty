# Agent guide

Before editing, read `ARCHITECTURE.md`, inspect the nearest tests, and state the boundary that owns the change. Run `make context CONTEXT_PATH=path/to/change` for a concise local map. Prefer a complete vertical slice over a new generic abstraction.

## Hard rules

- Never use the `agent/` Git branch prefix. Use a plain descriptive name.
- `cmd/api` imports only `internal/bootstrap` plus the standard library.
- Domain imports the standard library only and never knows JSON, HTTP, SQL, logging, environment, clocks, or random acquisition.
- Services expose named commands/queries, depend on domain-language ports, and never return HTTP statuses, SQL rows, or provider DTOs.
- Handlers own protocol decoding and define their minimal inbound interfaces. They never import a concrete adapter.
- Adapters translate I/O errors before they cross inward. Never leak SQL, provider messages, credentials, or connection strings.
- Dependencies are explicit constructor arguments. Do not add service locators, global mutable dependencies, `With...` mutation chains, or dependencies in context values.
- Use request contexts for all I/O. Do not create `context.Background()` in a request path.
- Decode JSON with `response.DecodeJSON`; do not add unbounded decoder or `io.ReadAll` calls.
- Use stable error codes and the shared response envelope. Unknown errors are safe externally and logged once.
- Register routes explicitly. Register dependency readiness and cleanup through `bootstrap.Module`.
- Do not create `utils`, `helpers`, `common`, `manager`, or generic repository packages. Keep private helpers beside their caller until two consumers establish a stable concept.
- Files over roughly 300 lines require a cohesion review; 500 lines fail unless generated or explicitly documented.
- Never edit generated artifacts manually or bypass a failing architecture/test check with an unexplained ignore.
- Architecture failures use stable IDs. Run `make arch-explain RULE=<ID>` before changing code or requesting an exception.

## Verification

Run `gofmt` on touched Go files, focused package tests during development, then `make check` before handoff. Run `make check-ci` for persistence, migration, infrastructure, or release changes. Report exact commands and any skipped checks. When adding a resource, test domain invariants, service behavior, handler mapping, adapter behavior against real infrastructure, and one compiled-process path in proportion to the change.

## Review questions

Identify the concrete use case, sole owner of each invariant, direction of every dependency, transaction boundary, input/output/error translator, cleanup path, and executable regression check. If an interface merely mirrors a concrete type or a service merely forwards to one port, challenge it before adding ceremony.
