# ADR 0003: Feature-first web boundaries and single state ownership

- Status: accepted
- Date: 2026-08-27

## Decision

Use `shared <- features <- routes <- app`. Each feature exposes one public entrypoint. TanStack Query owns server state, the router owns navigation state, Zustand owns synchronous workspace workflow state, React Hook Form owns form drafts, and component state owns private presentation.

## Consequences

Raw HTTP remains behind `shared/api`; feature services validate and map remote DTOs. The workspace controller is the explicit synchronization seam between a server snapshot and an editable local draft. Dependency rules and behavior tests reject implicit cross-feature and duplicate-state coupling.
