# ROUTINE: Research Methods And Evidence

> **Status**: Production | **Updated**: 2026-08-29

## Metadata
- **Routine Schema:** 2
- **Department:** peer2paper-operations
- **Agent:** Peer2Paper Orchestrator
- **Cadence:** OnDemand
- **Type:** sub-routine
- **Parent:** peer2paper-operations/run-peer2paper-scientific-audit
- **Max Concurrency:** 3
- **Status:** Production

## Purpose
Answer no more than three targeted questions raised by the frozen claim, reproduction result, or supplied-document conflicts, deep-map no more than three authorized sources through checkpointed Silk-aware workers, and package exact passages and manuscript consequences. Sensitivity results may trigger a bounded follow-up only during final verification.

## Pre-Requisites
- Call `browser_silk_load` before opening each mapped literature site used by the active step.
- Verify that each source is open access or otherwise authorized by the active audit before opening full text.
- Reuse the mapped funnels below; none of these read-only literature funnels requires authentication.
- Saved Browser Silk funnels available to the three bounded source slots:
  - `pmc.ncbi.nlm.nih.gov` / `open_and_extract_article` requires `pmcid` and returns visible full text, headings, tables, figures, supplements, and references.
  - `arxiv.org` / `open_and_extract_full_text` requires `arxiv_id` and returns article text and structure.
  - `journals.sagepub.com` / `open_and_extract_article` requires `doi` and returns visible article text and structure.
  - `www.cambridge.org` / `open_and_extract_article` requires `article_slug` and returns visible article text and structure.
  - `rameliaz.github.io` / `open_pdf` requires `document_path`; the local PDF parser supplies page and text locations when browser text is unavailable.
  - `www.bmj.com` / `open_and_extract_article` requires `article_path` and returns visible article text and structure.
  - `www.nber.org` / `open_working_paper_pdf` requires `document_path`; the local PDF parser supplies page and text locations when browser text is unavailable.
  - `osf.io` / `project inventory` verifies project identity and authorized download records; registered local copies supply full text when available.

## Inputs
| ID | Kind | Source | Binding | Required | Type | Description |
|----|------|--------|---------|----------|------|-------------|
| frozen-study-case-resource | resource | Parent | project://peer2paper/subroutine-inputs/frozen-study-case.json | dispatch | file | Parent-produced frozen study case. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| reproduction-package-resource | resource | Parent | project://peer2paper/subroutine-inputs/reproduction-package.json | dispatch | file | Parent-produced reproduction package. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| audit-request | parameter | Parent | - | dispatch | json | Claim scope, permissions, dates, languages, user sources, and external-search authorization. |
| execution-policy | parameter | Parent | - | dispatch | json | Research budget, source access, network, storage, and licence policy. |
| scientific-policy | parameter | Parent | - | dispatch | json | Required scientific rules for estimands, tolerances, admissibility, comparability, severity, disagreement, and allowed defaults. Supply an empty JSON object only when the routine's recorded defaults are intended. |
| document-consistency-findings-resource | resource | Parent | project://peer2paper/subroutine-inputs/document-consistency-findings.json | dispatch | file | Parent-produced verified conflicts and unresolved inconsistencies from the primary paper, supplement, preregistration, and completed code map. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |

## Workflow
1. **Freeze targeted research questions**: Freeze no more than three evidence questions that arise from the frozen claim, reproduction comparison, or verified supplied-document conflicts. Do not start a broad literature review or require a completed sensitivity map.
   - id: freeze-research-brief
   - agent: research-and-insights/researcher
   - routine-input: frozen-study-case-resource
   - routine-input: reproduction-package-resource
   - routine-input: document-consistency-findings-resource
   - routine-input: audit-request
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - output-artefact: research-brief
   - completion: The brief contains no more than three questions, and each question names the finding it can confirm, contradict, or qualify.
   - completion: Each question has source classes, search concepts, inclusion and comparison rules, authorization limits, and a stopping rule.
   - completion: The initial brief does not depend on sensitivity results; any later sensitivity-triggered source check is deferred to final verification.
   - execution: inherit
2. **Search, screen, and assign source slots**: Use toone-scholar and authorized sources to answer the frozen questions, metadata-screen broadly, deduplicate versions, and assign no more than three deep-source slots. Use Toone's persistent browser and complete saved Silk flows for recurring interactive sources.
   - id: search-screen-slice-sources
   - agent: research-and-insights/literature-searcher
   - routine-input: audit-request
   - routine-input: execution-policy
   - input-artefact: research-brief
   - output-artefact: screened-source-registry
   - completion: Every query, source, timestamp, version, access status, screening decision, and deduplication key is recorded.
   - completion: No more than three deep-source slots are assigned; extra candidates remain metadata-only.
   - completion: Each slot names its authorized access route, exact evidence target, and mapped Silk funnel when applicable.
   - execution: inherit
