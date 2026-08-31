# ROUTINE: Run Peer2Paper Scientific Audit

## Metadata
- **Routine Schema:** 2
- **Department:** peer2paper-operations
- **Agent:** Peer2Paper Orchestrator
- **Cadence:** OnDemand
- **Max Concurrency:** 3
- **Status:** Production

## Purpose
Audit one caller-supplied quantitative claim by freezing a compact case with an embedded analysis contract, reproducing that contract under bounded local execution, running authorized sensitivity and targeted evidence work, independently verifying decisive findings, and returning a lean replayable audit package.

## Inputs
| ID | Kind | Source | Binding | Required | Type | Description |
|----|------|--------|---------|----------|------|-------------|
| case-bundle | parameter | Caller | - | dispatch | directory | Local directory containing the paper or draft, data, code, environment files, supplements, data dictionaries, and authorized supporting sources. |
| audit-request | parameter | Caller | - | dispatch | json | Required audit scope containing one exact target claim and location, permissions, privacy classification, retention rules, public-redistribution intent, and external-search authorization. |
| scientific-policy | parameter | Caller | - | dispatch | json | Required scientific rules for estimands, tolerances, admissibility, comparability, severity, disagreement, and allowed defaults. Supply an empty JSON object only when the routine's recorded defaults are intended. |
| execution-policy | parameter | Caller | - | dispatch | json | Execution rules for runtime, attempt-local working directories, host network policy, operational memory and storage defaults, a 600-second command timeout, source immutability, source access, and internal-only release. |
| canonical-schema-bundle | resource | Definition | project://peer2paper/config/canonical-schemas.json | dispatch | file | Routine-owned versioned schemas for canonical findings, claims, actions, limitations, and frozen audit objects. |
| audit-template | resource | Definition | project://peer2paper/config/audit-template.html | dispatch | file | Project-owned pinned template used for the researcher-facing HTML report. |
| audit-schema | resource | Definition | project://peer2paper/config/audit-schema.json | dispatch | file | Routine-owned pinned JSON Schema for the machine-readable audit. |
| execution-runner | resource | Definition | project://peer2paper/execution | dispatch | directory | Project-owned versioned scientific execution runner used by reproduction, robustness, and independent statistical verification. Missing runner files block execution. |
| reproduction-continuation-decision | parameter | Caller | - | stage | json | Required only after partial_reproduction or numerical_mismatch. Supply a JSON decision of continue or stop, with a reason and the authorizing person or role. |

## Workflow
1. **Build compact study case**: Validate the target, map only its document and data dependency closure, preserve exact source-backed transformations and document conflicts, and freeze one study case containing primary_target and analysis_contract.
   - id: invoke-build-study-case
   - routine-input: case-bundle
   - routine-input: audit-request
   - routine-input: scientific-policy
   - routine-input: execution-policy
   - output-artefact: paper-documents
   - output-artefact: dataset-maps
   - output-artefact: code-project
   - output-artefact: document-consistency-findings
   - output-artefact: frozen-study-case
   - completion: The child returns one frozen study case with mandatory primary_target, compact analysis_contract, separate document_conflicts, bounded secondary benchmarks, and target-scoped data and code maps, or an explicit blocker.
   - completion: A unique caller-frozen estimand with a contradictory supplementary document emits case_frozen_with_limits and remains eligible for reproduction.
   - outcome: case_frozen | The claim and every required execution link are fixed without a blocking gap.
   - outcome: case_frozen_with_limits | The case can execute with visible nonblocking limitations.
   - outcome: case_not_assessable | terminal | The target cannot be fixed without an unsupported substantive choice.
   - outcome: operationally_blocked | terminal | Authority, privacy, safety, or inaccessible required files block execution.
   - sub-routine: peer2paper-operations/build-and-freeze-study-case
   - child-input: case-bundle | parent-input | case-bundle
   - child-input: audit-request | parent-input | audit-request
   - child-input: scientific-policy | parent-input | scientific-policy
   - child-input: execution-policy | parent-input | execution-policy
   - child-output: paper-documents | paper-documents | case_frozen;case_frozen_with_limits;case_not_assessable;operationally_blocked
   - child-output: dataset-maps | dataset-maps | case_frozen;case_frozen_with_limits;case_not_assessable;operationally_blocked
   - child-output: code-project | code-project | case_frozen;case_frozen_with_limits;case_not_assessable;operationally_blocked
   - child-output: document-consistency-findings | document-consistency-findings | case_frozen;case_frozen_with_limits;case_not_assessable;operationally_blocked
   - child-output: frozen-study-case | frozen-study-case | case_frozen;case_frozen_with_limits;case_not_assessable;operationally_blocked
   - execution: inherit
