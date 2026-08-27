# API application

The Go application demonstrates explicit domain, port, service, adapter, transport, and composition boundaries with a real Project/Workspace vertical slice.

Run focused commands from this directory:

```bash
make check-fast
make check
make test-integration TEST_DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/app?sslmode=disable'
```

Run PostgreSQL and both applications from the repository root with `make dev`. Build the production image from the root context:

```bash
docker build -f apps/api/Dockerfile -t micro1-api .
```

Read `AGENTS.md` and `ARCHITECTURE.md` before changing a service, transaction, transport, or persistence boundary. The root `contracts/openapi.yaml` is the canonical protocol contract.
