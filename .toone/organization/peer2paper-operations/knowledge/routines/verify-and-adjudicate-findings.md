# ROUTINE: Verify And Adjudicate Findings

## Metadata
- **Routine Schema:** 2
- **Department:** peer2paper-operations
- **Agent:** Peer2Paper Orchestrator
- **Cadence:** OnDemand
- **Type:** sub-routine
- **Parent:** peer2paper-operations/run-peer2paper-scientific-audit
- **Max Concurrency:** 2
- **Status:** Production

## Purpose
Independently rerun the primary and verdict-relevant sensitivity results from the same frozen analysis contract and source and data hashes in a separate local working directory, then verify decisive source evidence and adjudicate manuscript changes.

## Inputs
| ID | Kind | Source | Binding | Required | Type | Description |
|----|------|--------|---------|----------|------|-------------|
| case-bundle | parameter | Parent | - | dispatch | directory | Caller-supplied case directory containing the exact source code, data, and supporting files referenced by the frozen study case. |
| frozen-study-case-resource | resource | Parent | project://peer2paper/subroutine-inputs/frozen-study-case.json | dispatch | file | Parent-produced frozen study case. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| reproduction-package-resource | resource | Parent | project://peer2paper/subroutine-inputs/reproduction-package.json | dispatch | file | Parent-produced reproduction package. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| robustness-map-resource | resource | Parent | project://peer2paper/subroutine-inputs/robustness-map.json | dispatch | file | Parent-produced robustness map. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| literature-evidence-resource | resource | Parent | project://peer2paper/subroutine-inputs/literature-evidence-package.json | dispatch | file | Parent-produced literature evidence package. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| execution-policy | parameter | Parent | - | dispatch | json | Verification limits, 600-second command timeout, host network and source-access rules, attempt-local working-directory rules, operational memory and storage defaults, and internal-only release. |
| scientific-policy | parameter | Parent | - | dispatch | json | Required scientific rules for estimands, tolerances, admissibility, comparability, severity, disagreement, and allowed defaults. Supply an empty JSON object only when the routine's recorded defaults are intended. |
| execution-runner | resource | Parent | project://peer2paper/execution | dispatch | directory | Parent-supplied shared scientific execution runner. Every scientific command must use this runner. |
| execution-profile-resource | resource | Parent | project://peer2paper/subroutine-inputs/execution-profile.json | dispatch | file | Parent-produced validated execution profile containing the runner, runtime, dependency, isolation, mount, and resource-control identities frozen during reproduction. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| audit-request | parameter | Parent | - | dispatch | json | Audit scope, source authority, privacy and retention rules, and whether the audit package will be publicly redistributed. |

## Workflow
1. **Freeze important verification targets**: Select only verdict-relevant reproduction, sensitivity, document, and source findings, freeze their evidence identities, and gate statistical verification against the executable contract and reproduction execution identities.
   - id: freeze-verification-inputs
   - agent: peer2paper-verification/verification-lead
   - routine-input: frozen-study-case-resource
   - routine-input: reproduction-package-resource
   - routine-input: robustness-map-resource
   - routine-input: literature-evidence-resource
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: execution-runner
   - routine-input: execution-profile-resource
   - routine-input: audit-request
   - routine-input: case-bundle
   - output-artefact: verification-inputs
   - completion: Every selected finding has one precise statement, claim and estimand links, importance reason, allowed verification method, source or run identity, hashes, and required completion rule.
   - completion: The packet freezes the primary_target and analysis_contract from frozen-study-case, including source and data hashes, exact expressions, coefficient, orientation, and comparison rules.
   - completion: The supplied case bundle resolves the frozen source and data hashes, and the runner can create a separate attempt-local working directory with the required runtime, dependencies, source protection, and 600-second timeout.
   - completion: Verified network or DNS denial is not required. Memory and storage are operational defaults, and unavailable exact enforcement is recorded without blocking a runnable verification command.
   - completion: Previously decided candidate admissibility is preserved, and external release remains disabled unless the audit request contains separate approval.
   - execution: inherit
2. **Independently rerun statistical evidence**: Perform an independent execution of the frozen analysis_contract and only verdict-relevant sensitivities from a separate local working directory using the same source and data hashes.
   - id: verify-statistical-evidence
   - agent: peer2paper-verification/statistical-evidence-verifier
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: execution-runner
   - routine-input: execution-profile-resource
   - routine-input: case-bundle
   - input-artefact: verification-inputs
   - output-artefact: statistical-verification
   - completion: Preflight proves access to the case bundle, frozen source and data hashes, required runtime and dependencies, registered inputs, a separate attempt-local working directory, source protection, and a 600-second timeout.
   - completion: The verification run executes the exact frozen formula, filter, transformations, factor levels, treatment or exposure structure, coefficient, orientation, missing-data behavior, interval method, and inferential method.
   - completion: The primary run records command, working directory, timeout, environment, inputs, outputs, logs and hashes; eligible rows, model rows, nobs(), excluded rows and na.action evidence; coefficient, standard error, statistic, degrees of freedom, p-value, interval endpoints and method, confidence level, and orientation.
   - completion: Estimate, interval endpoints, statistic, p-value, and sample are compared independently with the reported and reproduction values.
   - completion: A different runner hash, workspace path, mount identity, or available operational control does not fail verification when the frozen scientific contract, source and data hashes, dependencies, source immutability, and recorded environment are equivalent for the claim.
   - completion: Verified network or DNS denial and exact memory or storage enforcement are not prerequisites. A scientific-contract or source/data hash mismatch remains a precise blocker and cannot support audit_ready.
   - execution: inherit
