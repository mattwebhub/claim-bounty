# GO-ARCH-002 — dependency direction

Dependencies point inward according to `ARCHITECTURE.md`. Services may import domain and ports; adapters may import domain and ports; transport may import domain and service-facing interfaces; bootstrap alone selects concrete implementations.

Violation: a service importing PostgreSQL, a handler importing an adapter, or `cmd/api` wiring features directly.

Move the external capability behind a consumer-oriented port or move concrete selection to bootstrap. Run `make arch-explain RULE=GO-ARCH-002` and `make arch`.
