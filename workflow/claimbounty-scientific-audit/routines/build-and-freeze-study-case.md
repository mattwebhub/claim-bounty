# ROUTINE: Build And Freeze Study Case

> **Status**: Production | **Updated**: 2026-08-31 | **Scope**: Public ClaimBounty workflow export

## Metadata

- **Department:** claim-bounty-operations
- **Agent:** ClaimBounty Orchestrator
- **Cadence:** OnDemand
- **Type:** sub-routine
- **Parent:** claim-bounty-operations/run-claimbounty-scientific-audit
- **Max Concurrency:** 3
- **Status:** Production

## Purpose

Freeze one caller-supplied quantitative claim into a compact study case containing the mandatory primary target, exact source-backed analysis contract, target dependency closure, bounded secondary benchmarks, and separately preserved document conflicts, without executing a scientific model during intake.

## Inputs

| ID                | Kind      | Source | Binding | Required | Type      | Description                                                                                                                                                                                                          |
| ----------------- | --------- | ------ | ------- | -------- | --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| case-bundle       | parameter | Parent | -       | dispatch | directory | Submitted case directory.                                                                                                                                                                                            |
| audit-request     | parameter | Parent | -       | dispatch | json      | Purpose, target claim, permissions, privacy, retention, and search authorization.                                                                                                                                    |
| scientific-policy | parameter | Parent | -       | dispatch | json      | Required scientific rules for estimands, tolerances, admissibility, comparability, severity, disagreement, and allowed defaults. Supply an empty JSON object only when the routine's recorded defaults are intended. |
| execution-policy  | parameter | Parent | -       | dispatch | json      | Execution, storage, network, and budget rules.                                                                                                                                                                       |

## Workflow

1. **Validate target claim and map relevant documents**: Validate the caller-supplied target claim and map only the primary paper, supplement, preregistration, and passages needed to identify its reported result and wording.
   - id: validate-claim-map-documents
   - agent: claim-bounty-intake/claim-mapper
   - routine-input: case-bundle
   - routine-input: audit-request
   - routine-input: scientific-policy
   - output-artefact: paper-documents
   - completion: One target claim is fixed with exact text and location, population, exposure or intervention, comparator, outcome, timepoint, observation unit, estimand, directional orientation, conclusion-change rules, and known ambiguities.
   - completion: The primary reported estimate, interval, p-value, sample size, and exact table, figure, or code target are recorded when present.
   - completion: Only the primary paper, supplement, preregistration, and claim-relevant passages are mapped; unreadable or missing required material is explicit.
   - completion: Record only values reported in authorized documents and their exact locations. Do not execute supplied analysis code, fit a model, or compute a diagnostic estimate, interval, statistic, p-value, or model sample size.
   - execution: inherit
2. **Map target data and variables**: Map only columns in the primary target dependency closure, including sample filters and variables required to derive those columns. Do not profile unrelated columns or search for participant keys unless target execution requires participant identity.
   - id: map-target-data
   - agent: claim-bounty-intake/variable-mapper
   - routine-input: case-bundle
   - routine-input: scientific-policy
   - input-artefact: paper-documents
   - output-artefact: dataset-maps
   - completion: Every primary-target element and bounded secondary benchmark has a direct data-field, derivation, sample-filter, or documented missing-link mapping within the target dependency closure.
   - completion: The map records units, condition codes, derivations, target-relevant missingness, observation unit, and unresolved high-impact ambiguity without profiling unrelated columns.
   - completion: Single-column uniqueness scans and combination-key searches are prohibited unless participant identity is required by the frozen target; any permitted identity check is bounded to the minimum declared columns.
   - completion: The checkpoint reports counts and stable lists of mapped columns, inspected columns, and intentionally excluded columns, with inspected columns limited to the target dependency closure.
   - completion: Map fields, filters, derivations, and target-relevant missingness without fitting a statistical model or calculating the target result.
   - execution: inherit
