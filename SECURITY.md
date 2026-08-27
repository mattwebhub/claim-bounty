# Security Policy

Report vulnerabilities through GitHub private vulnerability reporting. Do not open a public issue for suspected vulnerabilities.

Supported releases receive dependency, CodeQL, secret-scanning, and reachable-vulnerability checks. Browser configuration is public by definition: never place credentials or private tokens in `VITE_` variables.

The template intentionally excludes partial authentication. Add authentication only with an explicit threat model, server-managed authorization, secure credential storage, rotation, revocation, and abuse controls.
