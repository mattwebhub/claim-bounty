# Peer2Paper Organization Handbook

This organization coordinates reproducible scientific audits. Its checked-in
files define stable roles, routines, and handoffs; request-specific inputs,
runtime state, logs, credentials, and generated audit artifacts are never part
of the public repository.

## Department contract

Each department has three public definition files:

- `_charter.md` states its mission, responsibilities, ownership, and handoffs.
- `_agents.md` defines the agents that may perform its work.
- `_routines.md` indexes its callable routines.

Detailed routines live in `knowledge/routines/`. Runtime directories such as
`active/`, `inbox/`, `outbox/`, `logs/`, and `review/` are local-only.

## Routine rules

A routine definition must name one owning department and agent, declare its
inputs and outputs, and identify any parent routine. Sub-routines are dispatched
only by their declared parent. Every dispatch reference must resolve to an
existing `{department}/{routine}` identifier, and every artifact path must match
the producing routine exactly.

Routine indexes, agent rosters, and routine files must use the same display
names. Names are unique across the organization. Use kebab-case identifiers and
clear human-readable titles.

## Evidence and artifact boundaries

- Treat the frozen study case and sealed artifacts as immutable inputs.
- Keep claims, findings, limitations, and recommendations linked to exact
  evidence locations.
- Never manufacture missing evidence or silently replace a prior result.
- Store grader keys, service tokens, participant-level data, copied research
  sources, and private audit runs outside the repository and agent-visible
  workspace.
- Use `project://peer2paper/...` URIs for project-owned artifacts.

## External sources

Use only authorized sources. Record stable URLs, versions, retrieval dates, and
content hashes where the routine contract requires them. Do not bypass access
controls, persist credentials, or treat search-result snippets as evidence.

## Changes and review

When changing a routine, update its index and agent assignment in the same
change. Validate cross-routine paths and dispatch references before release.
Any action involving production systems, external publication, secrets,
destructive data operations, or authority beyond the active request requires
explicit approval.
