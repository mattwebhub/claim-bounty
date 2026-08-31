# Limitations

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Current public ClaimBounty submission

- The frozen ChatGPT comparator ended with `truncated_or_malformed_output`; its required report and all eight requested sections were absent. The two preserved progress summaries are not completed evidence. Alpha scored it 1/100, Beta 2/100, and adjudication set 1/100 with no cap and decision `invalid_incomplete`.
- The challenge recommends 10 or more evaluation cases when feasible, but this deadline-constrained evidence set contains one frozen Heliconius case. The measured multi-hour workflow runtime and the preparation, licence, freeze-validation, and isolated-execution requirements prevented a defensible 10-case evaluation before the deadline.
- The same-case ClaimBounty exploratory attempt used one parent invocation with no external restart and elapsed 3799.936556 seconds. Its build ended `case_frozen_with_limits`, reproduction was `not_executable`, no scientific commands or repairs ran, and five downstream stages were skipped.
- ClaimBounty qualification is blocked by two independent deviations. The dispatch omitted the frozen prompt file's final LF byte: 1,275 dispatched bytes versus 1,276 frozen bytes. Separately, the comparison routine injected two case-specific values from outside the frozen participant bundle. They were absent from the frozen prompt and every participant file and are omitted from the public projection. This out-of-contract contamination and the input-identity failure leave gates G2, G3, and G4 `BLOCKED`; the attempt remains exploratory and unscored and cannot support a qualified head-to-head result or speedup.
- Any future protocol-conformant result would still apply only to the frozen case. It could not establish general performance, human-quality superiority, or a broad speedup.
- Chat mode exposed only the visible setting High and no model name. Reusing the separate Work-mode model label for this comparator would be unsupported.
- The 6h49m06s scientific workflow wall time is the headline runtime. The 8h12m30s parent-thread envelope includes concurrency, pauses, retries, and post-workflow work and must not replace it.
- The observed workflow runtime and the estimated 56-hour human base case, with a 21 to 124-hour range, come from different evidence classes and are not a measured comparison. The human estimate covers one qualified person's active effort; it excludes unattended computation and waiting.
- Token totals count unique CLI sessions under the audited inclusion rules. They are usage evidence, not benchmark quality evidence.
- Monetary cost is unmeasured. The Pro-plan record contains no contemporaneous allocation, invoice, API rate, or per-run charge, so neither zero cost nor a retroactive API-price estimate is supported.
- The workflow package still needs a clean local import and full-run check. The `micro1/ClaimBounty` Explore Workflows listing must be verified against the release build.
- A frozen study may require R, Python, system libraries, licensed data, or network access beyond the package prerequisites.
- The hosted P0 ends at an authorized export. It does not run scientific code, adjudicate a claim, process payment, or publish an audit.
- Demonstration email uses a local mail sandbox and is not proof of production deliverability.
- File validation and malware scanning reduce common risks but cannot prove an uploaded research bundle is safe or correct.
- The account-generation detail for the fox-and-loupe source remains pending user confirmation.
- Restricted prior evidence, internal organization configuration, hidden evaluation material, and customer data are intentionally absent from this projection.
- Restricted screenshots, raw browser traces, authentication or session metadata, and frozen-file details are intentionally absent from the comparator projection.
