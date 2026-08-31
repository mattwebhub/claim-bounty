# ClaimBounty Kubernetes Deployment

> **Status**: In-Development | **Updated**: 2026-08-30 | **Scope**: Hosted intake and administration services

The Kustomize base deploys the browser application, same-origin API gateway, intake worker, migration Job, and retention CronJob. `POST /api/v1/orders/{orderId}/files` is the only browser upload path. The ingress sends `/api` to the API and `/` to the web service with request buffering disabled for bounded streaming uploads.

Private object storage has no public ingress. The demo overlay exposes its S3-compatible service only as `ClusterIP`, and a network policy accepts storage traffic only from the API, intake worker, and bucket initializer. The API generates write-once keys and must read the recorded object version or generation and verify SHA-256 during scan, promotion, export, and download.

Scientific execution stays on the operator's local workstation. The hosted worker performs intake inspection, export assembly, and object cleanup only. These manifests contain no scientific-execution worker, routine runner, research tool, or scientific execution queue consumer.

The API, worker/retention, and web pods use separate service accounts with token automount disabled. `claimbounty-api-object-storage` is referenced only by API and migration pods; `claimbounty-worker-object-storage` is referenced only by worker and retention pods. Shared database and encrypted-identity configuration remains in `claimbounty-runtime` because the current process image validates it at startup, but storage credentials never cross those workload roles.

## Overlays

- `overlays/demo` runs one web pod, one API pod, one intake worker, pinned PostgreSQL, S3-compatible storage, Mailpit, and ClamAV images, persistent data services, bucket versioning, and clearly marked demo-only credentials. It is for a disposable local or hackathon cluster.
- `overlays/external-data` runs two web and API replicas, one intake worker, the migration and retention jobs, and expects PostgreSQL, private object storage, SMTP, malware scanning, TLS, and secrets from the target environment.

Both overlays keep the public HTTP contract identical. Only dependency locations and operational scale change.

## Build and validate

Build the web image with its public same-origin API URL. For the example external host:

```sh
docker build -f apps/api/Dockerfile -t claimbounty-api:local .
docker build -f apps/web/Dockerfile -t claimbounty-web:local .
kubectl kustomize infra/kubernetes/overlays/demo >/dev/null
kubectl kustomize infra/kubernetes/overlays/external-data >/dev/null
```

The web bundle uses `/api/v1`, so the same images work behind any approved origin. Release publication creates `linux/amd64` and `linux/arm64` manifests for both images.

Load the local images into the chosen demo cluster, then apply and wait for the one-shot initialization jobs and deployments:

```sh
kubectl apply -k infra/kubernetes/overlays/demo
kubectl wait -n claimbounty-demo --for=condition=complete job/claimbounty-object-storage-init --timeout=10m
kubectl wait -n claimbounty-demo --for=condition=complete job/claimbounty-migrate --timeout=10m
kubectl rollout status -n claimbounty-demo deployment/claimbounty-api --timeout=10m
kubectl rollout status -n claimbounty-demo deployment/claimbounty-worker --timeout=10m
```

The demo values are public and must never be promoted. For a cluster without an ingress address, use the web service's same-origin Caddy gateway and port-forward Mailpit in separate terminals:

```sh
kubectl port-forward -n claimbounty-demo service/claimbounty-web 8080:8080
kubectl port-forward -n claimbounty-demo service/claimbounty-mailpit 8025:8025
```

Open <http://localhost:8080> and <http://localhost:8025>. The demo canonical origin is fixed to `http://localhost:8080`; patch it together with the ingress host before using another address.

## External-data preflight

Before applying the external overlay:

1. Replace both image references with release images pinned by digest.
2. Patch the public host, TLS secret, SMTP settings, object storage endpoint, bucket, region, active policy versions, trusted proxy CIDRs, and the trusted routine revision, validation timestamp, and evidence hash. The production process rejects the checked-in `.invalid` hosts, documentation CIDRs, repeated placeholder digests, and demonstration credentials.
3. Create `claimbounty-runtime` in the `claimbounty` namespace with `DATABASE_URL`, `CLAIMBOUNTY_SESSION_PEPPER`, independently generated `CLAIMBOUNTY_EMAIL_ENCRYPTION_KEY_B64` and `CLAIMBOUNTY_EMAIL_LOOKUP_HMAC_KEY_B64` values, `CLAIMBOUNTY_ADMIN_EMAILS`, and optional SMTP credentials. Review the production `PII_RETENTION_POLICY_VERSION`, `SOURCE_RETENTION_MAX_DURATION`, and `PII_RETENTION_MAX_DURATION` values in `claimbounty-config`; all three are required, duration values use Go syntax such as `720h`, and the source ceiling cannot exceed the PII ceiling. Both email keys must decode to 32 bytes; demo values are public and prohibited in production.
4. Create `claimbounty-api-object-storage` and `claimbounty-worker-object-storage` Secrets, each with its own `CLAIMBOUNTY_S3_ACCESS_KEY` and `CLAIMBOUNTY_S3_SECRET_KEY`. Never place provider root credentials in either Secret.
5. Require TLS verification in `DATABASE_URL`, enable object versioning, deny anonymous bucket access, and apply the documented prefix policies: API quarantine write/delete plus accepted/export authorized exact-version read, with no quarantine read or export mutation; worker quarantine exact-version read/delete and accepted/export read/write/delete for promotion and cleanup.
6. Patch `HTTP_TRUSTED_PROXY_CIDRS` to the narrow ingress-controller workload or node ranges. Confirm the ingress controller permits the configuration snippet that clears client-supplied forwarding headers, then verify the API ignores forwarded headers sent by a direct untrusted peer.
7. Confirm the ingress controller honors the request-body and buffering annotations, then run an upload at both the 50 MiB primary-paper boundary and the 250 MiB attachment boundary.
8. Confirm the active admin allowlist and authorization policy versions match the deployed configuration and that PII deletion or irreversible anonymization is scheduled.
9. Confirm the three `CLAIMBOUNTY_ROUTINE_*` values identify the release-approved local routine and match the admin intake assertion exactly. They authorize export packaging only and do not deploy scientific execution.

Delete the previous completed migration Job before a release, apply the overlay, and block rollout completion on the new Job:

```sh
kubectl delete job claimbounty-migrate -n claimbounty --ignore-not-found
kubectl apply -k infra/kubernetes/overlays/external-data
kubectl wait -n claimbounty --for=condition=complete job/claimbounty-migrate --timeout=10m
kubectl rollout status -n claimbounty deployment/claimbounty-api --timeout=10m
kubectl rollout status -n claimbounty deployment/claimbounty-worker --timeout=10m
```

Render and inspect the exact release before applying it:

```sh
kubectl kustomize infra/kubernetes/overlays/external-data >claimbounty-rendered.yaml
kubectl apply --dry-run=server -f claimbounty-rendered.yaml
kubectl apply -f claimbounty-rendered.yaml
```

Keep the rendered file outside version control because deployment tooling may inject environment-specific identifiers.
