# Contributing

## Development flow

1. Use a short, descriptive branch without an `agent/` prefix.
2. Install exactly from the lockfile with `pnpm install --frozen-lockfile`.
3. Run `pnpm hooks:install` once to enable staged formatting/linting and the pre-push gate.
4. Read `ARCHITECTURE.md`, identify each state owner, and work through public feature boundaries.
5. Add the nearest behavior-focused test and run `pnpm verify:fast` before requesting review.

Use small pull requests and squash merge into protected `main`. Do not commit generated browser reports, coverage, build output, secrets, or unexplained lint suppressions.

## Verification

- `pnpm verify:fast`: formatting, lint, strict types, architecture, and unit/component tests.
- `pnpm build && pnpm bundle:check`: production compilation and compressed size budgets.
- `pnpm e2e:install && pnpm e2e`: Chromium system and axe accessibility smoke tests.
- `pnpm verify`: the complete local release gate.

If a check cannot run, state the exact command, reason, and missing evidence in the pull request. Architecture exceptions require a documented decision and matching automated rule change.

Coverage thresholds are a ratchet, not a target. Pull requests may raise them and must not lower them; exclusions require the same architectural review as source changes.

## Commit and release hygiene

Write imperative commit subjects and explain the reason for non-obvious changes. Releases use `vMAJOR.MINOR.PATCH` tags and follow `RELEASE.md`. A tag produces immutable build evidence but does not publish automatically.
