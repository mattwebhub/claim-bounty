# Release checklist

1. Start from a clean clone using a supported Node and pnpm version.
2. Review dependency updates, licenses, security alerts, and generated-file drift.
3. Run `pnpm verify`, then build and health-check the production container.
4. Confirm browser environment values contain no secrets and point to the intended API.
5. Confirm private vulnerability reporting is enabled.
6. Update `CHANGELOG.md`, tag `vMAJOR.MINOR.PATCH`, and retain CI, browser, bundle, and container evidence.
7. Smoke-test the deployed root and deep-link routes, rollback path, telemetry sink, and API failure state.

Do not release with ignored type errors, failed budget/security checks, unreviewed architecture exceptions, proprietary assets, or a dirty generated tree.
