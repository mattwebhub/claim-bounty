# Contributing

Use a short descriptive branch without the `agent/` prefix. Keep commits focused and open a pull request against protected `main`.

Before handoff:

1. Run `./scripts/agent-context <changed-path>`.
2. Run `make check-fast` while iterating.
3. Add focused tests for changed behavior and boundaries.
4. Run `make check` before requesting review.
5. Run `make test-system` for cross-stack or contract changes.

Use `make check-ci` before a release or after persistence, infrastructure, dependency, or container changes. It adds isolated PostgreSQL integration, vulnerability scans, production image builds, and the real system flow to the deterministic `make check` gate.

Do not lower thresholds, weaken lint rules, edit generated API types, or create architecture exceptions merely to make a gate pass.
