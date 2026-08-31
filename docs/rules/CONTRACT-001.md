# CONTRACT-001 — Canonical protocol

> **Status**: Active | **Updated**: 2026-08-30 | **Scope**: HTTP and portable export contracts

`contracts/openapi.yaml` is the only editable HTTP contract. Versioned portable export contracts live under `contracts/schemas/`; they do not define extra HTTP behavior. Generate the TypeScript schema with `make contract-generate` and never edit generated files. `make contract-check` lints OpenAPI, compiles standalone schemas, validates examples, and fails when generated types drift.

Implemented contract changes require handler tests, feature service validation tests, and the real system test when behavior crosses the browser/API boundary. A contract-first change may precede those consumers only when an ADR records the deliberate incompatibility and follow-up ownership.