3. **Map selected source 1**: Process only source slot 1. Load the mapped domain before navigation, replay the complete saved Silk funnel, and write one independent checkpoint; if no source is assigned, write not_assigned without browsing.
   - id: map-full-text-source-1
   - agent: research-and-insights/source-extractor
   - routine-input: execution-policy
   - input-artefact: research-brief
   - input-artefact: screened-source-registry
   - output-artefact: full-text-source-1
   - completion: Source slot 1 always produces one checkpoint with status mapped, limited, inaccessible, failed, or not_assigned.
   - completion: This step is one independent Source Extractor execution in the three-way fan-out. It processes only its assigned slot and never takes another slot's source.
   - completion: Mapped evidence includes source identity, version, exact passage and location, context, comparability, supported or contradicted statement, manuscript consequence, access status, and replay provenance.
   - completion: Read maximum_seconds_per_full_text from execution-policy. Complete any started Silk funnel, then converge immediately to the checkpoint without opening another source or beginning optional extraction work when the limit is reached.
   - completion: For selector_not_found, perform at most one in-attempt Silk heal, retry the affected flow step once, and push a healed map. If it still fails, preserve the last failure evidence in the checkpoint; never switch to manual browsing.
   - execution: inherit
4. **Map selected source 2**: Process only source slot 2. Load the mapped domain before navigation, replay the complete saved Silk funnel, and write one independent checkpoint; if no source is assigned, write not_assigned without browsing.
   - id: map-full-text-source-2
   - agent: research-and-insights/source-extractor
   - routine-input: execution-policy
   - input-artefact: research-brief
   - input-artefact: screened-source-registry
   - output-artefact: full-text-source-2
   - completion: Source slot 2 always produces one checkpoint with status mapped, limited, inaccessible, failed, or not_assigned.
   - completion: This step is one independent Source Extractor execution in the three-way fan-out. It processes only its assigned slot and never takes another slot's source.
   - completion: Mapped evidence includes source identity, version, exact passage and location, context, comparability, supported or contradicted statement, manuscript consequence, access status, and replay provenance.
   - completion: Read maximum_seconds_per_full_text from execution-policy. Complete any started Silk funnel, then converge immediately to the checkpoint without opening another source or beginning optional extraction work when the limit is reached.
   - completion: For selector_not_found, perform at most one in-attempt Silk heal, retry the affected flow step once, and push a healed map. If it still fails, preserve the last failure evidence in the checkpoint; never switch to manual browsing.
   - execution: inherit
5. **Map selected source 3**: Process only source slot 3. Load the mapped domain before navigation, replay the complete saved Silk funnel, and write one independent checkpoint; if no source is assigned, write not_assigned without browsing.
   - id: map-full-text-source-3
   - agent: research-and-insights/source-extractor
   - routine-input: execution-policy
   - input-artefact: research-brief
   - input-artefact: screened-source-registry
   - output-artefact: full-text-source-3
   - completion: Source slot 3 always produces one checkpoint with status mapped, limited, inaccessible, failed, or not_assigned.
   - completion: This step is one independent Source Extractor execution in the three-way fan-out. It processes only its assigned slot and never takes another slot's source.
   - completion: Mapped evidence includes source identity, version, exact passage and location, context, comparability, supported or contradicted statement, manuscript consequence, access status, and replay provenance.
   - completion: Read maximum_seconds_per_full_text from execution-policy. Complete any started Silk funnel, then converge immediately to the checkpoint without opening another source or beginning optional extraction work when the limit is reached.
   - completion: For selector_not_found, perform at most one in-attempt Silk heal, retry the affected flow step once, and push a healed map. If it still fails, preserve the last failure evidence in the checkpoint; never switch to manual browsing.
   - execution: inherit
6. **Assemble targeted evidence**: Join the three source checkpoints into a targeted evidence package, preserving metadata-only candidates, access limits, exact passages, comparability judgments, and concrete manuscript consequences.
   - id: assemble-targeted-evidence
   - agent: research-and-insights/researcher
   - input-artefact: research-brief
   - input-artefact: screened-source-registry
   - input-artefact: full-text-source-1
   - input-artefact: full-text-source-2
   - input-artefact: full-text-source-3
   - output-artefact: full-text-map
   - output-artefact: literature-evidence-package
   - output-artefact: literature-version-history
   - completion: Every retained evidence item has an exact passage and location, the statement it supports or contradicts, a comparability assessment, verification state, and proposed manuscript consequence.
   - completion: The package joins all three independent source checkpoints, including limited, inaccessible, failed, and not_assigned slots, without serializing or rerunning a sibling slot.
   - completion: The package answers no more than three questions, contains no more than three deeply processed sources, distinguishes inaccessible and unassigned slots, and exposes remaining gaps.
   - completion: Evidence used in a verdict is marked for independent reopening during verification; a second complete extraction review is not run here.
   - completion: Write an immutable run-scoped evidence snapshot alongside the selected per-run evidence package; no prior-run replacement read is required.
   - outcome: evidence_ready | terminal | Required targeted evidence is available and supported without residual gaps.
   - outcome: evidence_ready_with_limits | terminal | The targeted search completed with visible nonblocking gaps.
   - outcome: evidence_not_assessable | terminal | A required evidence question cannot be answered from authorized accessible sources.
   - outcome: operationally_blocked | terminal | Authority or access policy prevents required research.
   - execution: inherit

