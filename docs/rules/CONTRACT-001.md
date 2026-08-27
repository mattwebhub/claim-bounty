# CONTRACT-001 — Canonical protocol

Only `contracts/openapi.yaml` is hand-edited. Generate the TypeScript schema with `make contract-generate`; never edit generated files. `make contract-check` fails when generation changes the committed tree.

Contract changes require handler tests, feature service validation tests, and the real system test when behavior crosses the browser/API boundary.
