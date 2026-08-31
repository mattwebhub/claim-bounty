# Testing strategy

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Repository test lanes and release gates

- Go domain and service tests prove invariants and orchestration without I/O.
- HTTP tests prove bounded decoding, stable error envelopes, headers, middleware, and status mapping.
- PostgreSQL integration tests use isolated schemas and replay migrations up/down/up.
- React unit and component tests prove schemas, state machines, query behavior, and accessible UI states.
- Deterministic Playwright smoke tests use API interception for fast browser feedback.
- The system lane uses the complete ClaimBounty Compose profile, production web build, and Chromium to prove submission, source download, export download, digest validation, and offline handoff.

`make test-integration` and `make test-system` each own a separate disposable Compose project and loopback-only host port. Cleanup removes only the stack created by that command. Override the documented Make variables when parallel jobs need additional isolation.

The required GitHub Actions system job runs `make test-system`; the deterministic mocked browser and accessibility smoke remains a separate job. The required aggregator rejects a failed or skipped applicable system job, and the system suite fails collection if the ClaimBounty gate is marked required without Compose mode.

Coverage thresholds are a ratchet, not a substitute for focused changed-behavior tests. Never lower a threshold to merge a change.
