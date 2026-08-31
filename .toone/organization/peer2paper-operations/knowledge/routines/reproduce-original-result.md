# ROUTINE: Reproduce Original Result

> **Status**: Production | **Updated**: 2026-08-31

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
Execute the exact analysis_contract embedded in the frozen study case through the shared runner, using an attempt-local working copy and bounded technical repairs without changing scientific expressions, then report execution_mode separately from the component-level scientific comparison.

## Inputs
| ID | Kind | Source | Binding | Required | Type | Description |
|----|------|--------|---------|----------|------|-------------|
| case-bundle | parameter | Parent | - | dispatch | directory | Caller-supplied case directory containing the exact source code, data, and supporting files referenced by the frozen study case. |
| frozen-study-case-resource | resource | Parent | project://peer2paper/subroutine-inputs/frozen-study-case.json | dispatch | file | Parent-produced frozen study case JSON. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |
| execution-policy | parameter | Parent | - | dispatch | json | Runtime, host-network, 600-second command timeout, operational memory and storage rules, source immutability, internal release, and resource-control requirements. maximum_cpu_cores is best-effort unless the policy explicitly requires hard enforcement for safety, authorization, or infrastructure protection. |
| scientific-policy | parameter | Parent | - | dispatch | json | Required scientific rules for estimands, tolerances, admissibility, comparability, severity, disagreement, and allowed defaults. Supply an empty JSON object only when the routine's recorded defaults are intended. |
| execution-runner | resource | Parent | project://peer2paper/execution/runner.py | dispatch | file | Parent-supplied shared scientific execution runner. Every scientific command uses this runner, an attempt-local working directory, parse-only R validation, input and output hashing, and the command timeout. |

## Workflow
1. **Freeze reproduction contract and inputs**: Freeze the mandatory primary target and embedded analysis_contract, a separate array of no more than three secondary benchmarks, comparison rules, untouched command, data and source hashes, and dependencies.
   - id: freeze-reproduction-inputs
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: frozen-study-case-resource
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - output-artefact: reproduction-inputs
   - completion: The frozen study case contains primary_target and analysis_contract with code and data hashes, formula, population filter, exact transformations, factor levels, coefficient, orientation, missing-data declaration, entrypoint, and dependencies.
   - completion: The supplied case bundle contains the source and data identities referenced by the analysis contract.
   - completion: The checkpoint fixes one primary target and a separate array of no more than three secondary benchmarks with reported estimate, interval, statistic, p-value, sample size, and comparison rules.
   - execution: inherit
2. **Preflight shared execution**: Use the project execution runner to validate the real runtime, dependencies, input hashes, attempt-local working copy, source immutability, host network policy, operational controls, cooperative CPU settings, and a trivial language command before the study command.
   - id: preflight-execution
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - output-artefact: execution-readiness
   - output-artefact: execution-profile
   - completion: Execution readiness records the case bundle, source and data hashes, runtime, dependencies, attempt-local working directory, registered input paths, permissions, host network policy, timeout, and available controls with exact evidence.
   - completion: When execution may proceed, execution-readiness and execution-profile set status to ready and execution_allowed to true. They preserve nonblocking control gaps in limitations rather than inventing another readiness status.
   - completion: maximum_cpu_cores is best-effort unless execution-policy explicitly requires hard enforcement for safety, authorization, or infrastructure protection. Set available cooperative thread controls to the requested value and record their names and values.
   - completion: When hard CPU enforcement is unavailable but not explicitly required, include a limitation with type cpu_limit_best_effort, requested_cores from execution-policy, and hard_enforcement_available false; keep status ready and execution_allowed true.
   - completion: Hard CPU enforcement may block only when execution-policy explicitly marks it as required for safety, authorization, or infrastructure protection and no safe authorized execution path exists.
   - completion: A missing runtime or dependency, unreadable required file, source or data hash mismatch, missing permission to create a protected attempt-local working copy, unenforceable required network isolation, or another explicit safety or authority requirement may block execution.
   - completion: Execution profile is always written as an attempt envelope. Operational controls and limitations are preserved for downstream consumers, which trust execution_allowed rather than reinterpreting a best-effort CPU field.
   - completion: Every scientific command has a 600-second hard timeout. No advisory routine budget is inferred.
   - execution: inherit
