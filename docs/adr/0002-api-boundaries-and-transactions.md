# ADR 0002: Hexagonal API boundaries and explicit transactions

- Status: accepted
- Date: 2026-08-27

## Decision

Dependencies point inward from bootstrap and transport/adapters through services and ports to a standard-library-only domain. Interfaces live at their consuming boundary and describe narrow capabilities. Multi-write invariants execute through a transaction manager that supplies transaction-bound repositories and context.

## Consequences

HTTP syntax, SQL errors, provider DTOs, and runtime acquisition never cross inward. Project creation and its initial workspace commit atomically. Read and write services can evolve independently and tests substitute capabilities without mirroring concrete implementations.
