# Public Trajectory Sanitization Policy

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: ClaimBounty public trajectory records

## Allowed projection

The generator uses a reviewed, explicit allowlist of public routine and agent IDs, ordered public step IDs, coarse statuses and outcomes, supported UTC timestamps, retry and correction counts, handoff topology, input classes, bounded error summaries, output artifact kinds, schema status, integrity-match booleans, and release limitations.

Opaque record locations use the `trajectory://` scheme. When integrity evidence is supported, the projection uses an export-local evidence ID, the algorithm name, and the match result. It never publishes an original digest.

## Required classification

Every record declares one of `exact`, `equivalent_predecessor`, or `reconstructed`, plus a `current` or `predecessor` source revision relation. Every record sets `contains_research_payload` and `contains_session_metadata` to `false` and lists the redaction classes applied.

The package does not claim a clean current-revision end-to-end run. Predecessor evidence is representative only.

## Excluded material

Generated records exclude claim wording, scientific values, manuscript passages, scientific source identities, research data, variable names, participant information, grading material, local or internal paths, original digests, run and invocation identifiers, interaction or account metadata, email and authentication data, prompt or message content, tool inputs and outputs, shell commands, environment details, browser material, screenshots, credentials, headers, host details, and uncleared artifacts.

The generator does not read raw messages, diagnostics, audit files, browser evidence, or comparator preflight material. Its source is the reviewed summary encoded in the generator allowlist.
