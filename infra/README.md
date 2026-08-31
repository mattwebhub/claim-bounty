# ClaimBounty Deployment Operations

> **Status**: In-Development | **Updated**: 2026-08-30 | **Scope**: Local Compose and hosted Kubernetes intake services

The deployment boundary contains the web application, API, PostgreSQL, private object storage, email delivery, malware scanning, the intake worker, database migration, and retention cleanup. The worker is limited to file inspection, export assembly, and object cleanup. Scientific analysis and routine execution stay on an operator workstation.

## Local Compose

The existing PostgreSQL-only workflow remains available through `make db-up`. The ClaimBounty profile adds the complete local intake stack:

| Service               | Purpose                                           | Host exposure                    |
| --------------------- | ------------------------------------------------- | -------------------------------- |
| `postgres`            | Transactional state and queues                    | `127.0.0.1:5432`                 |
| `object-storage`      | Versioned private source and export objects       | None                             |
| `object-storage-init` | Private bucket and versioning initialization      | None                             |
| `migrate`             | Sole database schema owner                        | One-shot process                 |
| `api`                 | Identity, intake, administration, and downloads   | `127.0.0.1:8080` for diagnostics |
| `web`                 | Public same-origin entrypoint and `/api` proxy    | `127.0.0.1:8081`                 |
| `worker`              | Inspection, export assembly, and queued cleanup   | None                             |
| `mailpit`             | Development SMTP and captured mail                | `127.0.0.1:8025` UI              |
| `scanner`             | ClamAV stream inspection                          | None                             |
| `retention`           | Scheduled PII and object expiry processing        | On-demand maintenance profile    |
| `verify-export`       | Offline archive structure and digest verification | On-demand operator profile       |

Start and stop the local stack:

```sh
docker compose --project-name micro1-claimbounty --file infra/compose.yaml --profile claimbounty up --build --wait --wait-timeout 300
docker compose --project-name micro1-claimbounty --file infra/compose.yaml --profile claimbounty ps
docker compose --project-name micro1-claimbounty --file infra/compose.yaml --profile claimbounty down
```

Open <http://127.0.0.1:8081> for ClaimBounty and <http://127.0.0.1:8025> for captured verification mail. The literal `127.0.0.1` host is part of the local origin contract; using `localhost` is a different browser origin and is rejected. Browsers never receive an object-storage address or credential. Caddy proxies the same-origin `/api` path to the private API service.

Run retention manually when testing expiry:

```sh
docker compose --project-name micro1-claimbounty --file infra/compose.yaml \
  --profile claimbounty --profile maintenance run --rm retention
```

Verify a downloaded export before local scientific work:

```sh
mkdir -p exports
cp /path/to/downloaded-export.zip exports/claimbounty-export.zip
mkdir -p verified-exports
EXPORT_SHA256=<expected-sha256-hex> docker compose --project-name micro1-claimbounty --file infra/compose.yaml \
  --profile operator run --rm verify-export
# Equivalent repository shortcut:
make verify-export EXPORT_SHA256=<expected-sha256-hex>
```

Set `EXPORT_SHA256` to the export resource's exact 64-character lowercase hexadecimal `sha256` value before running `make verify-export`. The underlying command is `api verify-export <archive.zip> <expected-sha256-hex> [new-destination]`. It hashes the complete archive and compares that expected value before ZIP parsing or extraction. The HTTP `Content-Digest` header represents the same digest as `sha-256=:BASE64:` using padded standard base64.

The verifier has a read-only archive mount, a separate writable extraction mount, and no network. It validates the complete archive from one open file handle before creating the exclusive `verified-exports/claimbounty-export/` destination, confines writes beneath that new directory, refuses to overwrite an existing destination, and removes partial output after a failed extraction. A zero exit status and the single JSON result containing the case-bundle and dispatch paths confirm the archive layout, schemas, exact file sizes, and SHA-256 digests. Only after verification succeeds may an operator load the reported case-bundle path into Toone and run the pinned scientific-audit routine locally. The hosted API and worker never start or dispatch that routine.

`down` preserves named volumes. `down --volumes` permanently removes the local database, objects, captured mail, and scanner definitions and should be used only for an intentional demo reset.

## Process contract

The API image is shared by application, job, and operator process roles:

- `api serve` starts HTTP intake and administration.
- `api migrate` applies embedded migrations once and exits.
- `api worker` processes inspection, export, and deletion queues until terminated.
- `api retention` processes one bounded retention batch and exits.
- `api verify-export /path/to/archive.zip <expected-sha256-hex> /new/output-directory` verifies the whole archive digest, then safely validates and extracts one downloaded handoff without network or hosted scientific execution. The digest is 64 lowercase hexadecimal characters and the destination must not exist.
- `api healthcheck` checks the serving process readiness endpoint and is used by the container health check.

Automatic migration stays disabled in Compose application processes and every Kubernetes overlay. The one-shot migration process is the sole schema owner.

## Configuration and secrets

