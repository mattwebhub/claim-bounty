# ROUTINE: Stress Test Analysis

> **Status**: Production | **Updated**: 2026-08-31 | **Scope**: Public ClaimBounty workflow export

## Metadata

- **Department:** claim-bounty-operations
- **Agent:** ClaimBounty Orchestrator
- **Cadence:** OnDemand
- **Type:** sub-routine
- **Parent:** claim-bounty-operations/run-claimbounty-scientific-audit
- **Max Concurrency:** 4
- **Status:** Production

## Purpose

Test three bounded families of scientifically defensible alternatives that preserve the frozen estimand, approve no more than five result-blind candidate cards, execute them through the shared local runner with bounded commands, and report one interpretable sensitivity result.

## Inputs

| ID                            | Kind      | Source | Binding                                                           | Required | Type      | Description                                                                                                                                                                                                                                                                                                                             |
| ----------------------------- | --------- | ------ | ----------------------------------------------------------------- | -------- | --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| case-bundle                   | parameter | Parent | -                                                                 | dispatch | directory | Caller-supplied case directory containing the exact source code, data, and supporting files referenced by the frozen study case.                                                                                                                                                                                                        |
| frozen-study-case-resource    | resource  | Parent | project://claimbounty/subroutine-inputs/frozen-study-case.json    | dispatch | file      | Parent-produced frozen study case. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy.                                                                                                                                    |
| reproduction-package-resource | resource  | Parent | project://claimbounty/subroutine-inputs/reproduction-package.json | dispatch | file      | Parent-produced reproduction package. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy.                                                                                                                                 |
| execution-policy              | parameter | Parent | -                                                                 | dispatch | json      | Frozen analysis limits, disabled analysis networking, 600-second command timeout, attempt-local working-directory rules, and operational memory and storage defaults.                                                                                                                                                                   |
| scientific-policy             | parameter | Parent | -                                                                 | dispatch | json      | Required scientific rules for estimands, tolerances, admissibility, comparability, severity, disagreement, and allowed defaults. Supply an empty JSON object only when the routine's recorded defaults are intended.                                                                                                                    |
| execution-runner              | resource  | Parent | project://claimbounty/execution                                   | dispatch | directory | Parent-supplied shared scientific execution runner. Every scientific command must use this runner.                                                                                                                                                                                                                                      |
| execution-profile-resource    | resource  | Parent | project://claimbounty/subroutine-inputs/execution-profile.json    | dispatch | file      | Parent-produced validated execution profile containing the runner, runtime, dependency, isolation, mount, and resource-control identities frozen during reproduction. Parent invocation supplies the resolved run-scoped parent artefact path at runtime; this fixed binding is a definition fallback and is not a shared staging copy. |

## Workflow

1. **Freeze sensitivity contract**: Preflight the supplied case bundle and shared local execution capability, confirm that reproduction may continue, and freeze the estimand, conclusion rules, admissibility rules, candidate cap, and stopping rules before alternative results exist.
   - id: freeze-sensitivity-contract
   - agent: claim-bounty-robustness/robustness-lead
   - routine-input: frozen-study-case-resource
   - routine-input: reproduction-package-resource
   - routine-input: execution-policy
   - routine-input: scientific-policy
   - routine-input: execution-runner
   - routine-input: execution-profile-resource
   - routine-input: case-bundle
   - output-artefact: estimand-rules
   - completion: The supplied case bundle resolves the frozen source and data hashes, and the runner can create an attempt-local working copy with the required runtime, dependencies, registered inputs, source protection, and 600-second timeout.
   - completion: Verify the frozen execution-policy digest plus live network and host-file denial before sensitivity work. Missing or failed isolation blocks execution. Memory and storage remain operational defaults and unavailable exact enforcement does not block sensitivity work by itself.
   - completion: The estimand fixes population, treatment or exposure, comparator, outcome, time, effect, observation unit, direction, practical threshold, and conclusion categories before candidate execution.
   - completion: Emit sensitivity_ready when the baseline is reproduced or explicitly authorized and the local execution preflight passes; emit sensitivity_blocked only for a scientific, source/data, runtime, dependency, permission, or source-protection blocker.
   - completion: The candidate pool is capped at five and each candidate changes one primary analytical choice.
   - outcome: sensitivity_ready | The frozen contract and shared execution capability are ready for bounded sensitivity design and execution.
   - outcome: sensitivity_blocked | terminal | A scientific, source/data, or execution-capability blocker prevents sensitivity work.
   - execution: inherit
