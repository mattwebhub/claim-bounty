# Local Grader Routine Integration Handoff

> **Status**: Draft | **Updated**: 2026-08-31 | **Scope**: Routine integration, manual finalizer, and matching Peer2Paper routine contracts

## Outcome

A score-only local grading service and manual post-run finalizer now exist under `peer2paper/grading/`. The routine editor should make the validation questions visible to the scientific workflow and preserve candidate answers with evidence in stage-owned sidecars. The finalizer, not the routine, collects those sidecars and emits the final grading submission. The answer key must remain outside the workspace and must never become a routine input or artefact.

Do not let a grading result choose, retry, repair, or otherwise influence a scientific transition. Grading evaluates a sealed run after the scientific workflow has reached its terminal state.

## Service contract

The service listens on loopback and accepts one authenticated call:

```text
POST http://127.0.0.1:8765/v1/grade
```

Contracts:

- Visible question contract: `peer2paper/grading/validation-question-contract.schema.json`
- Stage observations: `peer2paper/grading/stage-observations.schema.json`
- Finalized observations: `peer2paper/grading/observations.schema.json`
- Private answer key: `peer2paper/grading/answer-key.schema.json`
- Final submission: `peer2paper/grading/grading-submission.schema.json`
- Public receipt: `peer2paper/grading/grading-receipt.schema.json`

The submission contains candidate values and evidence references. The receipt contains target IDs, pass, fail, or missing states, and point totals. It contains no expected values.

## Required key split

The current benchmark file combines questions and answers. Split it before integration:

1. Put target IDs, question text, answer types, producer stages, and evidence requirements in a routine-visible validation question contract.
2. Put expected values, comparison rules, and points in the grader-only key.
3. Store the grader-only key outside the project as `{case_id}.json`.
4. Do not expose the key directory, token file, or internal receipt directory to Codex filesystem tools.

The visible contract is not an answer leak. The workflow must know what it is being asked to produce. Expected values and scoring rules remain private.

## Existing routine map

The production routine is `peer2paper-operations/run-peer2paper-scientific-audit`.

| Step | Existing producer | Validation use |
|---|---|---|
| 1. Build compact study case | `frozen-study-case`, `dataset-maps`, `code-project` | Bind visible validation targets to exact populations, models, contrasts, and source locations. Record the analytic sample answer with evidence. |
| 2. Reproduce primary result | `reproduction-package` and checkpoints | Execute every validation target classified as a reproduction target and record candidate numerical or qualitative answers. |
| 3. Continuation decision | Caller decision only | Do not grade here. A partial primary reproduction may still contain valid benchmark observations. |
| 4 and 5. Sensitivity and research | Robustness and literature artefacts | Use only for questions explicitly assigned to these stages by the visible question contract. Do not move reproduction questions here to avoid the frozen-target rules. |
| 6. Verification | `statistical-verification`, `source-verification`, `adjudication-package` | Independently confirm observations when the validation contract requires verification. Preserve disagreements rather than replacing the original answer silently. |
| 7. Delivery | `audit-json`, `audit-package` | Emit only questions assigned to delivery in the delivery-stage observation sidecar. Do not assemble a grading submission or call the grader. |

## Synthetic integration example

Tests use a fictional orchard-sensor case that is unrelated to any benchmark or
published study. Its visible contract assigns an event-count target to the study
case stage and synthetic moisture, slope, and trend targets to reproduction.
Only targets backed by sealed artifacts are submitted. Unproduced targets remain
`missing`; they are never filled from a hidden answer key.

## Routine inputs

Add one optional caller input for validation runs:

```text
validation-question-contract
```

It is a JSON object conforming to `validation-question-contract.schema.json`. Production audits without this input must behave exactly as they do today and must not produce or request a grade.

Do not add any of these as routine inputs:

- Answer-key path or content
- Expected values
- Comparison rules or points
- Grader token
- Internal receipt path

The grader token belongs to the manual finalizer or a future narrow Toone terminal hook, not the agent prompt or routine artefact graph.

## New routine artefacts

Each relevant child routine owns at most one observation sidecar. Declare only
the sidecars needed by stages that can produce visible targets:

| Artefact | Path | Producer |
|---|---|---|
| Study-case observations | `project://peer2paper/audits/{runId}/study-case/validation-observations.json` | The final study-case step. |
| Reproduction observations | `project://peer2paper/audits/{runId}/reproduction/validation-observations.json` | The reproduction packaging step. |
| Robustness observations | `project://peer2paper/audits/{runId}/robustness/validation-observations.json` | The robustness packaging step, only when assigned visible targets. |
| Research observations | `project://peer2paper/audits/{runId}/research/validation-observations.json` | The research packaging step, only when assigned visible targets. |
| Verification observations | `project://peer2paper/audits/{runId}/verification/validation-observations.json` | The verification packaging step, only when assigned visible targets. |
| Delivery observations | `project://peer2paper/audits/{runId}/delivery/validation-observations.json` | The delivery packaging step, only when assigned visible targets. |

