# Contributing to Peer2Paper

Thank you for helping make scientific claim audits easier to inspect and reproduce.

## Before opening a pull request

1. Create a focused branch from `main`.
2. Keep reusable code, schemas, tests, and documentation separate from request-specific audit material.
3. Install exact web dependencies with `npm --prefix peer2paper/web ci`.
4. Run the complete verification commands in the root README.
5. Explain the user or scientific impact, document important tradeoffs, and add proportionate tests.
6. Keep unrelated formatting or generated-file changes out of the pull request.

## Publication boundary

Never commit:

- credentials, private keys, tokens, or non-example environment files;
- audit runs, assessment bundles, reviewer packets, or internal request records;
- copied papers, supplements, datasets, participant-level records, or third-party code without documented redistribution rights;
- hidden benchmark answers, grader keys, or fixtures derived from them; or
- personal or confidential information that is unnecessary for the public project.

Public case studies must contain independently authored narrative, reviewed aggregate facts, authoritative source links, and an explicit publication boundary. A public source is not automatically a redistributable source.

## Review-sensitive changes

Changes to scientific verdict language, numerical comparison rules, authentication, Row Level Security, privacy behavior, licensing, or legal terms require an appropriate domain reviewer. Security-sensitive reports belong in the private channel described in [SECURITY.md](SECURITY.md), not in an issue or pull request.

By contributing, you agree that your contribution is licensed under the repository's [MIT License](LICENSE) and that you have the right to submit it.