2. **Design data and measurement candidates**: Propose no more than two high-value candidates covering sample inclusion, missingness, outcome or exposure definitions, and transformations while preserving the frozen estimand.
   - id: design-data-measurement-lane
   - agent: claim-bounty-robustness/data-choices-analyst
   - routine-input: frozen-study-case-resource
   - routine-input: reproduction-package-resource
   - routine-input: scientific-policy
   - input-artefact: estimand-rules
   - output-artefact: data-measurement-candidates
   - completion: The lane returns at most two candidate cards, each with one primary change, scientific rationale, supplied support, unchanged estimand elements, expected sample consequence, classification, diagnostics, and compatibility constraints.
   - completion: Candidate cards contain no alternative-analysis outcomes.
   - execution: inherit
3. **Design model specification candidates**: Propose no more than two high-value candidates covering covariates, functional form, weighting, or interactions while preserving the frozen estimand.
   - id: design-model-lane
   - agent: claim-bounty-robustness/model-and-covariate-analyst
   - routine-input: frozen-study-case-resource
   - routine-input: reproduction-package-resource
   - routine-input: scientific-policy
   - input-artefact: estimand-rules
   - output-artefact: model-specification-candidates
   - completion: The lane returns at most two candidate cards, each with one primary change, scientific rationale, supplied support, unchanged estimand elements, expected sample consequence, classification, diagnostics, and compatibility constraints.
   - completion: Post-treatment, collider, unavailable, and meaning-changing specifications are excluded before results exist.
   - execution: inherit
4. **Design inference candidates**: Propose no more than two high-value candidates covering standard errors, clustering, multiplicity, or confidence intervals under the frozen design.
   - id: design-inference-lane
   - agent: claim-bounty-robustness/inference-analyst
   - routine-input: frozen-study-case-resource
   - routine-input: reproduction-package-resource
   - routine-input: scientific-policy
   - input-artefact: estimand-rules
   - output-artefact: inference-candidates
   - completion: The lane returns at most two candidate cards, each with one primary change, scientific rationale, supplied support, unchanged estimand elements, classification, diagnostics, and compatibility constraints.
   - completion: Inference choices identify their design basis and contain no alternative-analysis outcomes.
   - execution: inherit
5. **Rank and freeze sensitivity candidates**: Join the three lanes, remove duplicates and incompatible combinations, and freeze a ranked pool of three to five candidate cards before results are available.
   - id: rank-sensitivity-candidates
   - agent: claim-bounty-robustness/robustness-lead
   - input-artefact: estimand-rules
   - input-artefact: data-measurement-candidates
   - input-artefact: model-specification-candidates
   - input-artefact: inference-candidates
   - output-artefact: ranked-candidates
   - completion: The frozen pool contains three to five candidates when that many defensible options exist, with stable IDs, one primary change each, rationale, support, unchanged estimand elements, expected sample consequence, type, cost, and no result fields.
   - completion: Excluded and unselected options retain concise reasons, and the selected pool covers more than one decision lane when scientifically possible.
   - execution: inherit