3. **Map target code and dependencies**: Identify the untouched entrypoint, target-producing code, exact formula, sample filter, transformations, factor levels, coefficient and orientation, dependencies, paths, inputs, outputs, seeds, and no more than three secondary benchmarks.
   - id: map-target-code
   - agent: claim-bounty-reproduction/reproduction-engineer
   - routine-input: case-bundle
   - routine-input: execution-policy
   - input-artefact: paper-documents
   - output-artefact: code-project
   - completion: The primary command and one unique target-producing code path are identified, or the exact missing or ambiguous entrypoint is recorded.
   - completion: The code map records source file identities and hashes plus exact source lines and literal expressions for the formula, population filter, transformations, factor levels, reference group, coefficient, orientation, and missing-data declaration.
   - completion: Expressions present in source are copied exactly into the code map and are not reconstructed from prose or line ranges.
   - completion: A dependency manifest records languages, lockfiles, imports, packages, system requirements, paths, mounts, permissions, network needs, seeds, and missing files.
   - completion: The mandatory primary target is separate from no more than three secondary benchmarks.
   - completion: This step is static code and dependency inspection. Parse-only validation is permitted, but evaluating scientific source, fitting a model, or generating result values is prohibited.
   - execution: inherit
4. **Verify supplied-document consistency**: Compare the primary paper, supplement, preregistration, and completed code map only where they define the target population, variables, condition codes, analysis, or reported result. Record exact conflicts and manuscript consequences; do not remap unrelated content.
   - id: verify-document-consistency
   - agent: claim-bounty-verification/source-evidence-verifier
   - routine-input: case-bundle
   - routine-input: audit-request
   - input-artefact: paper-documents
   - input-artefact: code-project
   - output-artefact: document-consistency-findings
   - completion: Every retained conflict links exact passages or code locations, identifies the affected claim or execution element, and is verified or marked unresolved.
   - completion: The check is limited to the target paper, supplement, preregistration, and completed code map.
   - completion: Compare supplied document passages and static code locations only. Do not execute the analysis or use a newly calculated model result as document-consistency evidence.
   - execution: inherit
5. **Bind and freeze analysis**: Join the claim, target-scoped data map, exact source-backed code map, and verified document findings into one frozen study case with a compact analysis_contract section.
   - id: bind-and-freeze-analysis
   - agent: claim-bounty-intake/claim-mapper
   - input-artefact: paper-documents
   - input-artefact: dataset-maps
   - input-artefact: code-project
   - input-artefact: document-consistency-findings
   - output-artefact: frozen-study-case
   - completion: primary_target is mandatory on every frozen case and remains separate from the bounded secondary_benchmarks array.
   - completion: analysis_contract contains code_hash, data_hash, formula, population_filter, exact transformations, factor levels and reference group, coefficient, orientation, missing-data declaration, required columns, entrypoint, and dependency identity copied from the code and data maps.
   - completion: document_conflicts remains a separate collection linked to exact supplied passages or code locations and does not overwrite the primary target or analysis contract.
   - completion: When the caller-frozen estimand maps to one unique supplied code path and the audit request, scientific policy, primary evidence, and code agree, a contradictory supplementary document emits case_frozen_with_limits without stopping reproduction.
   - completion: Emit case_frozen only when the unique analysis contract has no nonblocking conflict; emit case_not_assessable only when no unique target exists or choosing a code path would change the caller-frozen estimand; emit operationally_blocked only for authority, safety, privacy, or inaccessible required files.
   - completion: New intake artefacts must not contain source_consistency_probe or any model-derived diagnostic result. Scientific execution evidence begins only in Reproduce Original Result through the official runner.
   - outcome: case_frozen | terminal | The caller-frozen estimand maps to one schema-valid executable target without a blocking gap or source conflict.
   - outcome: case_frozen_with_limits | terminal | The executable target is schema-valid and may run, with a visible nonblocking supplied-document conflict or other declared limitation.
   - outcome: case_not_assessable | terminal | No unique executable target exists, or choosing a supplied code path would change the caller-frozen estimand.
   - outcome: operationally_blocked | terminal | Authority, privacy, safety, or inaccessible required files block execution.
   - execution: inherit

