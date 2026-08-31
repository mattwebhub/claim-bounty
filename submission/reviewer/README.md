# Peer2Paper Product Review

> **Status**: Product Review Ready | **Updated**: 2026-08-31 | **Scope**: Five-minute public Peer2Paper review path

Start with the [13-page Peer2Paper product guide](peer2paper-product-guide.pdf). It introduces the researcher, the problem, the hosted-to-local product boundary, workflow installation, the included scientific case, and the evidence-backed decision package.

## Minute 0 to 1

Read the root [README](../../README.md) and [architecture](../../docs/ARCHITECTURE.md) to understand who Peer2Paper serves, why scientific audits require more than a paper summary, and why research execution remains local.

## Minute 1 to 2

Watch the [Peer2Paper product walkthrough](../recordings/peer2paper-product-walkthrough.mp4). It shows the product guide, Toone workflow, local execution model, and resulting review package.

## Minute 2 to 3

Inspect `workflow/claimbounty-scientific-audit/workflow.json` and its parent routine. The `claimbounty` directory and workflow names are retained as compatibility identifiers while the public product is Peer2Paper. Confirm that all six child routines and the `project://` resource template are present.

## Minute 3 to 4

Follow the [reproduction guide](../../docs/REPRODUCE.md) to install Toone, a supported coding-agent client, R, and the included Moura et al. Heliconius Experiment 1 bundle. Import Peer2Paper through its current `micro1/ClaimBounty` compatibility listing.

## Minute 4 to 5

Inspect the workflow trajectories, [human-effort estimate](../../docs/HUMAN_BASELINE.md), and final report contract. The decision package separates the verdict, reproduction evidence, robustness summary, consistency findings, prioritized corrections, provenance, and review controls.

## Validate the package

```sh
pnpm --filter @micro1/web e2e:install
make public-release
```

The install command downloads the Chromium revision pinned by the locked Playwright dependency. `make public-release` checks the HTML and PDF guide, links, accessibility, package policy, and generated integrity manifests.

Set `PEER2PAPER_PDF_CHROME` to use a specific local Chromium executable. The legacy `CLAIMBOUNTY_PDF_CHROME` variable remains available as a compatibility fallback.

The generated `manifest.json` and `MANIFEST.sha256` in this directory are the authoritative inventory and integrity record for the Peer2Paper product-guide package.