3. **Run untouched primary target**: When a clean supplied entrypoint exists, copy the required inputs into a separate attempt-local working directory and execute the unchanged entrypoint once. Do not generate or use a wrapper in this step; otherwise write a skipped checkpoint for bounded repair.
   - id: run-primary-unchanged
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-readiness
   - input-artefact: execution-profile
   - output-artefact: primary-run
   - completion: The checkpoint is written for every run with status completed, failed, blocked, skipped_for_repair, or not_assessable and records the executable-contract hash.
   - completion: This step never creates or runs a wrapper. An unresolved working directory, input binding, output capture, or section-selection need writes skipped_for_repair and proceeds to the frozen repair candidates.
   - completion: Before execution, the runner verifies source and data hashes, parses any R command file without evaluating it, and confirms that the working directory is inside the attempt directory.
   - completion: A completed attempt proves that the source hashes, population filter, formula, transformations, treatment arms, factor levels, reference group, contrast, missing-data behavior, and coefficient orientation match the executable contract.
   - completion: A completed or failed attempt records the command, 600-second timeout, working directory, runtime, dependencies, inputs, outputs, logs, environment details, and hashes. Verified inputs are hashed before and after and must remain unchanged.
   - completion: Model diagnostics include initial eligible rows, model-frame rows, nobs(), excluded-row count, na.action class and excluded-row evidence, coefficient and standard error, test statistic and degrees of freedom, p-value, interval endpoints, interval method, confidence level, and coefficient orientation when scientific source execution reaches them.
   - completion: A best-effort CPU limitation never changes a ready execution profile to blocked and never prevents the untouched attempt or an eligible bounded repair.
   - execution: inherit
4. **Compare primary result**: Compare the untouched primary attempt with the reported result and executable contract. Score estimate, interval, statistic, p-value, and sample independently, and check internal numerical coherence without treating one matching component as evidence for another.
   - id: compare-primary-result
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: scientific-policy
   - input-artefact: reproduction-inputs
   - input-artefact: primary-run
   - output-artefact: primary-comparison
   - completion: The checkpoint records reported and reproduced values, absolute and relative differences, tolerance basis, displayed precision, and a separate pass, fail, not_reported, or not_comparable status for estimate, each interval endpoint, statistic, p-value, and sample.
   - completion: The checkpoint records initial eligible rows, model-frame rows, nobs(), excluded-row count, na.action class and evidence, coefficient standard error, degrees of freedom, interval method, confidence level, and coefficient orientation.
   - completion: For reported estimate 0.028 and interval [0.006, 0.041], the checkpoint records midpoint 0.0235 and coefficient-to-midpoint discrepancy 0.0045, then determines the interval method from evidence instead of inferring interval validity from the coefficient.
   - completion: exact_reproduction or within_tolerance is eligible only when every comparable required numerical component passes its frozen rule and the executed model specification matches the contract.
   - completion: Missing execution evidence is classified from the preceding checkpoint and never treated as a numerical result.
   - completion: Record execution_mode separately as untouched or technically_repaired_original. Record comparison_status separately as exact_reproduction, within_tolerance, partial_reproduction, numerical_mismatch, or not_executable.
   - completion: A technically repaired execution is scored under the same component rules as an untouched execution; repair provenance cannot replace or determine the scientific comparison status.
   - execution: inherit