## Transitions

| ID                     | From Step                    | To Step                     | Condition  | Value | Max Traversals |
| ---------------------- | ---------------------------- | --------------------------- | ---------- | ----- | -------------- |
| claim-to-data          | validate-claim-map-documents | map-target-data             | on-success | -     | -              |
| claim-to-code          | validate-claim-map-documents | map-target-code             | on-success | -     | -              |
| data-to-contract       | map-target-data              | bind-and-freeze-analysis    | on-success | -     | -              |
| documents-to-contract  | verify-document-consistency  | bind-and-freeze-analysis    | on-success | -     | -              |
| code-to-document-check | map-target-code              | verify-document-consistency | on-success | -     | -              |

## Handoffs

| ID                             | From Step                    | To Step                     | Artefacts                                                                                                         | Mode         | Acceptance                                                                                                                                                                                                              | Legacy Soft Budget | Retries | Backoff | Model |
| ------------------------------ | ---------------------------- | --------------------------- | ----------------------------------------------------------------------------------------------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ | ------- | ------- | ----- |
| claim-to-data-handoff          | validate-claim-map-documents | map-target-data             | paper-documents; project://claimbounty/audits/{runId}/study-case/paper-documents.json                             | await-result | The frozen claim and relevant passages have stable source locations and explicit ambiguity flags.                                                                                                                       | -                  | -       | -       | -     |
| claim-to-code-handoff          | validate-claim-map-documents | map-target-code             | paper-documents; project://claimbounty/audits/{runId}/study-case/paper-documents.json                             | await-result | The primary reported target and its exact document location are fixed for code tracing.                                                                                                                                 | -                  | -       | -       | -     |
| data-to-contract-handoff       | map-target-data              | bind-and-freeze-analysis    | dataset-maps; project://claimbounty/audits/{runId}/study-case/dataset-maps.json                                   | await-result | The data map is confined to the primary-target dependency closure, records required filters and derivations, and reports mapped, inspected, and intentionally excluded column counts without an unrelated key search.   | -                  | -       | -       | -     |
| documents-to-contract-handoff  | verify-document-consistency  | bind-and-freeze-analysis    | document-consistency-findings; project://claimbounty/audits/{runId}/study-case/document-consistency-findings.json | await-result | Every retained conflict has exact supplied-document evidence, verification status, an affected target element, and a finding ID that can remain visible in both frozen outputs.                                         | -                  | -       | -       | -     |
| code-to-document-check-handoff | map-target-code              | verify-document-consistency | code-project; project://claimbounty/audits/{runId}/study-case/code-project.json                                   | await-result | The completed code map identifies the unique target entrypoint, dependencies, data bindings, variable definitions, and source-backed expression registry needed for the bounded document check and executable contract. | -                  | -       | -       | -     |

## Artefacts

| ID                            | Artifact                                        | Path                                                                               | Format |
| ----------------------------- | ----------------------------------------------- | ---------------------------------------------------------------------------------- | ------ |
| paper-documents               | Target claim and relevant document map          | project://claimbounty/audits/{runId}/study-case/paper-documents.json               | json   |
| dataset-maps                  | Target data and variable map                    | project://claimbounty/audits/{runId}/study-case/dataset-maps.json                  | json   |
| code-project                  | Target code and dependency map                  | project://claimbounty/audits/{runId}/study-case/code-project.json                  | json   |
| document-consistency-findings | Verified supplied-document consistency findings | project://claimbounty/audits/{runId}/study-case/document-consistency-findings.json | json   |
| frozen-study-case             | Compact frozen study case contract              | project://claimbounty/audits/{runId}/study-case/frozen-study-case.json             | json   |
