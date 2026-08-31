# Five-Minute Reviewer Guide

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Short public ClaimBounty review path

Start with the [13-page reviewer guide](reviewer-guide.pdf). It is a Draft / In-Development guide and follows the same scored-comparator, unscored-ClaimBounty status as this package.

## Minute 0 to 1

Read the root `README.md`, [Improvement Changelog](../../IMPROVEMENT_CHANGELOG.md), [submission status](../../docs/SUBMISSION.md), and [human baseline](../../docs/HUMAN_BASELINE.md) to understand the intended user, hosted-to-local boundary, experiment decisions, current submission status, and estimated human-effort basis.

## Minute 1 to 2

Open `docs/TRAJECTORIES.md` and the hosted application screenshots or live deployment supplied by the submission owner. The repository contains no restricted screenshots.

## Minute 2 to 3

Inspect `workflow/claimbounty-scientific-audit/workflow.json` and its parent routine. Confirm that all six child routines and the `project://` resource template are present.

## Minute 3 to 4

Read the [benchmark status](../evidence/benchmark-status.json), [ChatGPT comparator](../evidence/chatgpt-comparator.json), [ClaimBounty exploratory attempt](../evidence/claimbounty-exploratory-attempt.json), [human estimate](../evidence/human-estimate.json), [timing observations](../evidence/timing-observations.json), and [usage and cost record](../evidence/usage-and-cost.json). Confirm that the comparator ended `truncated_or_malformed_output` after 785.539676 seconds; all eight requested report sections are missing; Alpha scored it 1/100, Beta 2/100, and adjudication set 1/100 with no cap and decision `invalid_incomplete`. Confirm that the ClaimBounty attempt elapsed 3799.936556 seconds and had two independent qualification deviations: a 1,275-versus-1,276-byte final-LF mismatch and injection of two case-specific values from outside the frozen participant bundle. Those values were absent from the frozen prompt and every participant file and are omitted from the public projection. G2/G3/G4 were blocked before any scientific command, and the attempt remains exploratory and unscored. No qualified head-to-head result or speedup is published. The 6h49m06s separate workflow runtime and 8h12m30s parent-thread envelope are context only; monetary cost is unmeasured.

The challenge recommends 10 or more evaluation cases when feasible, but this evidence set contains one frozen case. Treat the published outcomes as single-case evidence only, with no general performance, human-quality superiority, or broad speedup claim.

## Minute 4 to 5

Run the public package gate:

```sh
pnpm --filter @micro1/web e2e:install
make public-release
```

The install command downloads the Chromium revision pinned by the locked
Playwright dependency. `make public-release` runs the HTML overflow, link,
console, 13-sheet, and accessibility checker before the remaining package gates.
Set `CLAIMBOUNTY_PDF_CHROME` only to use a specific local Chromium executable.

The generated `manifest.json` and `MANIFEST.sha256` in this directory are the authoritative inventory and integrity record for the reviewer package.
