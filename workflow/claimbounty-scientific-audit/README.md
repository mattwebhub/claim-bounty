# ClaimBounty Scientific Audit Workflow

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Installable public local-workflow projection

This package projects the current scientific-audit routine family without internal organization configuration or case evidence.

## Contents

- `workflow.json` declares the parent routine, six ordered child routines, resource root, and local bindings.
- `routines/` contains the parent workflow and six child workflow definitions.
- `project-template/claimbounty/config/` contains audit schemas and the HTML report template.
- `project-template/claimbounty/execution/` contains the execution-manifest schema and local runner.
- `trajectories/` contains the sanitized representative projection for seven routines, nineteen current agents, and 47 ordered public steps.
- `manifest.json` and `MANIFEST.sha256` are generated integrity records.

## Verify

From the repository root:

```sh
node scripts/generate-public-manifests.mjs --check
(cd workflow/claimbounty-scientific-audit && sha256sum -c MANIFEST.sha256)
```

On macOS, use `shasum -a 256 -c MANIFEST.sha256` for the second command.

## Install

Use the latest [Toone release](https://trytoone.com). Create a project by selecting an empty directory, create `claimbounty/input/case-bundle/` inside it, and place the manuscript, data, code, supplementary materials, preregistration, environment files, and authorized supporting sources there. Open **Explore Workflows**, select **micro1/ClaimBounty**, and import it. Bind `case-bundle` to `claimbounty/input/case-bundle/`, review the remaining inputs, and run the parent workflow.

For repository fallback import, import `workflow.json` and every workflow definition as one package, then copy `project-template/claimbounty` to the local project root so `project://claimbounty/...` paths resolve. The fallback still requires the same `case-bundle` directory and dispatch documents.

The public package has not yet passed a clean import and end-to-end execution check. The `micro1/ClaimBounty` listing and import must be verified against the release build before publication.

## Regeneration boundary

`scripts/sync-public-workflow.mjs` copies an explicit allowlist from the current routine and safe project-resource sources. It excludes the general organization orchestrator, organization rosters, audit runs, study cases, answer keys, screenshots, and other internal files.

`scripts/generate-public-trajectories.mjs` separately constructs representative trajectory records from a reviewed summary allowlist. It does not copy raw messages, diagnostics, audit files, browser material, or research evidence. Predecessor records are labeled and do not prove a clean current-revision end-to-end run.
