# Agent workflow

Use deterministic feedback before reviewer judgment:

1. Read root and nearest scoped agent instructions.
2. Run `./scripts/agent-context <path>`.
3. Identify the use case, owning layer, state owner, trust boundary, and contract impact.
4. Add the closest failing behavior test when practical.
5. Keep changes inside one coherent boundary.
6. Run focused tests continuously and `make check-fast` after boundary changes.
7. Review the diff for dependency direction, duplicate state, error translation, cleanup, accessibility, and generated drift.
8. Run `make review` once the change is ready to share.

For persistence, dependency, container, infrastructure, or release work, use `make check-ci` or run its relevant named targets. `make arch-explain RULE=<ID>` prints the local rule document; do not search for policy that is already versioned with the code.

The pre-commit hook runs backend architecture, vet, unit tests, and golangci-lint findings introduced by the worktree change. The pre-push hook requires a clean worktree and outgoing refs that resolve to the checked-out commit, guaranteeing that tests inspect the exact tree being pushed; push a different commit by checking it out first. The hook captures Git's complete outgoing ref input, runs `make check-ci`, and then runs an authenticated, read-only Codex semantic review for every non-deletion range. Medium-or-higher findings block the push. Successful committed-range reviews are cached by range and review-policy hash; set `MICRO1_AI_REVIEW_REFRESH=1` only when intentionally requesting a fresh review of an unchanged range. Use `make ai-review` to rerun the semantic review over uncommitted work while fixing findings.

The Codex CLI and `codex login` are local prerequisites checked by `make doctor`. A pinned pnpm install activates both hooks automatically; `make hooks` repairs them explicitly.

Architecture changes require an ADR only when they alter repository-wide dependency direction, transaction semantics, state ownership, public API compatibility, trust boundaries, deployment guarantees, or mandatory quality gates.