The routine does not declare the consolidated observations, submission, or
receipt as Schema 2 artefacts. The post-run finalizer owns these files:

| Finalizer output | Path |
|---|---|
| Consolidated observations | `project://peer2paper/audits/{runId}/validation/observations.json` |
| Final grading submission | `project://peer2paper/audits/{runId}/validation/grading-submission.json` |
| Public grading receipt | `project://peer2paper/audits/{runId}/validation/grading-receipt.json` |

A routine-produced stage sidecar follows this shape. The finalizer adds
`producer_stage` to each consolidated observation:

```json
{
  "schema_version": "1.0.0",
  "case_id": "visible-case-id",
  "run_id": "native-run-id",
  "producer_stage": "reproduction",
  "observations": [
    {
      "target_id": "visible-target-id",
      "value": 0.73142,
      "evidence": [
        {
          "artifact_uri": "project://peer2paper/audits/run-id/reproduction/reproduction-package.json",
          "artifact_sha256": "lowercase-sha256",
          "json_pointer": "/synthetic_metrics/mean_moisture"
        }
      ]
    }
  ]
}
```

Do not include reported and reproduced values under the same target ID. The visible question contract must say which one answers the question.

## Consumption boundary

Implemented manual integration:

1. Relevant routine stages emit and seal their own `validation-observations.json` sidecars.
2. The routine finishes with its existing scientific terminal outcome.
3. Run `peer2paper/grading/finalize.py` with the run root, visible question contract, and terminal outcome.
4. The finalizer verifies every evidence file, SHA-256, JSON Pointer, value, answer type, case, run, and producer stage.
5. The finalizer writes consolidated `observations.json` and `grading-submission.json`.
6. The finalizer calls the loopback grader and writes the public response to `grading-receipt.json`.
7. The validation UI or designated reviewer reads the public receipt.

This leaves the scientific transition graph unchanged and prevents score feedback from affecting the audit.

Do not add a grading step to the routine. The manual command covers early terminal outcomes without replacing meaningful scientific outcomes such as `not_executable` or `audit_completed_with_limits`. A future automatic Toone terminal hook may call the same finalizer logic and should supply the runtime-owned terminal outcome directly.

## Early terminal outcomes

The Verified Audit Score awards credit for a correct blocked classification, so grading cannot depend exclusively on Step 7 completing. The manual finalizer assembles a partial submission from whichever stage sidecars exist when a run stops after:

- `case_not_assessable`
- `not_executable`
- `not_assessable`
- `stop_after_reproduction`
- `operationally_blocked`
- `sensitivity_blocked`

The visible validation contract should include an outcome-classification target when blocked classifications are being scored. Missing numerical answers remain missing and should not be manufactured.

This is another reason to keep finalization outside the routine rather than add transitions from every terminal branch.

## Idempotency and retry rules

- Use one stable `submission_id` for the final submission.
- Use the native routine `runId` as `run_id`; agents must not invent a replacement.
- An identical retry returns the original receipt.
- A changed submission for the same case and run is rejected with HTTP 409.
- Never create a new run ID to retry a failed grade.
- A failed or missing target does not trigger a scientific rerun.

## Acceptance checks for the routine editor

- A normal production audit without a validation contract is unchanged.
- Validation questions are visible, while expected values and comparison rules are absent from all routine inputs and artefacts.
- Every submitted value has an artefact URI, SHA-256, and JSON Pointer when the evidence is JSON.
- Every stage sidecar has one declared routine producer and conforms to `stage-observations.schema.json`.
- The grading submission is created only by the post-run finalizer from verified sealed scientific artefacts.
- No scientific transition reads `grading-receipt.json`.
- Early terminal runs can still be finalized into a partial submission.
- Missing targets remain missing.
- The local service tests pass with `python3 -m unittest discover -s peer2paper/grading/tests -v`.
- The organization routine linter passes after routine edits.

## Files implemented in this change

- `peer2paper/grading/service.py`
- `peer2paper/grading/finalize.py`
- `peer2paper/grading/README.md`
- `peer2paper/grading/answer-key.schema.json`
- `peer2paper/grading/validation-question-contract.schema.json`
- `peer2paper/grading/stage-observations.schema.json`
- `peer2paper/grading/observations.schema.json`
- `peer2paper/grading/grading-submission.schema.json`
- `peer2paper/grading/grading-receipt.schema.json`
- `peer2paper/grading/tests/test_service.py`
- `peer2paper/grading/tests/test_finalizer.py`

The corresponding Peer2Paper routine definitions under `.toone/organization/`
must use the same artifact paths and stage ownership.