Non-secret runtime configuration includes the canonical origin, trusted proxy CIDRs, SMTP and scanner addresses, object-storage endpoint, bucket and fixed prefixes, worker interval, retention batch size, policy version identifiers, and the trusted routine registry entry. When ClaimBounty is enabled in production, the process requires explicit browser trust, proxy trust, identity, email-protection, administrator, retention, routine, object-storage, scanner, and mail settings. Checked-in demonstration values and documentation placeholders are rejected. `CORS_ALLOWED_ORIGINS` and `CLAIMBOUNTY_CANONICAL_ORIGIN` must use HTTPS deployment hosts, and the canonical origin must exactly match an allowed origin. The storage endpoint must use HTTPS. Production SMTP requires `CLAIMBOUNTY_SMTP_TLS_MODE=starttls` or `implicit` and a certificate name in `CLAIMBOUNTY_SMTP_TLS_SERVER_NAME`; it never falls back to plaintext. Set `CLAIMBOUNTY_SMTP_TLS_CA_FILE` to a mounted PEM CA bundle only when the mail server uses a private CA. SMTP development logging must be disabled, and optional SMTP username and password values must be configured together.

Production requires explicit `PII_RETENTION_POLICY_VERSION`, `SOURCE_RETENTION_MAX_DURATION`, and `PII_RETENTION_MAX_DURATION` values. Durations use Go duration syntax such as `720h`, must be positive, and the source ceiling cannot exceed the PII ceiling. `CLAIMBOUNTY_ROUTINE_REVISION`, `CLAIMBOUNTY_ROUTINE_VALIDATED_AT`, and `CLAIMBOUNTY_ROUTINE_EVIDENCE_SHA256` must identify one validated release. Admin intake values are assertions and must match all three configured values exactly. The API body ceiling is 251 MiB so a 250 MiB attachment plus bounded multipart metadata can pass; route and domain checks still enforce the lower primary-paper and total-order limits.

`HTTP_TRUSTED_PROXY_CIDRS` must contain only the private network ranges used by the deployed Caddy or ingress proxy. The API ignores forwarded client addresses from any other peer. Caddy and the NGINX Ingress configuration discard incoming `Forwarded`, `X-Forwarded-For`, and `X-Real-IP` values and write one address derived from their direct client. Do not add a public range or a whole cloud VPC when a proxy workload or node range is available.

Keep these values in a secret store outside version control for any shared environment:

- `DATABASE_URL`
- `CLAIMBOUNTY_SESSION_PEPPER`
- `CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64` containing an independently generated 32-byte encryption key encoded with standard base64
- `CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64` containing a different independently generated 32-byte lookup-HMAC key encoded with standard base64
- `CLAIMBOUNTY_ADMIN_EMAILS`
- API object-store access and secret keys, mapped into `CLAIMBOUNTY_S3_ACCESS_KEY` and `CLAIMBOUNTY_S3_SECRET_KEY` only for the API process
- Worker object-store access and secret keys, mapped into the same process-local names only for worker and retention processes
- `CLAIMBOUNTY_SMTP_USERNAME` and `CLAIMBOUNTY_SMTP_PASSWORD`, when required

The Compose defaults are public demo values. Production must replace them; none is a deployable secret. Kubernetes expects database and identity material in `claimbounty-runtime`, API storage credentials in `claimbounty-api-object-storage`, and worker storage credentials in `claimbounty-worker-object-storage`. The email encryption and lookup-HMAC keys are independent, production-required values and must not be derived from the session pepper or from one another. Rotate session, email, API storage, worker storage, and SMTP credentials independently. An admin allowlist update must increment `CLAIMBOUNTY_ADMIN_ALLOWLIST_VERSION`; an authorization rule change must increment `CLAIMBOUNTY_AUTHORIZATION_VERSION`.

The checked-in MinIO policies under `infra/object-storage/` show the required bucket-prefix boundary. The API may write and delete `quarantine/` objects and perform authorized, exact-version reads from `accepted/` and `exports/`; it cannot read quarantine objects or write or delete exports. The worker may read an exact recorded quarantine version, delete it after verified promotion or cleanup, and read, write, or delete `accepted/` and `exports/`. Neither role receives bucket administration, anonymous access, or access outside those prefixes. Provider IAM must implement the same boundary, while the application continues to require the recorded immutable version and SHA-256 on every read.

## Backup and deletion policy

The demo volumes have no backup guarantee. External deployments should set and test explicit recovery objectives:

- PostgreSQL requires encrypted backups, point-in-time recovery, and a documented restore drill.
- Private object storage requires versioning, encryption, denied anonymous access, and recovery tests that read exact recorded versions.
- Backups contain restricted data. PII deletion deadlines must include backup expiry or destruction of the keys that can decrypt the deleted subject's backup records.
- Restores must not revive expired sessions, identity challenges, deleted PII, or objects whose retention deadline passed. Run retention immediately after a restore and before reopening traffic.

## Validation

Run these before deployment:

```sh
docker compose --file infra/compose.yaml --profile claimbounty --profile maintenance config --quiet
kubectl kustomize infra/kubernetes/overlays/demo >/tmp/claimbounty-demo.yaml
kubectl kustomize infra/kubernetes/overlays/external-data >/tmp/claimbounty-external.yaml
go run github.com/yannh/kubeconform/cmd/kubeconform@v0.7.0 \
  -strict -summary /tmp/claimbounty-demo.yaml /tmp/claimbounty-external.yaml
pnpm contract:check
make test-system
```

`make test-system` uses a dedicated Compose project and non-default host ports. It builds and starts PostgreSQL, versioned MinIO plus its identity/policy initializer, Mailpit, ClamAV, the migration process, API, intake worker, and same-origin web gateway before running the browser suite. Its exit trap tears down only that named project and its disposable volumes.

See [the Kubernetes runbook](kubernetes/README.md) for migration ordering, image pinning, external dependencies, and release checks.
