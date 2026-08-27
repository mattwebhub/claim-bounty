# ADR 0004: Deterministic verification before reviewer judgment

- Status: accepted
- Date: 2026-08-27

## Decision

Fast local gates enforce formatting, type safety, dependency direction, registry integrity, generated drift, and focused tests. The full infrastructure gate adds race detection, isolated PostgreSQL integration, migration replay, browser/system tests, security analysis, and container builds. Blocking diagnostics use stable rule identifiers and point to local remediation documents.

## Consequences

Agents receive actionable feedback without requiring an approval ceremony for ordinary feature work. Protected `main` requires stable aggregate checks. Exceptions to architecture or mandatory gates require a dated ADR with scope, owner, and expiry.
