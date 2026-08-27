# ADR 0001: One monorepo and one protocol contract

- Status: accepted
- Date: 2026-08-27

## Decision

Keep the Go API in `apps/api`, the React application in `apps/web`, and the sole editable HTTP contract in `contracts/openapi.yaml`. Use a root Makefile and pnpm workspace without an additional task orchestrator. Keep the Go application as one independent module; add no `go.work` until another Go module exists.

## Consequences

One clone and one CI graph can verify cross-stack behavior. App deployability remains independent. Generated frontend DTOs are committed for review but never edited. Shared application packages require two concrete consumers.