5. **Diagnose failure and freeze repairs**: After an unchanged execution failure, a skipped clean entrypoint, or a numerical mismatch, freeze no more than three ordered technical repairs from the explicit allowlist. Record denylisted proposals as rejected and never execute them.
   - id: diagnose-freeze-repairs
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-readiness
   - input-artefact: execution-profile
   - input-artefact: primary-run
   - input-artefact: primary-comparison
   - output-artefact: repair-candidates
   - completion: The checkpoint always exists and has status candidates_frozen or not_required.
   - completion: Allowed repairs are limited to binding the attempt-local working directory, wrapping unchanged source for launch or output capture, selecting an author-declared replication section, binding registered input paths, and capturing runtime dependencies.
   - completion: Filters, sample rules, formulas, transformations, factor levels, treatment arms, contrasts, missing-data behavior, intervals, and inferential methods cannot change.
   - completion: Each wrapper repair requires ASCII-safe string encoding with encodeString(value, quote = '"'), forbids dQuote and sQuote, persists the wrapper in the attempt directory, and requires parse-only validation before scientific execution.
   - completion: When the source declares a replication start, contains an empty setwd call, reads a registered local input, and may fail later during optional output work, rank as candidate 1 a wrapper that runs from the writable attempt directory, binds or copies the registered input there, starts at the author-declared replication section, neutralizes only the empty setwd call, and captures the target model before optional plotting or report-generation failures.
   - execution: inherit
6. **Test repair candidate 1**: Test the first frozen repair in an isolated execution copy when assigned and needed; otherwise write a skipped or not_required checkpoint.
   - id: test-repair-1
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-profile
   - input-artefact: primary-comparison
   - input-artefact: repair-candidates
   - output-artefact: repair-attempt-1
   - completion: The first candidate always writes a checkpoint with attempted, parse_failed, completed, failed, skipped, or not_required status and the exact reason.
   - completion: A generated R wrapper is stored in the attempt directory, uses ASCII-safe encoded string literals, contains no dQuote, sQuote, or typographic quotes, and passes parse-only validation before supplied scientific source is evaluated.
   - completion: An attempted repair records the wrapper content and hash, operational change, unchanged scientific expressions, source and data hashes before and after, the 600-second timeout, execution log, and outcome.
   - completion: A successful working-directory, launch, section-selection, output-capture, input-binding, or dependency-capture repair is classified technically_repaired_original.
   - completion: A parse failure or runtime failure consumes only this bounded candidate, preserves its evidence, and allows the next frozen candidate to run. A repair that changes the analysis contract cannot support patched_reproduction.
   - completion: A completed repaired run captures initial eligible rows, model rows, nobs(), excluded rows and na.action evidence, coefficient, standard error, statistic, degrees of freedom, p-value, interval endpoints and method, confidence level, and orientation.
   - completion: For the declared replication-section repair, the wrapper runs through the official runner from the attempt directory, binds or copies the registered CSV into that directory, begins at the author-declared replication marker, neutralizes only setwd("") in that section, and captures the target model and diagnostics before optional plotting.
   - completion: The checkpoint records execution_mode technically_repaired_original plus component-level comparison evidence; it does not use repaired execution as a scientific verdict.
   - completion: Missing hard CPU affinity cannot skip this repair when execution_allowed is true.
   - execution: inherit
7. **Test repair candidate 2**: Test the second frozen repair only when the first did not yield a valid reproduced primary result; otherwise write a skipped or not_required checkpoint.
   - id: test-repair-2
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-profile
   - input-artefact: repair-candidates
   - input-artefact: repair-attempt-1
   - output-artefact: repair-attempt-2
   - completion: The second candidate always writes a checkpoint with attempted, parse_failed, completed, failed, skipped, or not_required status and the exact reason.
   - completion: A generated R wrapper is stored in the attempt directory, uses ASCII-safe encoded string literals, contains no dQuote, sQuote, or typographic quotes, and passes parse-only validation before supplied scientific source is evaluated.
   - completion: An attempted repair records the wrapper content and hash, operational change, unchanged scientific expressions, source and data hashes before and after, the 600-second timeout, execution log, and outcome.
   - completion: A successful working-directory, launch, section-selection, output-capture, input-binding, or dependency-capture repair is classified technically_repaired_original.
   - completion: A parse failure or runtime failure consumes only this bounded candidate, preserves its evidence, and allows the next frozen candidate to run. A repair that changes the analysis contract cannot support patched_reproduction.
   - completion: A completed repaired run captures initial eligible rows, model rows, nobs(), excluded rows and na.action evidence, coefficient, standard error, statistic, degrees of freedom, p-value, interval endpoints and method, confidence level, and orientation.
   - completion: Missing hard CPU affinity cannot skip this repair when execution_allowed is true.
   - execution: inherit
