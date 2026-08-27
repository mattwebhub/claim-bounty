# Architecture

The API uses a globally layered hexagonal architecture. Dependencies point inward; only bootstrap sees all concrete implementations.

```text
cmd/api -> bootstrap -> transport -> services -> ports -> domain
                    \-> adapters -----------^       ^
                    \-> observability
```

Allowed imports:

| Package | May import |
|---|---|
| `domain` | Standard library only |
| `ports` | `domain` |
| `services` | `domain`, `ports` |
| `adapters` | `domain`, `ports`, infrastructure libraries |
| `transport/httpapi` | `domain`, consumer-owned service interfaces, transport helpers |
| `observability` | Standard library and telemetry libraries; no business packages |
| `bootstrap` | Every layer for construction and lifecycle only |
| `cmd/api` | `bootstrap` only |

## Boundary ownership

Domain owns invariants and typed errors. Ports describe outbound capabilities in domain language. Services expose named user/system intents and own application policy and multi-write transaction boundaries. Adapters translate external I/O and errors. HTTP transport owns protocol syntax, DTO mapping, body limits, and response rendering. Bootstrap constructs complete modules, publishes readiness only after startup, and drains HTTP before modules in reverse dependency order.

Handlers define the smallest inbound interface they consume. Services receive explicit command/query values rather than HTTP or generated DTOs. Interfaces are introduced at their consumer boundary and never as mirrors of concrete structs. Dependencies are constructor-injected and immutable; context carries cancellation and technical request metadata, never dependencies or business inputs.

Commands and queries are separate service types when their authority differs. A workspace query receives only `WorkspaceReader`; a workspace command receives the independent `WorkspaceReader` and `WorkspaceSaver` capabilities it needs to calculate and persist an optimistic update. HTTP handlers likewise consume separate workspace read and save interfaces. Do not recombine these into a broad reader/writer interface merely because one adapter implements both.

## HTTP pipeline

The outer-to-inner order is request ID, panic recovery, security headers, CORS, access logging, then routing. Unknown paths and application errors use the stable error envelope:

```json
{"error":{"code":"not_found","message":"resource not found","requestId":"..."}}
```

Successful resources use `{"data": ...}`. Transport parsing failures never expose decoder internals. Unknown failures are logged once at the safe outer boundary and returned as `internal_error`.

## Module contract

`bootstrap.Module` is the only feature-to-process seam. A module has a unique name, an optional explicit route registrar, an optional bounded readiness check, and optional start/shutdown callbacks. Startup proceeds in declaration order; shutdown proceeds in reverse order. Module construction must return a complete valid value before `NewApplication`.

Bootstrap contains wiring, never request or business branching. Adding a feature route is explicit; interface assertions do not silently decide whether an endpoint exists.

## Validation and transactions

Validation is split by meaning: transport syntax in handlers, application preconditions in services, invariants in domain constructors/methods, and integrity duplication in persistence constraints. Multi-write commands use a transaction manager that hands an explicit bounded transaction context and transaction-bound repositories into a callback. Every repository operation in that callback uses the supplied transaction context. The transaction itself is bound by the concrete repository values, never placed in the context; context carries only cancellation and deadline semantics. Network calls do not occur inside database transactions without an accepted ADR.

## Executable fitness rules

The root `../../architecture/rules.yaml` indexes stable rule IDs. The standard-library checker in `tools/archlint` enforces domain purity, the layer matrix, request-context propagation, and bounded HTTP decoding. Each rule has compliant and violating fixtures plus a short root rule page. `make arch` runs both repository checks and fixture tests; `make arch-explain RULE=...` explains remediation offline.

The checker deliberately covers only high-signal invariants with deterministic diagnostics. Standard concerns remain with the compiler, vet, golangci-lint, race tests, PostgreSQL integration tests, migration replay, govulncheck, and container/release gates. A prose rule is not considered blocking until it has an executable check and a negative fixture.
