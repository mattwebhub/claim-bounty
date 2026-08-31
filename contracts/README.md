# ClaimBounty Contracts

> **Status**: Active | **Updated**: 2026-08-30 | **Scope**: Hosted intake API and manual local handoff

`openapi.yaml` is the canonical HTTP contract. `schemas/v1/` contains the canonical JSON Schemas for the immutable export manifest and the three manual dispatch documents. Representative valid instances live in `examples/v1/` and are checked with the schemas.

The `v1` directory is a compatibility boundary. Compatible additions may add optional fields or enum values only after consumers are reviewed. Renaming or removing a field, tightening an accepted value, changing units, or changing required data creates a new major schema directory and a new media or file-format version.

All sizes are bytes, JSON and file-format hashes are lowercase SHA-256 hex, identifiers are UUIDs unless a pattern says otherwise, and timestamps are RFC 3339 date-times in UTC. `manifestSha256` hashes RFC 8785 canonical JSON with that property omitted. Download responses use the RFC 9530 header format `Content-Digest: sha-256=:BASE64:`, where `BASE64` is the padded standard-base64 encoding of the same 32 digest bytes exposed as lowercase hex in the resource's `sha256` field. The offline export verifier enforces archive-level checks such as duplicate normalized paths, symlinks, expansion bounds, undeclared members, and byte-for-byte checksums because JSON Schema cannot inspect archive contents.

Browser uploads use the same-origin `POST /api/v1/orders/{orderId}/files` multipart gateway. The API bounds and streams one file into a server-generated, write-once quarantine key while computing SHA-256. Private object storage has no browser endpoint. Every inspection, promotion, export, and download reads the exact recorded object version or generation and verifies its SHA-256 before use.

Authenticated and identity responses are non-cacheable. Private order, administration, download, session, and identity-challenge responses declare `Cache-Control: private, no-store`. `DELETE /api/v1/session` revokes the server-side session before expiring the secure cookie. Draft recovery uses the submitter-owned order URL and `GET /api/v1/orders/{orderId}`; the browser keeps the identifier in the URL rather than private local storage.

The exported archive has this fixed root layout:

```text
case-bundle/
  CASE-MANIFEST.json
  paper/
  attachments/
dispatch/
  audit-request.json
  scientific-policy.json
  execution-policy.json
```

`manifestPath` and `dispatch.*.path` are archive-root relative. Each manifest `files[].path` and the artifact path inside `dispatch/audit-request.json` are relative to the extracted `case-bundle/` routine working directory, so `paper/study.pdf` is valid and `case-bundle/paper/study.pdf` is rejected in those fields. A root-level `CASE-MANIFEST.json`, a presigned browser upload, or an upload-completion callback is not a compatible v1 artifact. The exporter must reject a missing, stale, or unvalidated pinned routine revision; a recorded exception cannot pass the release gate.

Submission records explicit customer authorization for upload processing and internal analysis under a named terms version. The P0 contract fixes `externalRedistributionAuthorized` to `false` in the submission request, audit request, and case manifest. No administrator, exporter, or local operator may widen that authority.

The retention policy is server-owned. Customer submission freezes `PII_RETENTION_POLICY_VERSION`, a source deletion deadline no later than `submittedAt + SOURCE_RETENTION_MAX_DURATION`, and a PII deadline no later than `submittedAt + PII_RETENTION_MAX_DURATION`; the source deadline cannot follow the PII deadline. The customer and administrator do not select longer values. An administrator preparing the audit request must preserve the policy version and P0 hard-delete disposition and may preserve or shorten either frozen deadline. At the applicable deadline, the service deletes source objects and hard-deletes PII, including email addresses, identity tokens, sessions, IP-derived metadata, and reversible lookup or mapping data. Preserved scientific outputs cannot retain those values. Every admin request rechecks the normalized email against the active allowlist and confirms that the session's authorization policy version is current.

An operator must verify the downloaded ZIP against the expected whole-archive digest before any ZIP parsing or extraction. The command contract is `api verify-export <archive.zip> <expected-sha256-hex> [new-destination]`, where the expected digest is exactly 64 lowercase hexadecimal characters. The value comes from the export resource's `sha256` field. The response `Content-Digest` is the same digest in RFC 9530 base64 form.

Run `make contract-check` for OpenAPI lint, standalone schema compilation, example validation, and generated TypeScript drift.