8. **Test repair candidate 3**: Test the third frozen repair only when earlier candidates did not yield a valid reproduced primary result; otherwise write a skipped or not_required checkpoint.
   - id: test-repair-3
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-profile
   - input-artefact: repair-candidates
   - input-artefact: repair-attempt-1
   - input-artefact: repair-attempt-2
   - output-artefact: repair-attempt-3
   - completion: The third candidate always writes a checkpoint with attempted, parse_failed, completed, failed, skipped, or not_required status and the exact reason.
   - completion: A generated R wrapper is stored in the attempt directory, uses ASCII-safe encoded string literals, contains no dQuote, sQuote, or typographic quotes, and passes parse-only validation before supplied scientific source is evaluated.
   - completion: An attempted repair records the wrapper content and hash, operational change, unchanged scientific expressions, source and data hashes before and after, the 600-second timeout, execution log, and outcome.
   - completion: A successful working-directory, launch, section-selection, output-capture, input-binding, or dependency-capture repair is classified technically_repaired_original.
   - completion: A parse failure or runtime failure consumes only this bounded candidate, preserves its evidence, and allows the next frozen candidate to run. A repair that changes the analysis contract cannot support patched_reproduction.
   - completion: A completed repaired run captures initial eligible rows, model rows, nobs(), excluded rows and na.action evidence, coefficient, standard error, statistic, degrees of freedom, p-value, interval endpoints and method, confidence level, and orientation.
   - completion: Missing hard CPU affinity cannot skip this repair when execution_allowed is true.
   - execution: inherit
9. **Run secondary target 1**: After the primary result succeeds unchanged or through a valid repair, run the first required secondary benchmark if assigned. Otherwise write not_assigned, blocked, or skipped.
   - id: run-secondary-target-1
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-profile
   - input-artefact: primary-comparison
   - input-artefact: repair-attempt-1
   - input-artefact: repair-attempt-2
   - input-artefact: repair-attempt-3
   - output-artefact: secondary-target-1
   - completion: The checkpoint always exists and records one benchmark comparison or the exact reason it was not run.
   - completion: The scientific command runs when the primary target has valid comparable execution evidence under either execution_mode, including a partial component comparison; it does not run after not_executable or not_assessable.
   - execution: inherit
10. **Run secondary target 2**: After the primary result succeeds unchanged or through a valid repair, run the second required secondary benchmark if assigned. Otherwise write not_assigned, blocked, or skipped.
   - id: run-secondary-target-2
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-profile
   - input-artefact: primary-comparison
   - input-artefact: repair-attempt-1
   - input-artefact: repair-attempt-2
   - input-artefact: repair-attempt-3
   - output-artefact: secondary-target-2
   - completion: The checkpoint always exists and records one benchmark comparison or the exact reason it was not run.
   - completion: The scientific command runs when the primary target has valid comparable execution evidence under either execution_mode, including a partial component comparison; it does not run after not_executable or not_assessable.
   - execution: inherit
11. **Run secondary target 3**: After the primary result succeeds unchanged or through a valid repair, run the third required secondary benchmark if assigned. Otherwise write not_assigned, blocked, or skipped.
   - id: run-secondary-target-3
   - agent: peer2paper-reproduction/reproduction-engineer
   - routine-input: execution-runner
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: case-bundle
   - input-artefact: reproduction-inputs
   - input-artefact: execution-profile
   - input-artefact: primary-comparison
   - input-artefact: repair-attempt-1
   - input-artefact: repair-attempt-2
   - input-artefact: repair-attempt-3
   - output-artefact: secondary-target-3
   - completion: The checkpoint always exists and records one benchmark comparison or the exact reason it was not run.
   - completion: The scientific command runs when the primary target has valid comparable execution evidence under either execution_mode, including a partial component comparison; it does not run after not_executable or not_assessable.
   - execution: inherit
