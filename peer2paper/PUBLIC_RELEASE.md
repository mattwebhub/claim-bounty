# Public release policy

Peer2Paper is intended for release as a public monorepo. Product code, schemas, reusable scientific workflow definitions, documentation, sanitized examples, and tests belong in the repository.

Operational runs and third-party research inputs do not. These directories are local/private and ignored by Git:

- `peer2paper/audits/`
- `peer2paper/assessment-bundles/`
- `peer2paper/state/`
- `peer2paper/validation/cases/`

They may contain copied papers, datasets, benchmark answer keys, internal review packages, or request-specific release restrictions. Public examples are authored independently under `peer2paper/examples/` and contain only reviewed aggregate facts, attribution, limitations, and links to authoritative sources. They never inherit redistribution rights from a repository-level licence.

The `elite-party-cues` example is the reference publication pattern: its narrative and structured result are public, while the paper, participant-level data, third-party analysis code, hidden answer key and complete audit records remain outside Git.

Test fixtures must be unmistakably synthetic. Values copied or lightly disguised from a private answer key remain prohibited even when the key filename and target identifiers are changed.

## Before making the repository public

From the monorepo root:

```bash
node scripts/check-public-release.mjs
npm --prefix peer2paper/web run ci
python3 -m unittest discover -s peer2paper/execution/tests -p 'test_*.py'
python3 -m unittest discover -s peer2paper/grading/tests -p 'test_*.py'
```

The release checker scans the Git index and fails if private runtime directories, restricted-release markers, secret-like values or prohibited research file types are tracked.

Before changing visibility, also enable GitHub private vulnerability reporting, verify a monitored private conduct-reporting channel, and replace any pending website or mailbox references with endpoints that have been tested from outside the operator's network. These external controls cannot be verified by repository CI.

If prohibited material exists in any reachable commit, removing it in a later commit is insufficient for the initial public release. Rewrite the affected history before changing repository visibility. If it has ever left the local machine, rotate exposed credentials and coordinate invalidation or cleanup with every existing remote and clone.

## Deployment release

Repository readiness does not replace operator configuration. Before announcing a production deployment at [peer2paper.com](https://peer2paper.com), configure the canonical URL, monitored legal and security mailboxes, Supabase project, SMTP, Google OAuth credentials, hosting controls, observability, backups, and legally reviewed privacy and terms text described in `web/README.md`.