## Transitions
| ID | From Step | To Step | Condition | Value | Max Traversals |
|----|-----------|---------|-----------|-------|----------------|
| brief-to-search | freeze-research-brief | search-screen-slice-sources | on-success | - | - |
| search-to-source-1 | search-screen-slice-sources | map-full-text-source-1 | on-success | - | - |
| search-to-source-2 | search-screen-slice-sources | map-full-text-source-2 | on-success | - | - |
| search-to-source-3 | search-screen-slice-sources | map-full-text-source-3 | on-success | - | - |
| source-1-to-assemble | map-full-text-source-1 | assemble-targeted-evidence | on-success | - | - |
| source-2-to-assemble | map-full-text-source-2 | assemble-targeted-evidence | on-success | - | - |
| source-3-to-assemble | map-full-text-source-3 | assemble-targeted-evidence | on-success | - | - |

## Handoffs
| ID | From Step | To Step | Artefacts | Mode | Acceptance | Legacy Soft Budget | Retries | Backoff | Model |
|----|-----------|---------|-----------|------|------------|---------|---------|---------|-------|
| brief-to-search-handoff | freeze-research-brief | search-screen-slice-sources | research-brief; project://peer2paper/audits/{runId}/research/research-brief.json | await-result | The brief contains no more than three finding-triggered questions with source, access, comparison, and stopping rules. | - | - | - | - |
| search-to-source-1-handoff | search-screen-slice-sources | map-full-text-source-1 | screened-source-registry; project://peer2paper/audits/{runId}/research/screened-sources.json | await-result | Source slot 1 has one stable assignment or is explicitly unassigned, with its authorized access route and expected evidence target. | - | - | - | - |
| search-to-source-2-handoff | search-screen-slice-sources | map-full-text-source-2 | screened-source-registry; project://peer2paper/audits/{runId}/research/screened-sources.json | await-result | Source slot 2 has one stable assignment or is explicitly unassigned, with its authorized access route and expected evidence target. | - | - | - | - |
| search-to-source-3-handoff | search-screen-slice-sources | map-full-text-source-3 | screened-source-registry; project://peer2paper/audits/{runId}/research/screened-sources.json | await-result | Source slot 3 has one stable assignment or is explicitly unassigned, with its authorized access route and expected evidence target. | - | - | - | - |
| source-1-to-assemble-handoff | map-full-text-source-1 | assemble-targeted-evidence | full-text-source-1; project://peer2paper/audits/{runId}/research/source-checkpoints/source-01.json | await-result | Source slot 1 has a complete mapped, limited, inaccessible, failed, or not_assigned checkpoint with replay provenance. | - | - | - | - |
| source-2-to-assemble-handoff | map-full-text-source-2 | assemble-targeted-evidence | full-text-source-2; project://peer2paper/audits/{runId}/research/source-checkpoints/source-02.json | await-result | Source slot 2 has a complete mapped, limited, inaccessible, failed, or not_assigned checkpoint with replay provenance. | - | - | - | - |
| source-3-to-assemble-handoff | map-full-text-source-3 | assemble-targeted-evidence | full-text-source-3; project://peer2paper/audits/{runId}/research/source-checkpoints/source-03.json | await-result | Source slot 3 has a complete mapped, limited, inaccessible, failed, or not_assigned checkpoint with replay provenance. | - | - | - | - |

## Artefacts
| ID | Artifact | Path | Format |
|----|----------|------|--------|
| research-brief | Targeted research questions | project://peer2paper/audits/{runId}/research/research-brief.json | json |
| screened-source-registry | Screened source shortlist and slot plan | project://peer2paper/audits/{runId}/research/screened-sources.json | json |
| full-text-source-1 | Source evidence checkpoint 1 | project://peer2paper/audits/{runId}/research/source-checkpoints/source-01.json | json |
| full-text-source-2 | Source evidence checkpoint 2 | project://peer2paper/audits/{runId}/research/source-checkpoints/source-02.json | json |
| full-text-source-3 | Source evidence checkpoint 3 | project://peer2paper/audits/{runId}/research/source-checkpoints/source-03.json | json |
| full-text-map | Assembled targeted full-text evidence | project://peer2paper/audits/{runId}/research/full-text-map.json | json |
| literature-evidence-package | Targeted literature evidence package | project://peer2paper/audits/{runId}/research/literature-evidence-package.json | json |
| literature-version-history | Targeted evidence version history | project://peer2paper/audits/{runId}/research/versions | directory |
