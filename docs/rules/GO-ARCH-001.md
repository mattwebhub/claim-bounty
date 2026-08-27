# GO-ARCH-001 — pure domain

Domain code may import only the Go standard library. It owns invariants and values and must remain independent of HTTP, SQL, logging, configuration, clocks, ID generation, and providers.

Compliant: accept acquired time and identifiers as values. Define an outbound capability in `ports` and implement it in `adapters`.

Violation: importing pgx, Goose, a provider SDK, or another internal layer from `internal/domain`.

Run `make arch-explain RULE=GO-ARCH-001` and `make arch`.