2. **Preflight and reproduce primary result**: Preflight the real case bundle and shared execution capability, run the contract-defined model unchanged, test bounded working-directory or input-binding repairs when needed, and preserve complete scientific outputs in one reproduction package.
   - id: invoke-reproduce-result
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: execution-runner
   - routine-input: case-bundle
   - input-artefact: frozen-study-case
   - output-artefact: reproduction-inputs
   - output-artefact: execution-readiness
   - output-artefact: execution-profile
   - output-artefact: primary-run
   - output-artefact: primary-comparison
   - output-artefact: repair-candidates
   - output-artefact: repair-attempt-1
   - output-artefact: repair-attempt-2
   - output-artefact: repair-attempt-3
   - output-artefact: secondary-target-1
   - output-artefact: secondary-target-2
   - output-artefact: secondary-target-3
   - output-artefact: reproduction-evidence
   - output-artefact: reproduction-package
   - completion: The child receives the real case bundle and frozen analysis_contract, captures sample, missingness, coefficient, standard error, interval, statistic, degrees of freedom and p-value, and returns a self-contained reproduction package.
   - completion: Execution provenance and scientific comparison remain separate: execution_mode is untouched or technically_repaired_original, while the emitted routine outcome follows comparison_status.
   - completion: Verified network or DNS denial is not a prerequisite. Memory and storage are operational defaults, and unavailable exact enforcement does not emit not_executable when the analysis can run in an attempt-local working directory.
   - outcome: exact_reproduction | The untouched primary target and required benchmarks reproduce exactly under frozen comparison rules.
   - outcome: within_tolerance | The untouched primary target and required benchmarks satisfy the frozen tolerance rules.
   - outcome: partial_reproduction | The primary target or required benchmarks reproduce only in part and need an explicit continuation decision.
   - outcome: numerical_mismatch | The executable analysis remains numerically inconsistent after the one remediation pass.
   - outcome: not_executable | terminal | A concrete runtime, dependency, path, permission, isolation, or resource-control blocker prevents execution.
   - outcome: not_assessable | terminal | The target cannot be assessed without an unsupported substantive choice.
   - sub-routine: peer2paper-operations/reproduce-original-result
   - child-input: execution-policy | parent-input | execution-policy
   - child-input: scientific-policy | parent-input | scientific-policy
   - child-input: execution-runner | parent-input | execution-runner
   - child-input: frozen-study-case-resource | parent-artefact | frozen-study-case
   - child-input: case-bundle | parent-input | case-bundle
   - child-output: reproduction-inputs | reproduction-inputs | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: execution-readiness | execution-readiness | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: execution-profile | execution-profile | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: primary-run | primary-run | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: primary-comparison | primary-comparison | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: repair-candidates | repair-candidates | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: repair-attempt-1 | repair-attempt-1 | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: repair-attempt-2 | repair-attempt-2 | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: repair-attempt-3 | repair-attempt-3 | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: secondary-target-1 | secondary-target-1 | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: secondary-target-2 | secondary-target-2 | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: secondary-target-3 | secondary-target-3 | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: reproduction-evidence | reproduction-evidence | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - child-output: reproduction-package | reproduction-package | exact_reproduction;within_tolerance;partial_reproduction;numerical_mismatch;not_executable;not_assessable
   - execution: inherit
