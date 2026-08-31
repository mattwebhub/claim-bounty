# Improvement Changelog

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Public evidence-linked ClaimBounty experiment ledger

This ledger records material experiments against fixed review methods. Evidence classes remain separate: the frozen one-shot comparator, the predecessor ClaimBounty audit, the partial current-revision run, the selected protocol-deviating exploratory attempt, hosted application tests, packaging checks, and qualitative interface review do not form one comparative score.

## Experiment ledger

### Experiment 1: Frozen one-shot ChatGPT baseline

- Approach / hypothesis: Test whether standard ChatGPT Chat mode could turn the five hash-verified frozen Heliconius inputs and one frozen prompt into the required reviewer report without intervention.
- Fixed evaluation or review method: One prompt, no follow-up, Continue action, regeneration, edit, branch, or retry. Preserve the complete terminal output and score every absent report section as missing.
- Observed result: Terminal detection occurred after 785.539676 seconds with `truncated_or_malformed_output`. The complete output was two progress summaries; the reviewer report and all eight required sections were absent. Alpha scored it 1/100, Beta 2/100, and adjudication set 1/100 with no cap and decision `invalid_incomplete`.
- Keep / change / remove decision: **Keep as the frozen baseline failure.** Do not reinterpret intermediate computation as a completed report or infer a hidden model name, measured cost, or speedup. Publish the adjudicated score only with its invalid-and-incomplete context.
- Public evidence path: [sealed comparator record](submission/evidence/chatgpt-comparator.json), [benchmark results](docs/BENCHMARK_RESULTS.md), and [benchmark protocol](docs/BENCHMARK_PROTOCOL.md).

### Experiment 2: Predecessor full ClaimBounty audit

- Approach / hypothesis: Test whether evidence-gated stages, independent verification, bounded correction rounds, deterministic rendering, and replay could recover from operational defects while preserving visible limits.
- Fixed evaluation or review method: Measure scientific workflow wall time separately from the parent activity envelope; require the research, statistical, and source-evidence joins before adjudication and delivery; retain bounded error and correction records; require matching rendered outputs and a clean replay.
- Observed result: Scientific workflow wall time was 6h49m06s and the terminal outcome was `audit_completed_with_limits`. The run encountered a research completion-envelope failure, an invalid verification-before-delivery sequence, and a renderer-selection defect. Bounded recovery and correction rounds produced matching renders and a clean replay while retaining unresolved limits.
- Keep / change / remove decision: **Keep staged verification, explicit adjudication, deterministic rendering, and replay. Change the affected completion and sequencing gates. Preserve the limited outcome rather than presenting an unconditional success.**
- Public evidence path: [timing observation](submission/evidence/timing-observations.json), [parent representative trajectory](workflow/claimbounty-scientific-audit/trajectories/routines/run-claimbounty-scientific-audit.json), [research trajectory](workflow/claimbounty-scientific-audit/trajectories/routines/research-methods-and-evidence.json), [verification trajectory](workflow/claimbounty-scientific-audit/trajectories/routines/verify-and-adjudicate-findings.json), and [assembly trajectory](workflow/claimbounty-scientific-audit/trajectories/routines/assemble-final-audit.json).

### Experiment 3: Current routine revision and partial current-run evidence

- Approach / hypothesis: Make build, reproduction, repair, scientific comparison, and continuation separate checkpoints so an incomplete run cannot imply current-revision end-to-end completion.
- Fixed evaluation or review method: Project only exact current evidence for build and reproduction; preserve ordered steps, coarse statuses, supported errors, and handoffs; require the parent continuation decision before later routines; label predecessor evidence separately.
- Observed result: The current build completed with limits. Current reproduction applied one bounded repair, ran one secondary target, and ended `partial_reproduction`. Repair status remains separate from scientific comparison, and the continuation gate is explicit. No later current-revision completion is evidenced.
- Keep / change / remove decision: **Keep the explicit continuation gate and the separation between operational repair and scientific comparison. Do not claim current-revision end-to-end success.**
- Public evidence path: [current build trajectory](workflow/claimbounty-scientific-audit/trajectories/routines/build-and-freeze-study-case.json), [current reproduction trajectory](workflow/claimbounty-scientific-audit/trajectories/routines/reproduce-original-result.json), [parent routine](workflow/claimbounty-scientific-audit/routines/run-claimbounty-scientific-audit.md), and [trajectory interpretation](docs/TRAJECTORIES.md).

### Experiment 4: Contract-first hosted intake and export boundary

- Approach / hypothesis: Keep private intake, authorization, retention, scanning, and export in the hosted system while keeping scientific code execution in a separately authorized local environment.
- Fixed evaluation or review method: Validate the canonical OpenAPI and JSON schemas, check generated-client drift, exercise authorization and domain tests, and use the Compose system-test path to cover intake through digest-bound export and whole-archive verification before extraction.
- Observed result: The contract, schemas, and generated client pass the public-release contract gate. Existing transport tests assert standard `Content-Digest` behavior, and the system-test specification covers hosted intake, export download, digest comparison, and offline verification. Scientific execution remains outside the hosted boundary.
- Keep / change / remove decision: **Keep the hosted-to-local artifact boundary. Keep scientific execution out of hosted credentials and require a trusted expected archive digest before parsing.**
- Public evidence path: [OpenAPI contract](contracts/openapi.yaml), [architecture](docs/ARCHITECTURE.md), [download digest tests](apps/api/internal/transport/httpapi/claimbounty_test.go), [full-stack system-test specification](apps/web/tests/system/claimbounty-compose.system.spec.ts), and [offline command contract](contracts/README.md).

