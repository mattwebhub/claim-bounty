# ClaimBounty

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Public hackathon repository projection

![ClaimBounty fox and loupe](apps/web/public/claimbounty-fox-loupe-icon.png)

ClaimBounty gives researchers an independent scientific audit before a paper reaches reviewers. The hosted application accepts a claim and private source material, an administrator freezes an authorized export, and the scientific audit runs locally with [Toone](https://trytoone.com).

This repository is a working submission scaffold. The deadline-constrained evaluation contains one frozen Heliconius case, although the challenge recommends 10 or more cases when feasible. The one-shot ChatGPT comparator is sealed and adjudicated 1/100 with decision `invalid_incomplete`; Alpha scored it 1/100 and Beta 2/100, with no cap applied. The completed same-case ClaimBounty attempt is exploratory and unscored because it failed exact prompt identity and injected two out-of-contract case-specific values from outside the frozen participant bundle. No qualified head-to-head comparison, general performance claim, or speedup is published here.

## Intended user

ClaimBounty is for empirical researchers, authors, and research teams who want to challenge a focal quantitative claim before submission or a consequential review decision.

## Current bottleneck

The work spans assembling the paper, data, and code; reproducing the focal result; challenging defensible analytical choices; reconciling documentary and external evidence; and turning discrepancies into traceable corrections. The [experiment ledger](IMPROVEMENT_CHANGELOG.md) records where those handoffs succeeded, failed, or remain pending.

## Practical value

The intended value is earlier evidence-backed correction through a bounded hosted-to-local handoff. The hosted application freezes an authorized, digest-bound package; the local workflow performs scientific work and produces review artifacts without granting hosted credentials the ability to execute research code. See the [architecture](docs/ARCHITECTURE.md) and [reproduction guide](docs/REPRODUCE.md).

## Main failure mode

The main failure mode is an incomplete or malformed evidence handoff, often compounded by environment or package gaps. The [one-shot comparator](submission/evidence/chatgpt-comparator.json) ended without the required report. The [ClaimBounty exploratory attempt](submission/evidence/claimbounty-exploratory-attempt.json) had two independent qualification failures: the dispatch omitted the frozen prompt file's final LF byte, and the comparison routine used case-specific values absent from the frozen prompt and participant files. Gates G2, G3, and G4 remained blocked before any scientific command ran. The public record omits the case-specific values.

## Hot take

Analysis that never becomes a traceable reviewer report is missing work, even when intermediate computation occurred. The baseline's progress summaries did not satisfy the requested deliverable, and a run with contaminated or non-identical inputs cannot qualify as a comparison. The [Improvement Changelog](IMPROVEMENT_CHANGELOG.md) keeps these failures visible without claiming a qualified comparison.

## Five-minute reviewer path

1. Watch the [4:54 solution video](submission/recordings/claimbounty-submission-video.mp4).
2. Open the [reviewer guide](submission/reviewer/README.md).
3. Read the [submission status](docs/SUBMISSION.md), [benchmark status](docs/BENCHMARK_RESULTS.md), [human baseline estimate](docs/HUMAN_BASELINE.md), [usage and cost disclosure](docs/USAGE_AND_COST.md), and [limitations](docs/LIMITATIONS.md).
4. Inspect the [installable workflow package](workflow/claimbounty-scientific-audit/README.md) and its generated integrity manifest.
5. For a visual-only web preview, run:

   ```sh
   pnpm install --frozen-lockfile
   pnpm --filter @micro1/web build
   pnpm --filter @micro1/web preview
   ```

The visual preview does not start the API, database, object storage, malware scanner, or worker. Use the hosted setup below to review the end-to-end application boundary. The planned demo sequence is in [docs/VIDEO_SCRIPT.md](docs/VIDEO_SCRIPT.md).

## Hosted web application setup

Use Docker and the ClaimBounty Compose profile to run the web application, API, database, object storage, scanner, mail sandbox, and worker. Follow [docs/SETUP.md](docs/SETUP.md) for prerequisites, local environment handling, startup, and shutdown.

The hosted application ends at an authorized, digest-bound export. It does not run the scientific audit or publish a finding.

## Local Toone scientific-workflow setup

This path is separate from the hosted application. The scientific workflow requires macOS, the latest [Toone release](https://trytoone.com), and one supported coding-agent client: Codex or Claude Code. It also requires Python 3.11 or later and the language runtimes required by the study. The included **Moura et al. 2023, Heliconius Experiment 1** demonstration uses R and is stored under `examples/moura-2023-heliconius-exp1/`.

In Toone, create a project by selecting an empty directory. Inside that project directory, place the manuscript, data, code, supplementary materials, preregistration, environment files, and authorized supporting sources under `claimbounty/input/case-bundle/`. Open **Explore Workflows**, select **micro1/ClaimBounty**, and import it. The workflow export installs the parent workflow, its child workflows, schemas, templates, and local runner. When prompted, bind `case-bundle` to `claimbounty/input/case-bundle/`, review the inputs, and run the parent workflow. Follow [docs/REPRODUCE.md](docs/REPRODUCE.md) for the full path and repository-package fallback.

For Codex on macOS or Linux, follow the [official Codex CLI guide](https://learn.chatgpt.com/docs/codex/cli): install with `curl -fsSL https://chatgpt.com/codex/install.sh | sh`, run `codex`, and sign in with ChatGPT. For Claude Code, use the [official Anthropic setup page](https://code.claude.com/docs/en/getting-started).

## Contributor gates

Run the smallest gate that matches the change:

```sh
make check-fast
make public-release
make test-system
make check-ci
```

`make check-fast` covers local static and contract checks. `make public-release` checks the public projection and package manifests. `make test-system` requires Docker. `make check-ci` is the full contributor gate and may require more time and disk.

## Repository map

| Path                                    | Purpose                                                          |
| --------------------------------------- | ---------------------------------------------------------------- |
| `apps/web`                              | Claim intake and administration interface                        |
| `apps/api`                              | Hosted application API and worker processes                      |
| `contracts`                             | Public HTTP and data contracts                                   |
| `infra`                                 | Compose, Kubernetes, and object-storage configuration            |
| `examples/moura-2023-heliconius-exp1`   | Included CC BY 4.0 paper, R code, and data for the demo          |
| `workflow/claimbounty-scientific-audit` | Installable local scientific-workflow projection                 |
| `submission/reviewer`                   | Short reviewer path and generated manifest                       |
| `submission/recordings`                 | Final solution video and screened source footage                 |
| `submission/evidence`                   | Public benchmark, timing, usage, cost, and evidence status       |
| `docs`                                  | Setup, architecture, benchmark, disclosure, and submission notes |

## Evidence boundary

This public projection excludes the prior 218 MiB assessment bundle, the `dev-01` case, hidden answer keys, internal organization configuration, absolute workstation paths, restricted screenshots, raw browser traces, authentication or session metadata, raw exploratory run identifiers, grader-only material, and customer data. The sealed comparator and exploratory-attempt records contain only cleared facts. See [docs/BENCHMARK_PROTOCOL.md](docs/BENCHMARK_PROTOCOL.md).

## Security and licensing

Read [SECURITY.md](SECURITY.md) before reporting a vulnerability or handling research data. Dependency and asset attribution is recorded in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and [docs/ASSET_PROVENANCE.md](docs/ASSET_PROVENANCE.md). Repository licensing is defined by [LICENSE](LICENSE).