3. **Decide whether to continue after reproduction**: Review a partial reproduction or unresolved numerical mismatch and apply the caller's explicit authorization boundary before expensive downstream analysis.
   - id: decide-reproduction-continuation
   - agent: peer2paper-operations/peer2paper-orchestrator
   - routine-input: reproduction-continuation-decision
   - input-artefact: reproduction-package
   - completion: The supplied decision is continue or stop, names the reason and authorizing person or role, and applies to the current reproduction package.
   - completion: continue_to_analysis is emitted only with explicit authorization to interpret the partial reproduction or numerical mismatch; otherwise stop_after_reproduction is emitted.
   - outcome: continue_to_analysis | The caller authorizes robustness and targeted research despite the recorded reproduction limitation.
   - outcome: stop_after_reproduction | terminal | The run ends with the case contract, document-consistency findings, reproduction package, and diagnostic evidence.
   - execution: inherit
4. **Test bounded sensitivity candidates**: Run three result-blind design lanes, rank no more than five candidates, always materialize the conditional second-review artifact, and execute approved analyses through the ready reproduction profile.
   - id: invoke-stress-test
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: execution-runner
   - routine-input: case-bundle
   - input-artefact: frozen-study-case
   - input-artefact: reproduction-package
   - input-artefact: execution-profile
   - output-artefact: estimand-rules
   - output-artefact: data-measurement-candidates
   - output-artefact: model-specification-candidates
   - output-artefact: inference-candidates
   - output-artefact: ranked-candidates
   - output-artefact: blind-review-alpha
   - output-artefact: blind-review-beta
   - output-artefact: approved-candidates
   - output-artefact: candidate-runs
   - output-artefact: robustness-map
   - output-artefact: robustness-version-history
   - completion: No more than five admissible candidates produce a common-schema sensitivity result, with untested dimensions and execution limits visible.
   - outcome: robustness_ready | Required sensitivity checks completed without residual gaps.
   - outcome: robustness_ready_with_limits | Sensitivity checks completed with visible nonblocking gaps.
   - outcome: analysis_not_assessable | A blocking scientific or executable gap prevents a defensible sensitivity conclusion.
   - outcome: operationally_blocked | terminal | Authority or unsafe execution prevents required work.
   - outcome: sensitivity_blocked | terminal | Sensitivity preparation found a scientific, source/data, or execution-capability blocker.
   - sub-routine: peer2paper-operations/stress-test-analysis
   - child-input: execution-policy | parent-input | execution-policy
   - child-input: scientific-policy | parent-input | scientific-policy
   - child-input: execution-runner | parent-input | execution-runner
   - child-input: frozen-study-case-resource | parent-artefact | frozen-study-case
   - child-input: reproduction-package-resource | parent-artefact | reproduction-package
   - child-input: execution-profile-resource | parent-artefact | execution-profile
   - child-input: case-bundle | parent-input | case-bundle
   - child-output: estimand-rules | estimand-rules | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: data-measurement-candidates | data-measurement-candidates | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: model-specification-candidates | model-specification-candidates | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: inference-candidates | inference-candidates | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: ranked-candidates | ranked-candidates | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: blind-review-alpha | blind-review-alpha | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: blind-review-beta | blind-review-beta | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: approved-candidates | approved-candidates | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: candidate-runs | candidate-runs | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: robustness-map | robustness-map | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - child-output: robustness-version-history | robustness-version-history | robustness_ready;robustness_ready_with_limits;analysis_not_assessable;operationally_blocked
   - execution: inherit
