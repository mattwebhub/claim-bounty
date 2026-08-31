# ADR 0005: ClaimBounty P0 Contract and Trust Boundaries

> **Status**: Active | **Date**: 2026-08-30 | **Scope**: Hosted intake, private administration, and manual local export

## Context

The repository is adding a ClaimBounty product boundary that accepts untrusted research files from verified-email public users, stores them privately, lets an authorized operator complete intake, and produces a downloadable package for manual local dispatch. The product-fork decision is not part of this contract-first change, so the previous demonstration endpoints remain during migration.

## Decision

P0 ends after an authorized admin downloads an immutable export and verifies it locally. The hosted application does not call `agent_routine_dispatch`, execute scientific code, create scientific run records, process payment, or deliver audit reports to customers. Kubernetes-hosted scientific execution is outside this boundary.

The contract owners are:

| Contract or state                                       | Authoritative owner                            | Consumer assumption                                            |
| ------------------------------------------------------- | ---------------------------------------------- | -------------------------------------------------------------- |
| HTTP paths, DTOs, status codes, auth requirements       | `contracts/openapi.yaml`                       | Backend handlers conform; web types are generated              |
| Export document shapes                                  | `contracts/schemas/v1/`                        | Export builder and local verifier validate the same files      |
| Session identity, authorization, lifecycle, idempotency | API and PostgreSQL                             | Browser holds no authoritative session or order state          |
| Uploaded and exported bytes                             | Private object storage                         | PostgreSQL stores opaque keys and verified metadata only       |
| Order and file state transitions                        | API services and PostgreSQL transaction policy | Workers request transitions and append events through services |
| Admin filters                                           | URL                                            | TanStack Query owns queue and detail server state              |
| Manual scientific dispatch                              | Local ClaimBounty operator                     | Hosted services have no dispatch or execution credentials      |

The browser uses a same-origin Secure, HttpOnly, SameSite cookie. A successful verification rotates the session. Unsafe authenticated methods also require a session-bound CSRF header and allowed `Origin`. Admin sessions require the same email proof plus a server-owned allowlist. Public reads enforce subject ownership and return not-found semantics across subjects.

Multipart uploads and authorized downloads stream through the same-origin API without exposing storage keys or signed URLs. Uploads write once to generated object keys while enforcing byte and digest limits without buffering the complete file. Files remain private and quarantined until size, hash, type, and scan checks pass. Export creation uses an immutable readiness snapshot and refuses unscanned files, missing authority, incomplete policy documents, path collisions, traversal paths, symlinks, and checksum disagreement.

The server owns the retention policy version and maximum source/PII durations. Customer submission freezes both deadlines from that server configuration. An administrator can preserve or shorten those values while preparing the audit request, but cannot change the policy version or disposition and cannot extend either deadline. Production startup requires explicit policy version and duration ceiling settings.

Every authorized file or export download carries an RFC 9530 `Content-Digest` value in the exact `sha-256=:BASE64:` format. JSON metadata continues to expose the same digest as lowercase hexadecimal. Offline export verification requires that hexadecimal whole-archive digest as a command argument and compares it from a single open file handle before constructing a ZIP reader or writing extraction output.

## Compatibility

The ClaimBounty operations are additive during the contract-first stage. Existing project and workspace paths remain until the product owner confirms the product fork and their frontend and backend consumers are replaced.

HTTP compatibility is versioned under `/api/v1`. Export schemas use `schemaVersion: 1.0.0` inside `contracts/schemas/v1`. Schema v1 uses camelCase and is not a declaration that the existing ClaimBounty validation fixtures satisfy it. A local adapter or regenerated fixture must pass the repository schemas before P0 claims routine compatibility.

Each export pins the exact scientific routine definition using a SHA-256 content revision. The only supported handoff consists of a verified `case-bundle` directory plus `audit-request.json`, `scientific-policy.json`, and `execution-policy.json`. The schemas intentionally reject floating empty policies, customer commands, host paths, study-specific entrypoints, and customer-selected resource ceilings.

## Rollout and Recovery

Contract and schema gates land before handlers, migrations, workers, or UI. Backend and frontend owners implement against the generated contract in coordinated follow-up changes. Until the system slice passes, this contract is not a deployed compatibility promise. Rollback before production data exists is a repository revert; after data exists, a new API/schema version and explicit migration are required.

## Consequences

P0 has one narrow hosted trust boundary and one observable manual handoff. Scientific execution remains isolated from internet-facing services. Product work must add rate limits, storage isolation, event auditing, retention cleanup, export verification, and the threat-model checks before accepting real customer files.