6. **Run primary blind admissibility review**: Judge each candidate from results-blind cards under the frozen admissibility rules and identify whether a second review is required for a dispute or high-impact judgment.
   - id: review-candidates-alpha
   - agent: claim-bounty-verification/blind-method-reviewer-alpha
   - routine-input: scientific-policy
   - input-artefact: estimand-rules
   - input-artefact: ranked-candidates
   - output-artefact: blind-review-alpha
   - completion: Every candidate receives accept, supplementary, reject, or unresolved with criterion-level reasons and no result exposure.
   - completion: Emit second_review_required only for a disagreement risk or high-impact admissibility judgment; otherwise emit primary_review_sufficient.
   - outcome: primary_review_sufficient | Every candidate can be resolved from the primary blind review.
   - outcome: second_review_required | At least one disputed or high-impact judgment needs a second blind review.
   - execution: inherit
7. **Record conditional second blind review**: Always write the second-review artifact. Review only candidates flagged as disputed or high-impact by the primary reviewer; when none are flagged, emit not_required without repeating the primary review.
   - id: review-candidates-beta
   - agent: claim-bounty-verification/blind-method-reviewer-beta
   - routine-input: scientific-policy
   - input-artefact: estimand-rules
   - input-artefact: ranked-candidates
   - input-artefact: blind-review-alpha
   - output-artefact: blind-review-beta
   - completion: The artifact always exists.
   - completion: Flagged candidates receive an independent criterion-level blind judgment with no result exposure.
   - completion: Every unflagged candidate is marked not_required with the primary-review reference; no substantive second review is performed for it.
   - execution: inherit
8. **Resolve candidate admissibility**: Apply the frozen disagreement rule once and produce the final set of executable sensitivity candidates without reconsidering admissibility later.
   - id: resolve-candidate-admissibility
   - agent: claim-bounty-robustness/robustness-lead
   - input-artefact: ranked-candidates
   - input-artefact: blind-review-alpha
   - input-artefact: blind-review-beta
   - output-artefact: approved-candidates
   - completion: No more than five accepted or supplementary candidates are executable, and every rejected or unresolved candidate retains its review evidence.
   - completion: Each approved candidate preserves the estimand and has one primary analytical change.
   - execution: inherit
9. **Execute approved sensitivity analyses**: Use the shared runner to execute each approved candidate as an isolated worker, compare outputs under one result schema, and preserve failed or untested candidates.
   - id: execute-approved-candidates
   - agent: claim-bounty-reproduction/reproduction-engineer
   - routine-input: execution-policy
   - routine-input: execution-runner
   - routine-input: execution-profile-resource
   - routine-input: case-bundle
   - input-artefact: estimand-rules
   - input-artefact: approved-candidates
   - output-artefact: candidate-runs
   - completion: Execution begins only after sensitivity_ready routing and after ranked candidates, blind-review-alpha, and the always-materialized blind-review-beta checkpoint have joined.
   - completion: Each approved candidate runs against the frozen source and data hashes through the shared runner from a separate attempt-local working directory and has a command, 600-second timeout, inputs, outputs, logs, diagnostics, estimate, interval, p-value, sample size, status, and hashes.
   - completion: Verified inputs remain unchanged. The verified network and host-file isolation identities and operational memory or storage defaults are recorded; differences in operational identities are limitations rather than scientific-validity results.
   - completion: Candidate execution is capped at five; failed, timed-out, and untested candidates remain visible without open-ended repair.
   - execution: inherit
10. **Build sensitivity result**: Summarize stability, fragility, coverage, and the smallest defensible conclusion change from approved candidate runs.

