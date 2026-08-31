# Peer2Paper Local Grading Service

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Local score-only validation boundary

This service grades final Peer2Paper validation submissions without exposing grader-only expected values to audit agents. It uses Python's standard library, listens only on a loopback interface, and records one immutable receipt per case and run.

## Boundary

The service expects three private locations outside the project workspace:

- An answer-key directory containing one `{case_id}.json` file per case.
- A receipt directory owned by the grading process.
- A bearer-token file readable by the grading process and the narrow Toone connector that calls it.

Secure startup rejects project-local or group/world-readable key, receipt, and token paths. `--allow-insecure-development` exists for service development only and must not be used for locked evaluations.

The service does not accept answer-key paths in requests. It does not return expected values, answer-key paths, questions, or descriptive hints. A case and run ID can be graded only once; an identical retry is idempotent, while a changed retry is rejected.

## Files

- `service.py`: Loopback HTTP service and grading engine.
- `finalize.py`: Manual post-run bridge that verifies stage observations, seals a submission, calls the grader, and saves the public receipt.
- `answer-key.schema.json`: Private answer-key contract.
- `stage-observations.schema.json`: Routine-produced per-stage observation contract.
- `observations.schema.json`: Finalized cross-stage observation contract.
- `grading-submission.schema.json`: Candidate-answer submission contract.
- `grading-receipt.schema.json`: Public score-only response contract.
- `validation-question-contract.schema.json`: Visible questions and target IDs without answers or scoring rules.
- `tests/test_service.py`: Grader contract and disclosure tests.
- `tests/test_finalizer.py`: Simulated terminal-run, evidence-verification, idempotency, and end-to-end tests.

## Private layout

The grader account should own a directory outside the repository:

```text
/private/path/peer2paper-grader/
├── keys/
│   └── dev-01-elite-party-cues.json
├── receipts/
└── service.token
```

Use mode `0700` for the directories and `0600` for key and token files. For a stronger local boundary, run the service under a separate macOS account and expose only its loopback endpoint or a future narrow Toone connector.

## Answer-key contract

Each key is named `{case_id}.json` and follows this shape:

```json
{
  "schema_version": "1.0.0",
  "key_id": "example-v1",
  "case_id": "example-case",
  "visibility": "grader_only",
  "targets": [
    {
      "id": "primary-p-value",
      "expected": 0.012,
      "comparison": "three_decimal_places",
      "points": 1
    }
  ]
}
```

Supported comparisons are `exact`, `case_insensitive_exact`, `three_decimal_places`, and an object with `type: absolute_tolerance` plus a nonnegative `tolerance`.

## Submission contract

The service accepts `POST /v1/grade` with a bearer token and a final submission:

```json
{
  "schema_version": "1.0.0",
  "submission_id": "example-run-final",
  "case_id": "example-case",
  "run_id": "example-run",
  "phase": "final",
  "answers": [
    {
      "target_id": "primary-p-value",
      "value": 0.0118,
      "evidence": [
        {
          "artifact_uri": "project://peer2paper/audits/example-run/reproduction/reproduction-package.json",
          "artifact_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
          "json_pointer": "/comparison/reproduced_result/p_value"
        }
      ]
    }
  ]
}
```

Evidence references are recorded for traceability. Version 1 verifies their syntax and seals them into the submission hash; it does not receive filesystem authority to reopen audit artifacts.

## Stage observation contract

Each validation-aware scientific stage writes at most one sidecar named
`validation-observations.json` in its existing run directory. The sidecar is a
declared routine artefact with one stage-owned producer. It contains candidate
values only; it never contains expected values, grading comparisons, or points.

The manual finalizer searches these fixed locations:

```text
study-case/validation-observations.json
reproduction/validation-observations.json
robustness/validation-observations.json
research/validation-observations.json
verification/validation-observations.json
delivery/validation-observations.json
```

Every observation must identify a target from the visible question contract and
must point to the exact sealed JSON value that supports it. The finalizer checks
the artefact URI, SHA-256, JSON Pointer, value, answer type, case, run, and
producer stage before creating a submission.

## Start the service

```sh
python3 peer2paper/grading/service.py serve \
  --key-root /private/path/peer2paper-grader/keys \
  --receipt-root /private/path/peer2paper-grader/receipts \
  --token-file /private/path/peer2paper-grader/service.token
```

The default address is `127.0.0.1:8765`. The process refuses non-loopback hosts.

Health check:

```sh
curl http://127.0.0.1:8765/health
```

The Toone integration should hold the bearer token outside agent-visible prompts and submit the JSON body to `POST /v1/grade`. It should persist the public receipt under the run's validation outputs. The internal service receipt also records the submission and key hashes but contains no expected values.

## Finalize a terminated run

Until Toone has an automatic terminal hook, run the bridge after the routine
reaches any terminal outcome:

```sh
python3 peer2paper/grading/finalize.py \
  --run-root /path/to/project/peer2paper/audits/RUN_ID \
  --question-contract /path/to/validation-question-contract.json \
  --terminal-outcome audit_completed \
  --grader-url http://127.0.0.1:8765/v1/grade \
  --token-file /private/path/peer2paper-grader/service.token
```

The command creates immutable run-scoped outputs:

```text
validation/observations.json
validation/grading-submission.json
validation/grading-receipt.json
```

Omit both `--grader-url` and `--token-file` to validate evidence and create the
first two files without contacting the grader. An identical retry is safe. A
retry that would change a sealed observation set or submission is rejected.

## Test

```sh
python3 -m unittest discover -s peer2paper/grading/tests -v
```

## Current limits

- Authentication uses one local bearer token. A future connector can replace it with a per-run capability.
- A caller with the token can invent additional run IDs. For locked evaluations, the Toone connector should issue or validate the run ID rather than accepting an agent-selected value.
- The manual command accepts the terminal outcome from its caller. An automatic Toone hook should supply the runtime-owned terminal outcome directly.
- Filesystem separation depends on macOS account permissions or the harness sandbox. Agent instructions alone are not an access boundary.
- The service grades only final submissions so benchmark feedback cannot influence scientific stages.
