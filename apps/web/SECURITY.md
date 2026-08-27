# Security policy

## Supported versions

Security fixes are made against the latest release and `main`. Consumers should update rather than relying on unsupported historical versions.

## Reporting a vulnerability

Use the repository's **Security → Report a vulnerability** flow. Do not open a public issue, attach production data, or include credentials. Maintainers must enable GitHub private vulnerability reporting before publication.

Include the affected version, impact, minimal reproduction, and any suggested mitigation. Maintainers should acknowledge a report within three business days and coordinate disclosure after a fix is available.

## Deployment responsibilities

- Browser `VITE_` values are public and must never contain secrets.
- Serve the application over HTTPS and configure an environment-specific Content Security Policy at the edge.
- Keep API cookies `Secure`, `HttpOnly`, and appropriately `SameSite`; this frontend does not persist credentials.
- Review dependency, CodeQL, and container findings before release.
