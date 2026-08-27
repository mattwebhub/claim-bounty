# Security policy

## Reporting a vulnerability

Do not open a public issue. Use the repository's private **Security → Report a vulnerability** form. Include affected versions, impact, reproduction steps, and any suggested remediation. Do not include real credentials or personal data.

Maintainers should acknowledge a report within three business days, validate severity, coordinate a fix and disclosure window, and credit the reporter when requested. There is currently no bug bounty.

## Supported versions

Security fixes target the latest release and `main`. Older template snapshots are not maintained. Consumers own updates after generating or copying this template.

## Baseline

The project scans reachable Go vulnerabilities, dependency changes, source with CodeQL, public-release contents, and the production container build. Dependencies, actions, build tools, and images are pinned and updated through reviewed pull requests. Runtime secrets belong in the deployment environment and must never enter source, logs, metrics, traces, test fixtures, or browser-readable state.