### Experiment 5: Reviewable public package

- Approach / hypothesis: A reviewer should be able to distinguish measured, estimated, predecessor, current, reconstructed, and pending evidence without access to private organization state.
- Fixed evaluation or review method: Generate deterministic SHA-256 manifests; validate usage and human-effort accounting; require seven routine and nineteen current-agent trajectory records against a restrictive schema; render and inspect the 13-page reviewer PDF; run link, path, credential, binary, size, and generated-drift checks.
- Observed result: Reviewer, evidence, workflow, and trajectory manifests verify. The public projection contains the cited 21/56/124 active-person-hour estimate, verified usage accounting, a sanitized 7-routine/19-current-agent/47-step trajectory package, and a 13-page PDF with no forbidden local annotations, page overflow, console errors, or accessibility violations.
- Keep / change / remove decision: **Keep generated manifests, evidence-class labels, schema and security validation, and the reviewer guide. Regenerate rather than hand-edit generated records or package hashes.**
- Public evidence path: [human baseline](docs/HUMAN_BASELINE.md), [usage accounting](docs/USAGE_AND_COST.md), [trajectory index](workflow/claimbounty-scientific-audit/trajectories/index.json), [trajectory sanitization policy](workflow/claimbounty-scientific-audit/trajectories/sanitization-policy.md), [reviewer guide](submission/reviewer/reviewer-guide.pdf), and [public-release check](scripts/check-public-release).

### Experiment 6: Interface direction review

- Approach / hypothesis: Compare literal, textured scientific motifs with a quieter archival image and a simple recognizable mark for the reviewer-facing interface.
- Fixed evaluation or review method: Qualitative user review of interface mockups and asset variants. This was a design review, not a scientific benchmark, timed evaluation, or scored preference study.
- Observed result: The raster-sampled ASCII and over-abstract topographic brain direction and the overly detailed tactile loupe direction were removed from the current interface. The blurred archival projection and simple fox-and-loupe mark were retained.
- Keep / change / remove decision: **Remove the rejected directions from the active UI. Keep the blurred archival projection and simple fox mark. Label the decision as qualitative user feedback.**
- Public evidence path: [current home route](apps/web/src/routes/home/home.route.tsx), [retained archival projection](apps/web/public/claimbounty-archival-projection-v3.png), [retained fox mark](apps/web/public/claimbounty-fox-loupe-icon.png), and [asset provenance](docs/ASSET_PROVENANCE.md). The rejected experiment files and the earlier Getty-derived projection remain local, ignored, and outside the public evidence projection.

### Experiment 7: Same-case ClaimBounty exploratory attempt

- Approach / hypothesis: Run the current ClaimBounty workflow on the same frozen Heliconius case used by the one-shot baseline, then apply independent scoring and adjudication.
- Fixed evaluation or review method: Use the exact frozen case and prompt contract, seal output before comparison, score missing work as missing, preserve runtime and evidence classifications, and stop scoring if input identity fails.
- Observed result: One parent invocation with no external restart elapsed 3799.936556 seconds. Build ended `case_frozen_with_limits`, reproduction ended `not_executable`, zero scientific commands and repairs ran, and five downstream stages were skipped. Qualification failed for two independent reasons: the dispatched 1,275-byte task contract omitted the frozen 1,276-byte prompt file's final LF byte, and the comparison routine injected two case-specific values from outside the frozen participant bundle. Those values were absent from the frozen prompt and every participant file. This is out-of-contract contamination; the values themselves are omitted from the public projection. Gates G2, G3, and G4 are `BLOCKED`.
- Keep / change / remove decision: **Keep as unscored exploratory failure evidence and remove it from any qualified head-to-head comparison.** Require byte-exact prompt verification and a contamination-free routine before any future scored dispatch. Do not publish a ClaimBounty quality result or speedup before independent scoring and adjudication of a qualified run.
- Public evidence path: [public-safe attempt record](submission/evidence/claimbounty-exploratory-attempt.json), [machine-readable benchmark status](submission/evidence/benchmark-status.json), [benchmark protocol](docs/BENCHMARK_PROTOCOL.md), and [submission status](docs/SUBMISSION.md).

## Main failure mode

The recurring failure mode is an incomplete, malformed, or contaminated evidence handoff across stage boundaries. The comparator stopped after intermediate progress text without producing the required report. The selected ClaimBounty attempt dispatched a task contract missing the frozen prompt's final LF byte and used a comparison routine containing case-specific values absent from participant inputs. It stopped before scientific execution. The predecessor workflow needed recovery when research completion, verification sequencing, and renderer selection did not satisfy the next stage's contract. The retained response is byte-exact input verification, contamination checks, explicit continuation gates, bounded corrections, independent joins, deterministic rendering, replay, and visible limited outcomes.

## Hot take

Analysis that never becomes a traceable reviewer report is missing work, even when intermediate computation occurred. A comparison that receives undisclosed case-specific information is also missing qualification, even if no scientific command runs. On this single case, the 785.539676-second baseline and its adjudicated 1/100 score are evidence for terminal incompleteness. The 3799.936556-second ClaimBounty attempt is evidence of two blocked qualification deviations, not a scored result. The multi-hour predecessor run shows only that staged verification and replay produced a limited audit package on different evidence. These observations do not establish general performance, a qualified same-case comparison, or speedup.

Git history remains authoritative for commit dates, authorship, and the exact content of released versions.
