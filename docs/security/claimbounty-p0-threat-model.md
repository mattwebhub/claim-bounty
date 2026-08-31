# ClaimBounty P0 Threat Model

> **Status**: Active | **Updated**: 2026-08-30 | **Scope**: Public intake, private storage, admin review, and manual export

## Assets and Trust Boundaries

Protected assets are email identities, session tokens, CSRF tokens, claim text, filenames, uploaded research files, policy documents, event history, and export archives. The public browser, admin browser, API, PostgreSQL, private object storage, inspection worker, email provider, and local operator workstation are separate trust boundaries. Scientific code execution is not a hosted boundary in P0.

## Security Invariants

- Email challenges have generic responses, short expiry, hashed codes, bounded attempts, rotation on success, and limits by IP prefix, normalized email hash, session, and order.
- Public ownership and admin role checks happen server-side on every resource access. Identifiers provide no authority.
- Sessions use `__Host-` Secure, HttpOnly, SameSite cookies. Unsafe authenticated requests require an allowed origin and a session-bound CSRF token.
- Object keys are random and contain no email, claim, or customer filename. Buckets are private; bounded uploads and authorized attachment downloads stream through the API without exposing storage capabilities.
- Uploaded bytes are never rendered inline, expanded, parsed deeply, or executed by the API or web process. Rejected or unavailable scans remain quarantined.
- Logs, metrics, events, and URLs exclude email addresses, claim text, filenames, tokens, signed URLs, and file contents unless a narrowly reviewed event field requires them.
- Downloads carry an RFC 9530 SHA-256 `Content-Digest`. Exports are immutable, checksum-addressed attachments. The offline verifier requires the expected whole-archive SHA-256 and checks it before ZIP parsing, then checks normalized paths, duplicate paths, symlinks, manifest and dispatch schemas, required roles, and file hashes before extraction to a new directory.
- Submission freezes the server-owned retention policy version and source/PII deadlines within configured ceilings. Administrative intake may preserve or shorten those deadlines, never extend them.
- Hosted credentials cannot dispatch routines or execute scientific workloads. Local execution requires a separate operator decision after verification.

## Threats and Required Controls

| Threat                                                | Required control and verification                                                                                                 |
| ----------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Email spoofing or address discovery                   | Generic challenge response, code proof, allowlist for admin purpose, expiry and attempt tests                                     |
| Challenge brute force and upload denial of service    | Layered rate limits, file count and byte budgets, idempotency, bounded bodies, `429` and `413` tests                              |
| Session theft, CSRF, or privilege confusion           | Cookie attributes, rotation, origin checks, session-bound CSRF, admin-role tests, revocation support                              |
| Cross-order or cross-file access                      | Subject-scoped repository queries, opaque object keys, deny tests for every read and mutation                                     |
| Malicious documents, archives, or decompression bombs | Allowlist, signature checks, quarantine, malware scan, no automatic archive expansion, size and timeout tests                     |
| Path traversal, Unicode collisions, or symlink export | Server-generated paths, normalized-path uniqueness, symlink rejection, verifier adversarial fixtures                              |
| Stored script injection through text or filenames     | Contextual output escaping, text-only display, attachment disposition, browser tests with hostile values                          |
| Storage capability disclosure or replay               | No browser-visible storage URLs, write-once generated keys, bounded streaming, `no-store`, and no logging                         |
| Object mutation or export tampering                   | Completion metadata checks, `Content-Digest`, immutable export key, pre-parse whole-archive SHA-256 verification, manifest hashes |
| PII in events and observability                       | Field allowlists, structured redaction tests, access-controlled admin detail, retention cleanup                                   |
| Unsafe local scientific execution                     | No hosted execution identity, policy ceilings, disabled analysis network, manual verified dispatch                                |

## Residual Risk and Release Gate

Verified email is contact proof, not strong identity proof. Admin email authentication is acceptable only for an allowlisted local-operator P0 and should move behind organization SSO when available. Malware scanning reduces known-file risk but does not make documents safe to open or code safe to run.

No real customer upload is accepted until abuse controls, private storage policies, scanner failure behavior, retention deletion, cross-subject authorization tests, export adversarial tests, secret scanning, and the end-to-end hosted intake and manual verification path pass.