5. **Answer targeted evidence questions**: Run beside sensitivity work after reproduction succeeds or is explicitly continued. Answer no more than three questions raised by the claim, reproduction, or supplied-document conflicts, with no more than three deep-source checkpoints.
   - id: invoke-research-evidence
   - routine-input: audit-request
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - input-artefact: frozen-study-case
   - input-artefact: reproduction-package
   - input-artefact: document-consistency-findings
   - output-artefact: research-brief
   - output-artefact: screened-source-registry
   - output-artefact: full-text-source-1
   - output-artefact: full-text-source-2
   - output-artefact: full-text-source-3
   - output-artefact: full-text-map
   - output-artefact: literature-evidence-package
   - output-artefact: literature-version-history
   - completion: The child returns exact passages, comparability judgments, verification state, and manuscript consequences for no more than three targeted questions and three deep-mapped sources.
   - outcome: evidence_ready | Required targeted evidence is available and supported without residual gaps.
   - outcome: evidence_ready_with_limits | The targeted search completed with visible nonblocking gaps.
   - outcome: evidence_not_assessable | A required evidence question cannot be answered from authorized accessible sources.
   - outcome: operationally_blocked | terminal | Authority or access policy prevents required research.
   - sub-routine: peer2paper-operations/research-methods-and-evidence
   - child-input: audit-request | parent-input | audit-request
   - child-input: execution-policy | parent-input | execution-policy
   - child-input: scientific-policy | parent-input | scientific-policy
   - child-input: frozen-study-case-resource | parent-artefact | frozen-study-case
   - child-input: reproduction-package-resource | parent-artefact | reproduction-package
   - child-input: document-consistency-findings-resource | parent-artefact | document-consistency-findings
   - child-output: research-brief | research-brief | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - child-output: screened-source-registry | screened-source-registry | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - child-output: full-text-source-1 | full-text-source-1 | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - child-output: full-text-source-2 | full-text-source-2 | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - child-output: full-text-source-3 | full-text-source-3 | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - child-output: full-text-map | full-text-map | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - child-output: literature-evidence-package | literature-evidence-package | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - child-output: literature-version-history | literature-version-history | evidence_ready;evidence_ready_with_limits;evidence_not_assessable;operationally_blocked
   - execution: inherit
6. **Verify decisive findings**: Wait for sensitivity and targeted research, then independently rerun the frozen analysis contract from the real case bundle in a clean workspace with equivalent isolation and resource controls, reopen verdict sources, and produce adjudicated findings.
   - id: invoke-verify-findings
   - routine-input: audit-request
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: execution-runner
   - routine-input: case-bundle
   - input-artefact: frozen-study-case
   - input-artefact: reproduction-package
   - input-artefact: robustness-map
   - input-artefact: literature-evidence-package
   - input-artefact: execution-profile
   - output-artefact: verification-inputs
   - output-artefact: statistical-verification
   - output-artefact: source-verification
   - output-artefact: adjudication-package
   - output-artefact: manuscript-recommendations
   - completion: The child uses the same frozen analysis contract and source and data hashes, captures complete required numerical comparisons in a clean attempt-local workspace, and returns allowed wording plus concrete manuscript consequences.
   - completion: Different runner or mount identities do not fail verification when the scientific contract, source and data hashes, dependencies, source immutability, timeout, and recorded environment are equivalent for the claim.
   - completion: External release stays disabled unless separately approved.
   - outcome: audit_ready | All required findings are independently verified and ready for reporting.
   - outcome: verification_incomplete | The audit can report verified work with explicit unresolved limitations.
   - outcome: operationally_blocked | terminal | Required verification cannot proceed safely or with authority.
   - sub-routine: peer2paper-operations/verify-and-adjudicate-findings
   - child-input: audit-request | parent-input | audit-request
   - child-input: execution-policy | parent-input | execution-policy
   - child-input: scientific-policy | parent-input | scientific-policy
   - child-input: execution-runner | parent-input | execution-runner
   - child-input: frozen-study-case-resource | parent-artefact | frozen-study-case
   - child-input: reproduction-package-resource | parent-artefact | reproduction-package
   - child-input: robustness-map-resource | parent-artefact | robustness-map
   - child-input: literature-evidence-resource | parent-artefact | literature-evidence-package
   - child-input: execution-profile-resource | parent-artefact | execution-profile
   - child-input: case-bundle | parent-input | case-bundle
   - child-output: verification-inputs | verification-inputs | audit_ready;verification_incomplete;operationally_blocked
   - child-output: statistical-verification | statistical-verification | audit_ready;verification_incomplete;operationally_blocked
   - child-output: source-verification | source-verification | audit_ready;verification_incomplete;operationally_blocked
   - child-output: adjudication-package | adjudication-package | audit_ready;verification_incomplete;operationally_blocked
   - child-output: manuscript-recommendations | manuscript-recommendations | audit_ready;verification_incomplete;operationally_blocked
   - execution: inherit
