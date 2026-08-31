# ROUTINE: Assemble Final Audit

## Metadata
- **Routine Schema:** 2
- **Department:** peer2paper-operations
- **Agent:** Peer2Paper Orchestrator
- **Cadence:** OnDemand
- **Type:** sub-routine
- **Parent:** peer2paper-operations/run-peer2paper-scientific-audit
- **Max Concurrency:** 1
- **Status:** Production

## Purpose
Publish one canonical JSON audit, a concise researcher-facing HTML report, and a replay package with scoped release checks. PDF is outside the required hackathon output.

## Inputs
| ID | Kind | Source | Binding | Required | Type | Description |
|----|------|--------|---------|----------|------|-------------|
| frozen-study-case-resource | resource | Parent | project://peer2paper/subroutine-inputs/frozen-study-case.json | dispatch | file | Parent-produced frozen study case. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| reproduction-package-resource | resource | Parent | project://peer2paper/subroutine-inputs/reproduction-package.json | dispatch | file | Parent-produced reproduction package. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| robustness-map-resource | resource | Parent | project://peer2paper/subroutine-inputs/robustness-map.json | dispatch | file | Parent-produced robustness map. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| literature-evidence-resource | resource | Parent | project://peer2paper/subroutine-inputs/literature-evidence-package.json | dispatch | file | Parent-produced literature evidence package. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| adjudication-package-resource | resource | Parent | project://peer2paper/subroutine-inputs/adjudication-package.json | dispatch | file | Parent-produced adjudication package. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| audit-request | parameter | Parent | - | dispatch | json | Audit scope, intended internal audience, privacy, retention, redaction, and release boundaries. |
| execution-policy | parameter | Parent | - | dispatch | json | Rendering, replay, storage, access, and internal release policy. |
| canonical-schema-bundle-resource | resource | Parent | project://peer2paper/config/canonical-schemas.json | dispatch | file | Parent-supplied versioned canonical schema bundle. |
| audit-template-resource | resource | Parent | project://peer2paper/config/audit-template.html | dispatch | file | Parent-supplied pinned template used for the researcher-facing HTML report. |
| audit-schema-resource | resource | Parent | project://peer2paper/config/audit-schema.json | dispatch | file | Parent-supplied pinned JSON audit schema. |
| manuscript-recommendations-resource | resource | Parent | project://peer2paper/subroutine-inputs/manuscript-recommendations.json | dispatch | file | Parent-produced concrete manuscript changes, each linked to a verified finding and exact affected text or section. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |

## Workflow
1. **Build canonical audit and replay package**: Build one canonical JSON audit and replay directory from the frozen case, reproduction, sensitivity, targeted evidence, adjudicated findings, and complete manuscript recommendations. Keep raw logs, hashes, prompts, and provenance in the machine package.
   - id: build-canonical-audit
   - agent: peer2paper-delivery/audit-report-builder
   - routine-input: frozen-study-case-resource
   - routine-input: reproduction-package-resource
   - routine-input: robustness-map-resource
   - routine-input: literature-evidence-resource
   - routine-input: adjudication-package-resource
   - routine-input: manuscript-recommendations-resource
   - routine-input: audit-request
   - routine-input: execution-policy
   - routine-input: canonical-schema-bundle-resource
   - routine-input: audit-schema-resource
   - output-artefact: audit-json
   - output-artefact: replay-package
   - completion: The canonical JSON contains the verdict, reported and reproduced values, numerical differences, sensitivity result, verified conflicts, concrete manuscript recommendations, limitations, and content-addressed references to technical evidence.
   - completion: The JSON validates against the pinned schema and uses one value for every displayed number and statement.
   - completion: The replay package contains the shared execution profile, exact commands, 600-second timeout, environments, inputs, outputs, checks, and hashes needed to rerun every scientific result included in the verdict.
   - completion: Raw logs, IDs, prompts, and detailed provenance remain in the machine package rather than the researcher report.
   - completion: Every concrete manuscript recommendation from the verified recommendations resource appears in the canonical audit with its affected section or statement, exact change, evidence, urgency, and required or optional status.
   - completion: The package is internal-only unless audit-request contains separate explicit external-release approval.
   - execution: inherit
