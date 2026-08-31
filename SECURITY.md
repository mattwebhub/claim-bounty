# Security Policy

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Public repository, hosted ClaimBounty P0, and local workflow package

Report vulnerabilities through GitHub private vulnerability reporting. Do not open a public issue for suspected vulnerabilities.

Supported releases receive dependency, CodeQL, secret-scanning, and reachable-vulnerability checks. Browser configuration is public by definition: never place credentials or private tokens in `VITE_` variables.

ClaimBounty P0 adds verified-email public sessions and allowlisted admin sessions under the explicit [P0 threat model](docs/security/claimbounty-p0-threat-model.md). Authentication remains server-managed and requires secure credential storage, rotation, revocation, origin and CSRF controls, and abuse limits before real customer use.

Do not upload customer data, hidden evaluation keys, restricted research material, production exports, internal organization configuration, or workstation paths to the public repository. Use synthetic inputs for demonstrations. Authorized export downloads must be checked against the trusted whole-archive SHA-256 value before ZIP parsing or extraction.

The local workflow can execute study-provided code. Run it in an isolated environment with the least filesystem and network access the study requires, inspect the frozen manifest first, and treat produced HTML and data files as untrusted until reviewed.
