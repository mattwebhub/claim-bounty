# ClaimBounty Representative Trajectories

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Sanitized public routine and current-agent trajectory projection

The public trajectory package is [workflow/claimbounty-scientific-audit/trajectories](../workflow/claimbounty-scientific-audit/trajectories/README.md). It contains seven routine records, nineteen current-agent records, and the 47 ordered steps declared by the public workflow revision.

## Evidence boundary

Each record labels its evidence as `exact`, `equivalent_predecessor`, or `reconstructed` and separately declares whether the source revision is `current` or `predecessor`. Predecessor records are representative evidence of the responsibility and handoff pattern. They do not prove that the current workflow revision completed end to end.

The current revision has exact public summaries for the bounded build and reproduction stages. Later representative records come from predecessor evidence or sanitized reconstruction as labeled. The package makes no clean current-revision end-to-end claim.

## Public records

- [index.json](../workflow/claimbounty-scientific-audit/trajectories/index.json) is the generated inventory and count record.
- [trajectory.schema.json](../workflow/claimbounty-scientific-audit/trajectories/trajectory.schema.json) defines the record contract.
- [sanitization-policy.md](../workflow/claimbounty-scientific-audit/trajectories/sanitization-policy.md) defines allowed fields, transformations, and exclusions.
- `routines/` contains one generated record for each public routine.
- `agents/` contains one generated record for each of the nineteen current agents. Removed predecessor roles are not represented as current agents.

The records retain only allowlisted structural facts: public IDs, ordered steps, coarse statuses and outcomes, supported UTC timestamps, supported retry and correction counts, handoff topology, input classes, bounded error summaries, output artifact kinds, schema state, integrity-match booleans, and release limitations.

## Verification

```sh
node scripts/generate-public-trajectories.mjs --check
node scripts/check-public-trajectories.mjs
node scripts/generate-public-manifests.mjs --check
```

The generator is an explicit reviewed allowlist. It does not ingest raw interaction logs, diagnostics, audits, browser evidence, or comparator preflight material. Public-release validation enforces counts, schema, evidence labels, current-agent membership, URI policy, redaction flags, and forbidden field and value patterns.
