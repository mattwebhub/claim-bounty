# Contributing

## Workflow

1. Run `make doctor` and read `AGENTS.md` plus `ARCHITECTURE.md`.
2. Create a short, plainly named branch such as `project-pagination`. Never use the `agent/` prefix.
3. Use `make context CONTEXT_PATH=path/to/change` before adding a boundary.
4. Keep commits focused and add regression tests with the implementation.
5. Run `make check-fast` during development and `make check` before opening a pull request.
6. Run `make check-ci` when persistence, migrations, infrastructure, or release behavior changes.

Pull requests are squash-merged into protected `main`. Keep API and migration changes backward compatible unless the pull request explicitly documents a versioned breaking change. Never rewrite an applied migration; add a new monotonically numbered migration with a tested down path.

## Design expectations

Describe the named user/system intent, boundary owners, transaction and concurrency behavior, error translations, cleanup paths, and evidence. Interfaces belong to their consumer. A new package needs a precise bounded noun and a real consumer. Architecture exceptions require an ADR with scope, owner, reason, and expiry.

Generated SQL files are changed only through their generator. Keep generated output in the same pull request as source queries and migrations. Do not reduce lint, test, security, or architecture enforcement to make a change pass.

## Verification levels

- `make check-fast`: formatting, architecture fixtures, vet, and unit/HTTP tests.
- `make check`: fast gate plus pinned golangci-lint, race tests, build, and public hygiene.
- `make check-ci`: complete local gate plus disposable PostgreSQL, migration up/down/up, integration tests, vulnerability scan, and container build.

Handoffs state exact commands run and anything skipped. A CI link is not a substitute for explaining a known risk.