12. **Package reproduction result**: Assemble the durable readiness, primary, repair, and secondary checkpoints into a compact comparison package and replay directory without rerunning analysis.
   - id: package-reproduction
   - agent: peer2paper-reproduction/reproduction-engineer
   - input-artefact: reproduction-inputs
   - input-artefact: execution-readiness
   - input-artefact: execution-profile
   - input-artefact: primary-run
   - input-artefact: primary-comparison
   - input-artefact: repair-candidates
   - input-artefact: repair-attempt-1
   - input-artefact: repair-attempt-2
   - input-artefact: repair-attempt-3
   - input-artefact: secondary-target-1
   - input-artefact: secondary-target-2
   - input-artefact: secondary-target-3
   - output-artefact: reproduction-evidence
   - output-artefact: reproduction-package
   - completion: The reproduction package records or content-addresses the frozen analysis contract, case-bundle identities, execution profile, commands, 600-second timeout, environment, inputs, outputs, logs, hashes, repairs, numerical diagnostics, component comparisons, and replay instructions.
   - completion: Every completed checkpoint remains durable, and a later failure cannot erase earlier successful evidence.
   - completion: A successful permitted wrapper records execution_mode technically_repaired_original. The routine outcome follows comparison_status and never follows the repair label.
   - completion: comparison_status exact_reproduction or within_tolerance requires every comparable required estimate, interval endpoint, statistic, p-value, and sample component to pass its frozen rule, regardless of execution_mode.
   - completion: A missing hard CPU limit is recorded as cpu_limit_best_effort and cannot produce not_executable when execution_allowed is true. Required network isolation, source protection, and explicit safety or authority controls remain gates.
   - completion: External release remains disabled unless separately approved.
   - completion: comparison_status is partial_reproduction when valid execution yields a mixed component result, including a matching estimate, statistic, and p-value with a failing reported interval. A paper-omitted model sample is recorded as not_reported and never silently treated as a match.
   - outcome: exact_reproduction | terminal | Every comparable required component reproduces exactly under frozen rules, using either execution_mode.
   - outcome: within_tolerance | terminal | Every comparable required component passes its frozen tolerance rule, using either execution_mode.
   - outcome: partial_reproduction | terminal | Valid execution produced a mixed component comparison, such as matching estimate, statistic, and p-value with a failing interval.
   - outcome: numerical_mismatch | terminal | The executable analysis remains materially inconsistent after the bounded remediation pass and does not qualify as a partial component match.
   - outcome: not_executable | terminal | A missing runtime or dependency, unreadable required input, source or data hash mismatch, permission failure, required isolation failure, or inability to create a protected attempt-local working copy prevents every scientific attempt. Best-effort CPU enforcement alone cannot emit this outcome.
   - outcome: not_assessable | terminal | The target cannot be assessed without an unsupported substantive choice.
   - execution: inherit

## Transitions
| ID | From Step | To Step | Condition | Value | Max Traversals |
|----|-----------|---------|-----------|-------|----------------|
| freeze-to-preflight | freeze-reproduction-inputs | preflight-execution | on-success | - | - |
| preflight-to-primary | preflight-execution | run-primary-unchanged | on-success | - | - |
| primary-to-comparison | run-primary-unchanged | compare-primary-result | on-success | - | - |
| comparison-to-diagnosis | compare-primary-result | diagnose-freeze-repairs | on-success | - | - |
| diagnosis-to-repair-1 | diagnose-freeze-repairs | test-repair-1 | on-success | - | - |
| repair-1-to-repair-2 | test-repair-1 | test-repair-2 | on-success | - | - |
| repair-2-to-repair-3 | test-repair-2 | test-repair-3 | on-success | - | - |
| repairs-to-secondary-1 | test-repair-3 | run-secondary-target-1 | on-success | - | - |
| repairs-to-secondary-2 | test-repair-3 | run-secondary-target-2 | on-success | - | - |
| repairs-to-secondary-3 | test-repair-3 | run-secondary-target-3 | on-success | - | - |
| secondary-1-to-package | run-secondary-target-1 | package-reproduction | on-success | - | - |
| secondary-2-to-package | run-secondary-target-2 | package-reproduction | on-success | - | - |
| secondary-3-to-package | run-secondary-target-3 | package-reproduction | on-success | - | - |

## Artefacts
| ID | Artifact | Path | Format |
|----|----------|------|--------|
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