7. **Publish JSON-first audit package**: Build one canonical JSON audit from the verified findings and complete manuscript recommendations, render concise HTML from that JSON, and package replay evidence and scoped release checks.
   - id: invoke-deliver-audit
   - routine-input: audit-request
   - routine-input: execution-policy
   - routine-input: canonical-schema-bundle
   - routine-input: audit-template
   - routine-input: audit-schema
   - input-artefact: frozen-study-case
   - input-artefact: reproduction-package
   - input-artefact: robustness-map
   - input-artefact: literature-evidence-package
   - input-artefact: adjudication-package
   - input-artefact: manuscript-recommendations
   - output-artefact: audit-json
   - output-artefact: replay-package
   - output-artefact: audit-html
   - output-artefact: release-gates
   - output-artefact: audit-package
   - completion: One canonical JSON audit, concise HTML report, replay package, and package manifest agree on all numbers, findings, recommendations, and limitations.
   - outcome: audit_completed | terminal | The audit and replay package pass every required scoped check.
   - outcome: audit_completed_with_limits | terminal | The audit is releasable with explicit nonblocking limitations.
   - outcome: operationally_blocked | terminal | A required completeness, replay, authority, or scoped release check blocks packaging.
   - sub-routine: peer2paper-operations/assemble-final-audit
   - child-input: audit-request | parent-input | audit-request
   - child-input: execution-policy | parent-input | execution-policy
   - child-input: canonical-schema-bundle-resource | parent-input | canonical-schema-bundle
   - child-input: audit-template-resource | parent-input | audit-template
   - child-input: audit-schema-resource | parent-input | audit-schema
   - child-input: frozen-study-case-resource | parent-artefact | frozen-study-case
   - child-input: reproduction-package-resource | parent-artefact | reproduction-package
   - child-input: robustness-map-resource | parent-artefact | robustness-map
   - child-input: literature-evidence-resource | parent-artefact | literature-evidence-package
   - child-input: adjudication-package-resource | parent-artefact | adjudication-package
   - child-input: manuscript-recommendations-resource | parent-artefact | manuscript-recommendations
   - child-output: audit-json | audit-json | audit_completed;audit_completed_with_limits;operationally_blocked
   - child-output: replay-package | replay-package | audit_completed;audit_completed_with_limits;operationally_blocked
   - child-output: audit-html | audit-html | audit_completed;audit_completed_with_limits;operationally_blocked
   - child-output: release-gates | release-gates | audit_completed;audit_completed_with_limits;operationally_blocked
   - child-output: audit-package | audit-package | audit_completed;audit_completed_with_limits;operationally_blocked
   - execution: inherit

