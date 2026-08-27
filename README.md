# Micro1 Go API template

A production-shaped Go HTTP API starter optimized for fast human and coding-agent feedback. It uses the standard library HTTP stack, explicit composition, strict environment validation, stable JSON envelopes, bounded request bodies, structured logs, readiness registration, and graceful reverse-order shutdown.

## Quick start

Requires Go 1.26.6 or newer. Patch releases are security requirements, not optional upgrades.

```sh
make doctor
make dev
```

`make dev` starts the pinned ephemeral PostgreSQL service and runs the API with safe development defaults. Copy `.env.example` only when you need overrides, then export it through your process runner; the API deliberately does not load dotenv files.

The process listens on `127.0.0.1:8080` by default:

- `GET /health/live` proves the process can answer HTTP.
- `GET /health/ready` proves it is accepting traffic and registered dependencies are healthy.

Run `make check-fast` during the edit loop and `make check` before handoff. `make check-ci` adds an ephemeral PostgreSQL migration up/down/up, integration tests, vulnerability scanning, and a production container build. The latter owns and removes only the Compose stack declared in this repository.

## Configuration

Configuration is read once at startup. Invalid values stop the process before it binds a port. See [.env.example](.env.example) for every foundational setting. Production rejects wildcard CORS. Secrets must be injected by the runtime, never committed or logged.

## Agent feedback

`make context CONTEXT_PATH=internal/services` prints the applicable boundary and nearest tests. `make arch` runs dependency/context/body rules plus positive and negative fixtures. `make arch-explain RULE=GO-ARCH-002` provides an offline compliant example, violation, documentation path, and verification command. See [`architecture/rules.yaml`](architecture/rules.yaml) for stable rule IDs.

Install repository hooks with `make hooks`. Pre-commit stays focused on formatting, architecture, unit tests, and public hygiene; pre-push runs the complete local gate. CI independently repeats every required check.

## Adding a feature

Read [AGENTS.md](AGENTS.md) and [ARCHITECTURE.md](ARCHITECTURE.md) first. A feature normally adds matching files through `domain`, `ports`, `services`, an external adapter, and `transport/httpapi/handlers`; bootstrap is changed only to wire the complete module. Handlers register explicit routes through `httpapi.RouteRegistrar`. A module may also contribute one bounded readiness check and lifecycle callbacks.

Every I/O API accepts `context.Context`; every body is decoded with `response.DecodeJSON`; every response uses `response.WriteData` or the central error mapper. Request IDs are available with `middleware.RequestIDFromContext` and returned as `X-Request-ID`.

## Container

The multi-stage image builds a static binary and runs from `scratch` as numeric non-root user `65532`. It needs no writable filesystem. Bind to `0.0.0.0` in a container:

```sh
docker build -t micro1-go-template .
docker run --read-only --cap-drop=ALL -p 8080:8080 -e SERVER_HOST=0.0.0.0 micro1-go-template
```

## Tool version policy

Go is pinned in `.go-version` and `go.mod`. Make targets pin golangci-lint, govulncheck, Goose, and Lefthook. GitHub Actions and container bases use immutable commit or image digests with readable version comments/tags; Dependabot proposes reviewed updates. Never replace these with `latest` or an unreviewed floating reference.

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch/review expectations and [SECURITY.md](SECURITY.md) for private disclosure.