2. **Render concise researcher report**: Render a two-to-three-page HTML report from the canonical JSON using the pinned template.
   - id: render-researcher-html
   - agent: peer2paper-delivery/audit-report-builder
   - routine-input: audit-template-resource
   - input-artefact: audit-json
   - output-artefact: audit-html
   - completion: The HTML presents the verdict, reported versus reproduced result, sensitivity summary, verified contradictions or ambiguities, exact manuscript changes, and remaining limitations.
   - completion: Every number and allowed statement is read from the canonical JSON, and detailed provenance is linked rather than expanded into large report tables.
   - completion: The HTML is accessible, readable, and contains no unsupported scientific finding.
   - execution: inherit
3. **Check and package audit**: Check cross-format identity, replay evidence, completeness, and internal release boundaries, then write the final manifest. Run licence and retention checks only when the audit request authorizes public redistribution.
   - id: check-and-package-audit
   - agent: peer2paper-delivery/release-quality-reviewer
   - routine-input: audit-request
   - routine-input: execution-policy
   - routine-input: audit-schema-resource
   - input-artefact: audit-json
   - input-artefact: audit-html
   - input-artefact: replay-package
   - output-artefact: release-gates
   - output-artefact: audit-package
   - completion: Every displayed number and verdict statement matches the canonical JSON, and each scientific result has replayable evidence.
   - completion: Release checks cover completeness, JSON-to-HTML consistency, provenance, replay, accessibility, and internal access boundaries.
   - completion: External release is disabled unless audit-request contains separate explicit approval. Full licence and retention adjudication runs only with that approval; otherwise the manifest records internal-only scope.
   - completion: The manifest reports pass, limited, or blocking for each required check and never lists PDF as a required core output.
   - completion: Emit audit_completed only with no blocking or limited gate; emit audit_completed_with_limits only with no blocking gate and at least one visible limitation.
   - outcome: audit_completed | terminal | The audit and replay package pass every required scoped check.
   - outcome: audit_completed_with_limits | terminal | The audit is releasable with explicit nonblocking limitations.
   - outcome: operationally_blocked | terminal | A required completeness, replay, authority, or scoped release check blocks packaging.
   - execution: inherit

## Transitions
| ID | From Step | To Step | Condition | Value | Max Traversals |
|----|-----------|---------|-----------|-------|----------------|
| canonical-to-html | build-canonical-audit | render-researcher-html | on-success | - | - |
| html-to-package | render-researcher-html | check-and-package-audit | on-success | - | - |

## Handoffs
| ID | From Step | To Step | Artefacts | Mode | Acceptance | Legacy Soft Budget | Retries | Backoff | Model |
|----|-----------|---------|-----------|------|------------|---------|---------|---------|-------|
| html-to-package-handoff | render-researcher-html | check-and-package-audit | audit-html; project://peer2paper/audits/{runId}/delivery/audit.html | await-result | The HTML is rendered only from the canonical JSON and identifies the same workflow run and content hash.; The HTML contains the verdict, reproduced result, sensitivity summary, verified conflicts, manuscript recommendations, and limitations required for release review. | - | - | - | - |

## Artefacts
| ID | Artifact | Path | Format |
|----|----------|------|--------|
| audit-json | Canonical audit JSON | project://peer2paper/audits/{runId}/delivery/audit.json | json |
| replay-package | Replay package | project://peer2paper/audits/{runId}/delivery/replay | directory |
| audit-html | Researcher audit report HTML | project://peer2paper/audits/{runId}/delivery/audit.html | html |
| release-gates | Scoped release checks | project://peer2paper/audits/{runId}/delivery/release-gates.json | json |
| audit-package | Audit package manifest | project://peer2paper/audits/{runId}/delivery/audit-package.json | json |