## Transitions
| ID | From Step | To Step | Condition | Value | Max Traversals |
|----|-----------|---------|-----------|-------|----------------|
| case-frozen-to-reproduction | invoke-build-study-case | invoke-reproduce-result | outcome-is | case_frozen | - |
| case-limited-to-reproduction | invoke-build-study-case | invoke-reproduce-result | outcome-is | case_frozen_with_limits | - |
| reproduction-exact-to-sensitivity | invoke-reproduce-result | invoke-stress-test | outcome-is | exact_reproduction | - |
| reproduction-exact-to-research | invoke-reproduce-result | invoke-research-evidence | outcome-is | exact_reproduction | - |
| reproduction-tolerance-to-sensitivity | invoke-reproduce-result | invoke-stress-test | outcome-is | within_tolerance | - |
| reproduction-tolerance-to-research | invoke-reproduce-result | invoke-research-evidence | outcome-is | within_tolerance | - |
| reproduction-partial-to-decision | invoke-reproduce-result | decide-reproduction-continuation | outcome-is | partial_reproduction | - |
| reproduction-mismatch-to-decision | invoke-reproduce-result | decide-reproduction-continuation | outcome-is | numerical_mismatch | - |
| decision-continue-to-sensitivity | decide-reproduction-continuation | invoke-stress-test | outcome-is | continue_to_analysis | - |
| decision-continue-to-research | decide-reproduction-continuation | invoke-research-evidence | outcome-is | continue_to_analysis | - |
| sensitivity-ready-to-verification | invoke-stress-test | invoke-verify-findings | outcome-is | robustness_ready | - |
| sensitivity-limited-to-verification | invoke-stress-test | invoke-verify-findings | outcome-is | robustness_ready_with_limits | - |
| sensitivity-unassessable-to-verification | invoke-stress-test | invoke-verify-findings | outcome-is | analysis_not_assessable | - |
| research-ready-to-verification | invoke-research-evidence | invoke-verify-findings | outcome-is | evidence_ready | - |
| research-limited-to-verification | invoke-research-evidence | invoke-verify-findings | outcome-is | evidence_ready_with_limits | - |
| research-unassessable-to-verification | invoke-research-evidence | invoke-verify-findings | outcome-is | evidence_not_assessable | - |
| verification-ready-to-delivery | invoke-verify-findings | invoke-deliver-audit | outcome-is | audit_ready | - |
| verification-incomplete-to-delivery | invoke-verify-findings | invoke-deliver-audit | outcome-is | verification_incomplete | - |

