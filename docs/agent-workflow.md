# Agent workflow

Use deterministic feedback before reviewer judgment:

1. Read root and nearest scoped agent instructions.
2. Run `./scripts/agent-context <path>`.
3. Identify the use case, owning layer, state owner, trust boundary, and contract impact.
4. Add the closest failing behavior test when practical.
5. Keep changes inside one coherent boundary.
6. Run focused tests continuously and `make check-fast` after boundary changes.
7. Review the diff for dependency direction, duplicate state, error translation, cleanup, accessibility, and generated drift.
8. Run `make check`; use `make test-system` for cross-stack changes.

For persistence, dependency, container, infrastructure, or release work, use `make check-ci` or run its relevant named targets. `make arch-explain RULE=<ID>` prints the local rule document; do not search for policy that is already versioned with the code.

The pre-commit hook runs backend architecture, vet, unit tests, and golangci-lint findings introduced by the worktree change. The pre-push hook runs the complete deterministic repository gate. A pinned pnpm install activates both hooks automatically; `make hooks` repairs them explicitly.

Architecture changes require an ADR only when they alter repository-wide dependency direction, transaction semantics, state ownership, public API compatibility, trust boundaries, deployment guarantees, or mandatory quality gates.
