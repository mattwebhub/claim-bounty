# Testing strategy

- Go domain and service tests prove invariants and orchestration without I/O.
- HTTP tests prove bounded decoding, stable error envelopes, headers, middleware, and status mapping.
- PostgreSQL integration tests use isolated schemas and replay migrations up/down/up.
- React unit and component tests prove schemas, state machines, query behavior, and accessible UI states.
- Deterministic Playwright smoke tests use API interception for fast browser feedback.
- The system lane uses a real PostgreSQL database, running Go API, production web build, and Chromium.

`make test-integration` and `make test-system` each own a separate disposable Compose project and loopback-only host port. Cleanup removes only the stack created by that command. Override the documented Make variables when parallel jobs need additional isolation.

Coverage thresholds are a ratchet, not a substitute for focused changed-behavior tests. Never lower a threshold to merge a change.