## Artefacts
| ID | Artifact | Path | Format |
|----|----------|------|--------|
| paper-documents | Target claim and relevant document map | project://peer2paper/audits/{runId}/study-case/paper-documents.json | json |
| dataset-maps | Target data and variable map | project://peer2paper/audits/{runId}/study-case/dataset-maps.json | json |
| code-project | Target code and dependency map | project://peer2paper/audits/{runId}/study-case/code-project.json | json |
| document-consistency-findings | Verified supplied-document consistency findings | project://peer2paper/audits/{runId}/study-case/document-consistency-findings.json | json |
| frozen-study-case | Compact frozen study case contract | project://peer2paper/audits/{runId}/study-case/frozen-study-case.json | json |
| reproduction-inputs | Frozen reproduction contract and inputs | project://peer2paper/audits/{runId}/reproduction/reproduction-inputs.json | json |
| execution-readiness | Execution readiness | project://peer2paper/audits/{runId}/reproduction/execution-readiness.json | json |
| execution-profile | Execution profile attempt | project://peer2paper/audits/{runId}/reproduction/execution-profile.json | json |
| primary-run | Untouched primary run checkpoint | project://peer2paper/audits/{runId}/reproduction/checkpoints/primary-run.json | json |
| primary-comparison | Primary numerical comparison | project://peer2paper/audits/{runId}/reproduction/checkpoints/primary-comparison.json | json |
| repair-candidates | Frozen repair candidates | project://peer2paper/audits/{runId}/reproduction/checkpoints/repair-candidates.json | json |
| repair-attempt-1 | Repair attempt 1 checkpoint | project://peer2paper/audits/{runId}/reproduction/checkpoints/repair-attempt-01.json | json |
| repair-attempt-2 | Repair attempt 2 checkpoint | project://peer2paper/audits/{runId}/reproduction/checkpoints/repair-attempt-02.json | json |
| repair-attempt-3 | Repair attempt 3 checkpoint | project://peer2paper/audits/{runId}/reproduction/checkpoints/repair-attempt-03.json | json |
| secondary-target-1 | Secondary target 1 checkpoint | project://peer2paper/audits/{runId}/reproduction/checkpoints/secondary-target-01.json | json |
| secondary-target-2 | Secondary target 2 checkpoint | project://peer2paper/audits/{runId}/reproduction/checkpoints/secondary-target-02.json | json |
| secondary-target-3 | Secondary target 3 checkpoint | project://peer2paper/audits/{runId}/reproduction/checkpoints/secondary-target-03.json | json |
| reproduction-evidence | Per-target reproduction evidence | project://peer2paper/audits/{runId}/reproduction/evidence | directory |
| reproduction-package | Reproduction package | project://peer2paper/audits/{runId}/reproduction/reproduction-package.json | json |
| estimand-rules | Frozen estimand and conclusion rules | project://peer2paper/audits/{runId}/robustness/estimand-rules.json | json |
| data-measurement-candidates | Data and measurement candidates | project://peer2paper/audits/{runId}/robustness/lanes/data-measurement.json | json |
| model-specification-candidates | Model specification candidates | project://peer2paper/audits/{runId}/robustness/lanes/model-specification.json | json |
| inference-candidates | Inference candidates | project://peer2paper/audits/{runId}/robustness/lanes/inference.json | json |
| ranked-candidates | Ranked sensitivity candidate pool | project://peer2paper/audits/{runId}/robustness/ranked-candidates.json | json |
| blind-review-alpha | Primary blind admissibility review | project://peer2paper/audits/{runId}/robustness/blind-review-alpha.json | json |
| blind-review-beta | Conditional second blind review | project://peer2paper/audits/{runId}/robustness/blind-review-beta.json | json |
| approved-candidates | Approved sensitivity candidates | project://peer2paper/audits/{runId}/robustness/approved-candidates.json | json |
| candidate-runs | Isolated sensitivity runs | project://peer2paper/audits/{runId}/robustness/candidate-runs.json | json |
| robustness-map | Sensitivity result and coverage map | project://peer2paper/audits/{runId}/robustness/robustness-map.json | json |
| robustness-version-history | Sensitivity version history | project://peer2paper/audits/{runId}/robustness/versions | directory |
| research-brief | Targeted research questions | project://peer2paper/audits/{runId}/research/research-brief.json | json |
| screened-source-registry | Screened source shortlist and slot plan | project://peer2paper/audits/{runId}/research/screened-sources.json | json |
| full-text-source-1 | Source evidence checkpoint 1 | project://peer2paper/audits/{runId}/research/source-checkpoints/source-01.json | json |
| full-text-source-2 | Source evidence checkpoint 2 | project://peer2paper/audits/{runId}/research/source-checkpoints/source-02.json | json |
| full-text-source-3 | Source evidence checkpoint 3 | project://peer2paper/audits/{runId}/research/source-checkpoints/source-03.json | json |
| full-text-map | Assembled targeted full-text evidence | project://peer2paper/audits/{runId}/research/full-text-map.json | json |
| literature-evidence-package | Targeted literature evidence package | project://peer2paper/audits/{runId}/research/literature-evidence-package.json | json |
| literature-version-history | Targeted evidence version history | project://peer2paper/audits/{runId}/research/versions | directory |
| verification-inputs | Frozen important verification targets | project://peer2paper/audits/{runId}/verification/verification-inputs.json | json |
| statistical-verification | Independent statistical rerun | project://peer2paper/audits/{runId}/verification/statistical-verification.json | json |
| source-verification | Independent source and document verification | project://peer2paper/audits/{runId}/verification/source-verification.json | json |
| adjudication-package | Verified finding package | project://peer2paper/audits/{runId}/verification/adjudication-package.json | json |
| manuscript-recommendations | Concrete manuscript recommendations | project://peer2paper/audits/{runId}/verification/manuscript-recommendations.json | json |
| audit-json | Canonical audit JSON | project://peer2paper/audits/{runId}/delivery/audit.json | json |
| replay-package | Replay package | project://peer2paper/audits/{runId}/delivery/replay | directory |
| audit-html | Researcher audit report HTML | project://peer2paper/audits/{runId}/delivery/audit.html | html |
| release-gates | Scoped release checks | project://peer2paper/audits/{runId}/delivery/release-gates.json | json |
| audit-package | Audit package manifest | project://peer2paper/audits/{runId}/delivery/audit-package.json | json |
