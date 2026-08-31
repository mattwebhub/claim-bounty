# Benchmark Results

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Public benchmark status and timing context

The ChatGPT comparator was adjudicated 1/100 with decision `invalid_incomplete`. The ClaimBounty exploratory attempt remains unscored, so no qualified head-to-head comparison or speedup is available.

The selected case is Heliconius Experiment 1 from Moura et al. (2023). The article DOI is `10.1016/j.cub.2023.06.009`; the associated data and code record DOI is `10.5281/zenodo.7985236`.

## Evaluation sample

The challenge recommends 10 or more evaluation cases when feasible. This deadline-constrained evidence set contains **one frozen Heliconius case**. The measured multi-hour full-workflow runtime and the preparation, licence, freeze-validation, and isolated-execution requirements for each case prevented a defensible 10-case evaluation before the deadline.

Any final benchmark result will apply only to this case. It will not support a general performance claim, human-quality superiority, or a broad speedup.

## Sealed ChatGPT comparator

The one-shot comparator used standard ChatGPT Chat mode in a non-personalized Temporary Chat with the visible setting High. Chat mode exposed no model name, so the public record does not infer one. Five frozen files were staged in manifest order and all hashes were verified. The frozen prompt SHA-256 was `da014c3be55d7a37a35a57816d72adf5ce112a54d4af6c306dd2134f762f4ae2`.

One prompt was submitted with zero follow-ups, Continue actions, regenerations, edits, branches, or retries. Send was confirmed at `2026-08-31T03:52:52.532Z`; terminal state was detected at `2026-08-31T04:05:58.072Z`, after 785.539676 seconds.

The terminal classification is `truncated_or_malformed_output`. The complete preserved output consists of two progress summaries:

1. “Reviewed PDF pages, audited feeder data, and reproduced GLMM results”
2. “Computed robust binomial models, tests, and phase-by-reward effects”

The required reviewer report and all eight requested sections were absent. All missing work remains missing; the progress summaries are not treated as completed evidence. Alpha scored the attempt 1/100, Beta scored it 2/100, and adjudication set the final score to 1/100 with no cap applied and decision `invalid_incomplete`. Cost is null and unmeasured. The public machine record is `submission/evidence/chatgpt-comparator.json`.

## Same-case ClaimBounty exploratory attempt

The attempt used one parent invocation and no external restart. It ran from `2026-08-31T04:12:11.625158Z` to `2026-08-31T05:15:31.561714Z`, an elapsed 3799.936556 seconds. Build ended `case_frozen_with_limits`, reproduction ended `not_executable`, no scientific commands or repairs ran, and five downstream stages were skipped.

Two independent deviations block qualification. First, the dispatched 1,275-byte task contract omitted the frozen 1,276-byte prompt file's final LF byte. Second, the comparison routine injected two case-specific values from outside the frozen participant bundle. They were absent from the frozen prompt and every participant file. The second deviation is out-of-contract contamination, and the values themselves are omitted from the public projection. Gates G2, G3, and G4 are `BLOCKED`.

The attempt remains exploratory and unscored. It cannot support a qualified cross-system quality comparison or speedup. The public-safe machine record is `submission/evidence/claimbounty-exploratory-attempt.json`; it excludes the raw run identifier, absolute paths, restricted case files, grader-only material, raw diagnostics, and superseded internal seal hashes.

## Verified runtime and usage context

The primary scientific workflow wall time was **6h49m06s**, which is the headline runtime. The wider parent-thread activity envelope was **8h12m30s** and includes concurrency, pauses, retries, and post-workflow work. It is supporting context rather than the workflow runtime.

The separate workflow usage audit counted 109 unique CLI sessions across the primary and secondary runs and 181,802,664 total tokens on provider `openai` using model `gpt-5.6-sol`. This model label belongs only to those audited CLI sessions and does not identify the ChatGPT comparator. Cached input is already included within input, and reasoning output is already included within output.

A human comparison was estimated at **56 hours** for one qualified human on the frozen scope, with an estimated range of **21 to 124 active person-hours**. This is an estimate, not observed duration or a human quality score. Unattended computation and waiting are excluded, and no speedup is calculated. See [HUMAN_BASELINE.md](HUMAN_BASELINE.md) for the eight work packages, assumptions, exclusions, sensitivity drivers, and cited basis.

These runtime and usage records are not results for the selected benchmark and do not establish quality. Monetary cost is unmeasured; the records identify a Pro plan but contain no contemporaneous allocation, invoice, API rate, or per-run charge. No zero-dollar or retroactively priced cost is reported.

See [USAGE_AND_COST.md](USAGE_AND_COST.md) for the full accounting. The authoritative machine-readable records are `submission/evidence/benchmark-status.json`, `submission/evidence/human-estimate.json`, `submission/evidence/timing-observations.json`, and `submission/evidence/usage-and-cost.json`.

## Publication gate

The comparator's 1/100 score is published only as an `invalid_incomplete` single-case outcome. A ClaimBounty score or cross-system comparison requires a new protocol-conformant execution with byte-identical frozen inputs, no out-of-contract case information, executable reproduction, sealed artifacts, independent review, adjudication, and cleared public evidence.
