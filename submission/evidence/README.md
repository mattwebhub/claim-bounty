# Public Evidence Package

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Benchmark status, timing context, and public-safe attempt evidence

This package contains public status records, the cleared sealed comparator record, and a public-safe record of the unscored Peer2Paper exploratory attempt. It does not contain frozen case files, restricted manifest details, an answer key, the prior assessment bundle, customer material, screenshots, raw browser traces, authentication or session metadata, raw run identifiers, or grader-only material.

- `benchmark-status.json` is the authoritative machine-readable candidate and gate status.
- `chatgpt-comparator.json` is the authoritative public-safe one-shot comparator record, including its complete preserved output, Alpha 1/100 and Beta 2/100 scores, and adjudicated 1/100 `invalid_incomplete` decision with no cap applied.
- `claimbounty-exploratory-attempt.json` records the single parent invocation, elapsed time, blocked execution, exact-input failure, and out-of-contract contamination without publishing the raw run identifier, private paths, case files, grader-only material, or superseded internal seal hashes. Its filename is retained as a historical compatibility identifier from before the Peer2Paper rebrand.
- `timing-observations.json` separates an observed internal workflow duration from an estimated human comparison.
- [human-estimate.json](human-estimate.json) records the 21/56/124 active-person-hour scenarios, eight additive work packages, exclusions, and cited basis. The readable derivation is [docs/HUMAN_BASELINE.md](../../docs/HUMAN_BASELINE.md).
- `usage-and-cost.json` is the authoritative verified CLI session, token, runtime-envelope, and monetary-cost disclosure.
- `expected-evidence.json` distinguishes completed comparator evidence, the unscored Peer2Paper attempt, and the unavailable qualified comparison.
- `manifest.json` and `MANIFEST.sha256` are generated package integrity records.

The challenge recommends 10 or more evaluation cases when feasible; this deadline-constrained package contains one frozen Heliconius case. The comparator is complete and adjudicated 1/100. Peer2Paper qualification is blocked by two independent deviations: the dispatched task contract omitted the frozen prompt file's final LF byte, and the comparison routine injected two case-specific values from outside the frozen participant bundle. Those values were absent from the frozen prompt and every participant file. The second deviation is out-of-contract contamination, and the values themselves are omitted from the public projection. Gates G2, G3, and G4 are blocked, and the attempt remains exploratory and unscored. The two attempts cannot support a qualified head-to-head comparison, general performance claim, human-quality superiority claim, or speedup.