- id: build-robustness-map
- agent: claim-bounty-robustness/robustness-lead
- input-artefact: estimand-rules
- input-artefact: approved-candidates
- input-artefact: candidate-runs
- output-artefact: robustness-map
- output-artefact: robustness-version-history
- completion: The result states how many approved analyses preserve direction and statistical support, the estimate and interval range, any valid direction reversal, the smallest defensible conclusion-changing choice, and whether changes arise from estimate size, uncertainty, or sample composition.
- completion: Tested and untested dimensions, failed candidates, coverage, and limitations are explicit; significance counts alone are not used as the conclusion.
- completion: Write an immutable run-scoped sensitivity snapshot alongside the selected per-run result; no prior-run replacement read is required.
- outcome: robustness_ready | terminal | Required sensitivity checks completed without residual gaps.
- outcome: robustness_ready_with_limits | terminal | Sensitivity checks completed with visible nonblocking gaps.
- outcome: analysis_not_assessable | terminal | A blocking scientific or executable gap prevents a defensible sensitivity conclusion.
- outcome: operationally_blocked | terminal | Authority or unsafe execution prevents required work.
- execution: inherit

## Transitions

| ID                         | From Step                       | To Step                         | Condition  | Value                     | Max Traversals |
| -------------------------- | ------------------------------- | ------------------------------- | ---------- | ------------------------- | -------------- |
| contract-to-data-lane      | freeze-sensitivity-contract     | design-data-measurement-lane    | outcome-is | sensitivity_ready         | -              |
| contract-to-model-lane     | freeze-sensitivity-contract     | design-model-lane               | outcome-is | sensitivity_ready         | -              |
| contract-to-inference-lane | freeze-sensitivity-contract     | design-inference-lane           | outcome-is | sensitivity_ready         | -              |
| data-lane-to-rank          | design-data-measurement-lane    | rank-sensitivity-candidates     | on-success | -                         | -              |
| model-lane-to-rank         | design-model-lane               | rank-sensitivity-candidates     | on-success | -                         | -              |
| inference-lane-to-rank     | design-inference-lane           | rank-sensitivity-candidates     | on-success | -                         | -              |
| rank-to-alpha              | rank-sensitivity-candidates     | review-candidates-alpha         | on-success | -                         | -              |
| alpha-to-beta              | review-candidates-alpha         | review-candidates-beta          | outcome-is | second_review_required    | -              |
| beta-to-resolution         | review-candidates-beta          | resolve-candidate-admissibility | on-success | -                         | -              |
| resolution-to-execution    | resolve-candidate-admissibility | execute-approved-candidates     | on-success | -                         | -              |
| execution-to-map           | execute-approved-candidates     | build-robustness-map            | on-success | -                         | -              |
| alpha-sufficient-to-beta   | review-candidates-alpha         | review-candidates-beta          | outcome-is | primary_review_sufficient | -              |

## Handoffs

