# Benchmark Protocol

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Public benchmark freeze, execution, and publication gates

## Selected candidate

The selected candidate is Heliconius Experiment 1 from Moura et al. (2023). The article DOI is `10.1016/j.cub.2023.06.009`; the associated data and code record DOI is `10.5281/zenodo.7985236`.

The case freeze is complete and validated. The one-shot ChatGPT comparator used five frozen files in manifest order with every hash verified. Its independent scoring and adjudication are complete. The completed same-case ClaimBounty attempt is exploratory and unscored because two independent deviations blocked qualification. The public repository does not redistribute the frozen files or their restricted manifest details.

## Evaluation sample

The challenge recommends 10 or more evaluation cases when feasible. This deadline-constrained evidence set contains **one frozen case**. The full workflow has a measured multi-hour runtime, and each additional case requires preparation, licence review, freeze validation, and isolated execution. A defensible 10-case evaluation could not be completed before the deadline.

Any result from this protocol is limited to the Heliconius case. It cannot establish general performance, human-quality superiority, or a broad speedup.

## Case-freeze requirements

The benchmark owner must record:

- Stable source identifiers, citation, retrieval date, and redistribution terms.
- The exact target claim, estimand, reported result, sample, code entry point, and numeric tolerance.
- Any approved secondary outcomes and the rule used to score them.
- A manifest and SHA-256 value for every public case file.
- A sealed answer key outside the routines and public package when blinding requires one.
- Runtime versions and an executable baseline command.

The frozen manifest is immutable for the scored run. Any correction creates a new case version.

## Execution protocol

1. Verify the frozen case manifest in a clean environment.
2. Run the original baseline before the ClaimBounty workflow.
3. Keep reviewers blind to the sealed reference outcome where the study design requires it.
4. Record machine, runtime, command, exit status, elapsed time, and artifact hashes.
5. Run the parent workflow through all six child routines.
6. Seal the result before comparison with the reference outcome.
7. Have the result reviewed and adjudicated before publication.

## One-shot comparator controls

The ChatGPT comparator used standard Chat mode in a non-personalized Temporary Chat with the visible setting High. Chat mode exposed no model name, so none is inferred. One prompt was submitted with no follow-up, Continue action, regeneration, edit, branch, or retry.

Before Send, one file-picker selector failed. The same unused session then staged all five files through the hidden file input. This pre-send staging event did not add a prompt or retry and is disclosed only to make the input procedure reproducible.

The required report and all eight requested sections are scored as missing when absent. Progress summaries are preserved as output but do not count as completed evidence.

## Observed qualification deviations

The ClaimBounty exploratory attempt used one parent invocation with no external restart. It ran from `2026-08-31T04:12:11.625158Z` to `2026-08-31T05:15:31.561714Z`, an elapsed 3799.936556 seconds. Build ended `case_frozen_with_limits`; reproduction ended `not_executable`; zero scientific commands and zero repairs ran; and five downstream stages were skipped.

Two independent deviations block qualification:

1. The dispatched task contract was 1,275 bytes while the frozen prompt file was 1,276 bytes. The dispatched form omitted the prompt file's final LF byte, so exact input identity failed.
2. The comparison routine injected two case-specific values from outside the frozen participant bundle. They were absent from the frozen prompt and every participant file, so this is out-of-contract contamination. The public projection omits the values themselves.

Gates G2, G3, and G4 are `BLOCKED`. The attempt cannot be scored or used for a qualified head-to-head comparison or speedup. A qualified rerun must use byte-identical frozen inputs and a comparison routine that contains no case-specific information absent from participant inputs.

## Measures

The result package should distinguish reproduction correctness, evidence traceability, sensitivity coverage, unsupported-claim detection, adjudicated quality, elapsed machine time, active human time, and estimated human comparison time. Estimated and observed values must not be combined into a speedup claim.

## Publication gate

The comparator score may be published with its `invalid_incomplete` decision and missing-work context. No ClaimBounty quality score, success claim, head-to-head result, or comparative speedup may be published from the protocol-deviating attempt. A future qualified result requires byte-identical frozen inputs, a contamination-free routine, executable reproduction, sealed output, independent review, adjudication, and redistribution review. Any qualified result must remain labeled as single-case evidence and must not be generalized. Public artifacts belong under `submission/evidence/`; local run artifacts remain outside this projection until cleared.
