# Representative Public Trajectories

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Sanitized routine and current-agent trajectory summaries

This package contains representative, sanitized trajectories for the seven public workflow routines and nineteen current agents. It is a structural review aid. It does not contain research evidence and does not claim that the current 47-step workflow revision completed end to end.

## Evidence labels

- `exact` means the summarized event or provenance is directly supported for the stated source revision.
- `equivalent_predecessor` means a predecessor revision demonstrates equivalent responsibility or handoff behavior, not completion of the current revision.
- `reconstructed` means the record was rebuilt from the sanitized inventory and public workflow topology because an exact shareable trajectory was unavailable.

Every record separately labels `source_revision_relation` as `current` or `predecessor`. A predecessor record must not be read as current-revision execution evidence.

## Contents

- [index.json](index.json) lists all generated records and package counts.
- [trajectory.schema.json](trajectory.schema.json) defines the allowlisted record contract.
- [sanitization-policy.md](sanitization-policy.md) defines the public evidence boundary.
- `routines/` contains seven generated routine records and 47 ordered public workflow steps.
- `agents/` contains nineteen generated current-agent records.
- `MANIFEST.sha256` covers the package files using package-relative paths.

Run `node scripts/generate-public-trajectories.mjs --check` to detect generated drift and `node scripts/check-public-trajectories.mjs` to validate schema, counts, labels, agent membership, and security rules.
