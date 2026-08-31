# Reproduce the Public Projection

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Package verification and local scientific-workflow installation

## Verify repository packages

```sh
node scripts/generate-public-manifests.mjs --check
scripts/check-public-release
```

Each package has a `manifest.json` containing relative paths, sizes, and SHA-256 values. `MANIFEST.sha256` covers every payload file and the manifest itself.

To verify an authorized application export before ZIP parsing or extraction, use the expected whole-archive SHA-256 value supplied by the trusted download response:

```sh
make verify-export EXPORT_SHA256=<64-lowercase-hex>
```

The command reads `exports/claimbounty-export.zip` and rejects a missing or malformed 64-character lowercase hexadecimal digest.

## Install the local workflow

This path requires macOS, the latest [Toone release](https://trytoone.com), one supported coding-agent client, and the R runtime used by the included example:

- Codex: follow the [official Codex CLI guide](https://learn.chatgpt.com/docs/codex/cli). On macOS or Linux, install with `curl -fsSL https://chatgpt.com/codex/install.sh | sh`, run `codex`, and sign in with ChatGPT.
- Claude Code: follow the [official Anthropic setup page](https://code.claude.com/docs/en/getting-started). No local command is copied here because upstream setup may change.
- R: install the current R release from [The R Project](https://www.r-project.org/). The included author script loads `lme4`, `DHARMa`, and `car`.

The intended Toone path is:

1. Open Toone and create a project by selecting an empty directory.
2. Create `claimbounty/input/case-bundle/` inside the selected project directory.
3. For the included demonstration, copy the contents of `examples/moura-2023-heliconius-exp1/` into that directory. The example is **Moura et al. 2023, Heliconius Experiment 1**. For another case, place the manuscript, data, code, supplementary materials, preregistration, environment files, and authorized supporting sources there.
4. Open **Explore Workflows**, select **micro1/ClaimBounty**, and import it. The workflow export installs the parent and child workflows, schemas, templates, and local runner.
5. Bind the `case-bundle` input to `claimbounty/input/case-bundle/`, review the remaining inputs, and run the parent workflow.

Verify the `micro1/ClaimBounty` listing and import against the release build before publication. If it is unavailable, use this repository package as the fallback:

1. Verify `workflow/claimbounty-scientific-audit/MANIFEST.sha256` from the repository root.
2. Import `workflow/claimbounty-scientific-audit/workflow.json` with all files under `routines/` as one package.
3. Copy `workflow/claimbounty-scientific-audit/project-template/claimbounty` into the local project root so the `project://claimbounty/...` bindings resolve.
4. Open the imported project and select the parent workflow.
5. Bind `case-bundle` to `claimbounty/input/case-bundle/` and supply the three dispatch documents required by the parent workflow.
6. Confirm Python 3.11 or later and each study-specific runtime before dispatch.
7. Run the parent workflow.

## Run the included example from a clean checkout

The included example bundle contains the CC BY 4.0 paper, author-supplied R code, and both data files read by the Experiment 1 script. From the repository root:

```sh
mkdir -p claimbounty/input/case-bundle
cp examples/moura-2023-heliconius-exp1/* claimbounty/input/case-bundle/
cd claimbounty/input/case-bundle
Rscript -e 'install.packages(c("lme4", "DHARMa", "car"), repos="https://cloud.r-project.org")'
Rscript exp1.R > exp1-output.txt 2>&1
```

Expected source-analysis output includes fitted binomial mixed-effects model summaries and analysis-of-deviance tables. R may also write its default plot file. The scientific workflow then adds frozen inputs, reproduction records, evidence research, stress tests, independent verification, adjudication, and the reviewer package around that source analysis.

The tested local workstation had R 4.6.1. The repository application uses Node 22, pnpm 9.15.0, Go 1.26.6, Python 3.11 or later, and Docker Compose v2. Package installation and container startup dominate first-run time. On the tested workstation, the clean source script completed in 1.41 seconds after package installation and produced 228 lines of model output plus a 25 KB `Rplots.pdf`. A prior full scientific workflow took 6h49m06s. Monetary cost is unmeasured because the recorded runs used a subscription plan without a per-run invoice or rate allocation.

The export is scaffolded for local installation. A clean `micro1/ClaimBounty` import and full run still require release-build verification.

## Regenerate the public workflow projection

Maintainers with access to the organization routine source and safe project resources can run:

```sh
node scripts/sync-public-workflow.mjs \
  --routine-source <routine-source-directory> \
  --resource-source <claimbounty-resource-directory>
node scripts/generate-public-manifests.mjs
```

The sync script copies an explicit allowlist. It does not copy organization configuration, audit runs, study cases, screenshots, or hidden evaluation material.

## Benchmark status

The selected candidate is Heliconius Experiment 1 from Moura et al. (2023). The article DOI is `10.1016/j.cub.2023.06.009`, and the associated data and code record DOI is `10.5281/zenodo.7985236`. The case freeze is complete and validated. The sealed ChatGPT comparator was adjudicated 1/100 with decision `invalid_incomplete`. The completed same-case ClaimBounty attempt is exploratory and unscored because the dispatch omitted the frozen prompt file's final LF byte and the comparison routine injected two case-specific values from outside the frozen participant bundle. Those values were absent from the frozen prompt and all participant files and are omitted from the public projection. Gates G2, G3, and G4 are blocked. See [BENCHMARK_PROTOCOL.md](BENCHMARK_PROTOCOL.md) and [BENCHMARK_RESULTS.md](BENCHMARK_RESULTS.md); no qualified comparison or speedup is available.