| ID                              | From Step                       | To Step                         | Artefacts                                                                                                      | Mode         | Acceptance                                                                                                   | Legacy Soft Budget | Retries | Backoff | Model |
| ------------------------------- | ------------------------------- | ------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------ | ------------------ | ------- | ------- | ----- |
| contract-to-data-handoff        | freeze-sensitivity-contract     | design-data-measurement-lane    | estimand-rules; project://claimbounty/audits/{runId}/robustness/estimand-rules.json                            | await-result | The estimand and conclusion rules are frozen and contain no candidate outcomes.                              | -                  | -       | -       | -     |
| contract-to-model-handoff       | freeze-sensitivity-contract     | design-model-lane               | estimand-rules; project://claimbounty/audits/{runId}/robustness/estimand-rules.json                            | await-result | The estimand and conclusion rules are frozen and contain no candidate outcomes.                              | -                  | -       | -       | -     |
| contract-to-inference-handoff   | freeze-sensitivity-contract     | design-inference-lane           | estimand-rules; project://claimbounty/audits/{runId}/robustness/estimand-rules.json                            | await-result | The estimand and conclusion rules are frozen and contain no candidate outcomes.                              | -                  | -       | -       | -     |
| data-to-rank-handoff            | design-data-measurement-lane    | rank-sensitivity-candidates     | data-measurement-candidates; project://claimbounty/audits/{runId}/robustness/lanes/data-measurement.json       | await-result | The lane contains at most two result-blind candidate cards with one primary change each.                     | -                  | -       | -       | -     |
| model-to-rank-handoff           | design-model-lane               | rank-sensitivity-candidates     | model-specification-candidates; project://claimbounty/audits/{runId}/robustness/lanes/model-specification.json | await-result | The lane contains at most two result-blind candidate cards with one primary change each.                     | -                  | -       | -       | -     |
| inference-to-rank-handoff       | design-inference-lane           | rank-sensitivity-candidates     | inference-candidates; project://claimbounty/audits/{runId}/robustness/lanes/inference.json                     | await-result | The lane contains at most two result-blind candidate cards with one primary change each.                     | -                  | -       | -       | -     |
| rank-to-alpha-handoff           | rank-sensitivity-candidates     | review-candidates-alpha         | ranked-candidates; project://claimbounty/audits/{runId}/robustness/ranked-candidates.json                      | await-result | The pool is frozen, capped at five, and contains no alternative-analysis outcomes.                           | -                  | -       | -       | -     |
| alpha-to-beta-handoff           | review-candidates-alpha         | review-candidates-beta          | blind-review-alpha; project://claimbounty/audits/{runId}/robustness/blind-review-alpha.json                    | await-result | The primary review identifies the disputed or high-impact candidates requiring a second blind judgment.      | -                  | -       | -       | -     |
| beta-to-resolution-handoff      | review-candidates-beta          | resolve-candidate-admissibility | blind-review-beta; project://claimbounty/audits/{runId}/robustness/blind-review-beta.json                      | await-result | Flagged candidates have a separate blind judgment; all other candidates are marked not_required.             | -                  | -       | -       | -     |
| resolution-to-execution-handoff | resolve-candidate-admissibility | execute-approved-candidates     | approved-candidates; project://claimbounty/audits/{runId}/robustness/approved-candidates.json                  | await-result | Only accepted or supplementary candidates are executable, capped at five, with rejected decisions preserved. | -                  | -       | -       | -     |
| execution-to-map-handoff        | execute-approved-candidates     | build-robustness-map            | candidate-runs; project://claimbounty/audits/{runId}/robustness/candidate-runs.json                            | await-result | Every approved candidate has a common-schema run record or an exact failed or untested status.               | -                  | -       | -       | -     |

## Artefacts

| ID                             | Artifact                             | Path                                                                           | Format    |
| ------------------------------ | ------------------------------------ | ------------------------------------------------------------------------------ | --------- |
| estimand-rules                 | Frozen estimand and conclusion rules | project://claimbounty/audits/{runId}/robustness/estimand-rules.json            | json      |
| data-measurement-candidates    | Data and measurement candidates      | project://claimbounty/audits/{runId}/robustness/lanes/data-measurement.json    | json      |
| model-specification-candidates | Model specification candidates       | project://claimbounty/audits/{runId}/robustness/lanes/model-specification.json | json      |
| inference-candidates           | Inference candidates                 | project://claimbounty/audits/{runId}/robustness/lanes/inference.json           | json      |
| ranked-candidates              | Ranked sensitivity candidate pool    | project://claimbounty/audits/{runId}/robustness/ranked-candidates.json         | json      |
| blind-review-alpha             | Primary blind admissibility review   | project://claimbounty/audits/{runId}/robustness/blind-review-alpha.json        | json      |
| blind-review-beta              | Conditional second blind review      | project://claimbounty/audits/{runId}/robustness/blind-review-beta.json         | json      |
| approved-candidates            | Approved sensitivity candidates      | project://claimbounty/audits/{runId}/robustness/approved-candidates.json       | json      |
| candidate-runs                 | Isolated sensitivity runs            | project://claimbounty/audits/{runId}/robustness/candidate-runs.json            | json      |
| robustness-map                 | Sensitivity result and coverage map  | project://claimbounty/audits/{runId}/robustness/robustness-map.json            | json      |
| robustness-version-history     | Sensitivity version history          | project://claimbounty/audits/{runId}/robustness/versions                       | directory |
