# Contributing

Use a short descriptive branch without the `agent/` prefix. Keep commits focused and open a pull request against protected `main`.

`pnpm install --frozen-lockfile` installs the pinned Lefthook binary and activates the repository hooks. Run `make hooks` if a Git client or worktree needs them refreshed.

Before handoff:

1. Run `./scripts/agent-context <changed-path>`.
2. Run `make check-fast` while iterating.
3. Add focused tests for changed behavior and boundaries.
4. Run `make check` before requesting review.
5. Run `make test-system` for cross-stack or contract changes.

Use `make review` before sharing work. It runs `make check-ci`—including isolated PostgreSQL integration, vulnerability scans, production image builds, and the real system flow—then performs a read-only Codex semantic review. The pre-push hook runs this same flow automatically over Git's exact outgoing refs. Use `make ai-review` while fixing semantic findings without repeating the deterministic suite.

Do not lower thresholds, weaken lint rules, edit generated API types, or create architecture exceptions merely to make a gate pass.