3. **Independently reopen source evidence**: Reopen only supplied-document conflicts and external passages used in the verdict or recommendations, using authorized local sources, toone-scholar, or the saved Browser Silk funnel for each mapped domain. Perform full licence adjudication only when audit-request says the package will be publicly redistributed.
   - id: verify-source-evidence
   - agent: peer2paper-verification/source-evidence-verifier
   - routine-input: audit-request
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: verification-inputs
   - output-artefact: source-verification
   - completion: Every verdict-relevant source statement has an independent identity, version, exact-passage, context, correction or retraction, and comparability check or a visible access limitation.
   - completion: Internal audits record access authority and provenance. Full licence verification is required only when public redistribution is intended or authorized.
   - completion: For a selector_not_found failure, perform at most one in-attempt Silk heal, retry the affected flow step once, and push a healed map. Preserve the last failure evidence for runtime attempt handling; never switch to manual browsing.
   - completion: No snippet-only statement passes verification and sources unrelated to the verdict are not reopened.
   - completion: One bounded locator correction may be made; unresolved source problems remain limitations.
   - completion: If a sensitivity result raises a new verdict-critical methods question, perform at most one bounded follow-up using the same source limits and record it in source-verification.
   - execution: inherit
4. **Adjudicate findings and manuscript changes**: Join the independent statistical and source checks, assign final finding status and allowed wording, and write specific manuscript changes without introducing new analyses or source claims.
   - id: adjudicate-verified-findings
   - agent: peer2paper-verification/scientific-adjudicator
   - input-artefact: verification-inputs
   - input-artefact: statistical-verification
   - input-artefact: source-verification
   - output-artefact: adjudication-package
   - output-artefact: manuscript-recommendations
   - completion: Every retained finding is verified, verified with limits, rejected, incomplete, or disputed, with severity, confidence, exact evidence, allowed wording, and no duplicated admissibility review.
   - completion: Each manuscript recommendation names the affected section or statement, the exact requested change, its evidence, urgency, and whether it is required or optional.
   - completion: audit_ready requires an independent rerun from a separate local working directory with the same frozen scientific contract and source and data hashes, complete required numerical comparisons, protected source inputs, recorded environment details, and no unresolved scientific blocker.
   - completion: Different runner, workspace, mount, network-control, memory-control, or storage-control identities do not block audit_ready by themselves.
   - completion: Weak nonblocking findings are dropped; operationally_blocked is reserved for missing authority or an unsafe or impossible required verification.
   - outcome: audit_ready | terminal | All required findings are independently verified and ready for reporting.
   - outcome: verification_incomplete | terminal | The audit can report verified work with explicit unresolved limitations.
   - outcome: operationally_blocked | terminal | Required verification cannot proceed safely or with authority.
   - execution: inherit

## Transitions
| ID | From Step | To Step | Condition | Value | Max Traversals |
|----|-----------|---------|-----------|-------|----------------|
| freeze-to-statistical | freeze-verification-inputs | verify-statistical-evidence | on-success | - | - |
| freeze-to-source | freeze-verification-inputs | verify-source-evidence | on-success | - | - |
| statistical-to-adjudication | verify-statistical-evidence | adjudicate-verified-findings | on-success | - | - |
| source-to-adjudication | verify-source-evidence | adjudicate-verified-findings | on-success | - | - |

## Handoffs
| ID | From Step | To Step | Artefacts | Mode | Acceptance | Legacy Soft Budget | Retries | Backoff | Model |
|----|-----------|---------|-----------|------|------------|---------|---------|---------|-------|
| freeze-to-statistical-handoff | freeze-verification-inputs | verify-statistical-evidence | verification-inputs; project://peer2paper/audits/{runId}/verification/verification-inputs.json | await-result | The packet identifies only verdict-relevant statistical findings, fixes the executable-contract hash, resolves the source and data hashes, and supplies the runtime, dependencies, attempt-local working-directory rule, source-protection evidence, and 600-second timeout needed for independent execution. | - | - | - | - |
| freeze-to-source-handoff | freeze-verification-inputs | verify-source-evidence | verification-inputs; project://peer2paper/audits/{runId}/verification/verification-inputs.json | await-result | The packet identifies only verdict-relevant source statements with stable local or external source identities and exact expected passages. | - | - | - | - |
| statistical-to-adjudication-handoff | verify-statistical-evidence | adjudicate-verified-findings | statistical-verification; project://peer2paper/audits/{runId}/verification/statistical-verification.json | await-result | The primary result and important sensitivity findings have independent local execution evidence with complete model diagnostics and per-component numerical verdicts, or precise blockers that force verification_incomplete. | - | - | - | - |
| source-to-adjudication-handoff | verify-source-evidence | adjudicate-verified-findings | source-verification; project://peer2paper/audits/{runId}/verification/source-verification.json | await-result | Verdict-relevant document and literature claims have independently reopened exact passages or explicit access limitations. | - | - | - | - |

## Artefacts
| ID | Artifact | Path | Format |
|----|----------|------|--------|
| verification-inputs | Frozen important verification targets | project://peer2paper/audits/{runId}/verification/verification-inputs.json | json |
| statistical-verification | Independent statistical rerun | project://peer2paper/audits/{runId}/verification/statistical-verification.json | json |
| source-verification | Independent source and document verification | project://peer2paper/audits/{runId}/verification/source-verification.json | json |
| adjudication-package | Verified finding package | project://peer2paper/audits/{runId}/verification/adjudication-package.json | json |
| manuscript-recommendations | Concrete manuscript recommendations | project://peer2paper/audits/{runId}/verification/manuscript-recommendations.json | json |
