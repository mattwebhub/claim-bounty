# Peer2Paper

> **Status**: In Development | **Updated**: 2026-08-31 | **Scope**: Peer2Paper product and scientific-audit workflow

![Peer2Paper fox and loupe](apps/web/public/claimbounty-fox-loupe-icon.png)

[Peer2Paper](https://peer2paper.com) turns a quantitative claim and its supporting research bundle into an independent, evidence-backed scientific audit. It brings the paper, data, code, environment, and supporting sources into one traceable workflow, then produces a decision package a researcher can inspect, share, and act on.

The product combines a secure web application for case intake and administration with a local [Toone](https://trytoone.com) workflow for scientific execution. Research code runs locally; the hosted service handles authorized intake, validation, and handoff.

## Who it is for

Peer2Paper is built for empirical researchers, authors, research teams, journals, and organizations that need to challenge an important quantitative claim before publication or a consequential decision.

## The problem

A serious scientific review rarely stops at rerunning one script. A reviewer must locate the exact claim, reconstruct its analysis contract, reproduce the reported result, test defensible alternatives, reconcile the manuscript with code and data, inspect supporting literature, and turn every discrepancy into an actionable correction.

Today that work is fragmented across files, tools, people, and long-running agent sessions. Context is lost between stages, failures are difficult to resume, and the final report often obscures how each conclusion was reached.

## How Peer2Paper works

1. **Create a case.** Upload the manuscript, data, code, supplements, environment files, and authorized supporting sources through the Peer2Paper intake application.
2. **Define the audit.** Identify the exact target claim and provide the permissions, privacy rules, scientific policy, and execution policy that govern the work.
3. **Freeze the handoff.** An administrator validates the request and prepares an immutable, digest-bound export for local execution.
4. **Run the scientific audit.** Import Peer2Paper from **Explore Workflows** using its current compatibility listing, `micro1/ClaimBounty`, and run the parent workflow. Seven coordinated routines, 19 agents, and 47 ordered steps handle case construction, reproduction, sensitivity analysis, evidence research, independent verification, adjudication, and delivery.
5. **Review the decision package.** Inspect the verdict, supporting evidence, robustness results, consistency findings, prioritized corrections, reproduction record, and provenance.

## What the product delivers

Each completed audit is designed to produce one coherent decision package containing:

- **Verdict** — the bounded conclusion for the target claim and the evidence supporting it.
- **Reproduction evidence** — commands, environments, exit states, logs, numeric comparisons, tolerances, and artifact hashes.
- **Robustness summary** — how the conclusion changes across defensible data, model, covariate, and inference choices.
- **Consistency findings** — agreement or conflict across the manuscript, data, code, figures, tables, citations, and reported values.
- **Prioritized corrections** — actionable changes ordered by consequence, evidence, and effort.
- **Provenance** — source identifiers, file digests, workflow revisions, model usage, reviewer actions, and adjudication history.

The goal is not to replace scientific judgment. It is to make the evidence behind that judgment complete, inspectable, and reproducible.

## Product walkthrough

Watch the [Peer2Paper product walkthrough](submission/recordings/claimbounty-submission-video.mp4) for a complete tour of the intake experience, workflow installation, execution model, and resulting audit package.

For a deeper visual overview, open the [product guide](submission/reviewer/reviewer-guide.pdf).

## Product architecture

Peer2Paper has three deliberate boundaries:

| Surface                   | Responsibility                                                                                              |
| ------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Researcher intake         | Creates a case and uploads the authorized academic bundle.                                                  |
| Administration            | Reviews cases, validates policy and permissions, and prepares digest-bound exports.                         |
| Local scientific workflow | Runs research code and coordinated agents inside the user's Toone project, then produces the audit package. |

The Go API owns identity, case metadata, policy validation, storage coordination, and export handoffs. The React application provides the public intake and administration interfaces. PostgreSQL, object storage, malware scanning, and background workers support the hosted boundary. Scientific execution remains local so private research material and arbitrary research code do not need to run inside the hosted service.

Read the full [architecture](docs/ARCHITECTURE.md) and [security policy](SECURITY.md).

## Run the hosted application

The local product stack uses Docker Compose to start the web application, API, PostgreSQL, object storage, malware scanner, mail sandbox, and worker.

```sh
cp .env.example .env
docker compose --profile claimbounty up --build
```

Follow [docs/SETUP.md](docs/SETUP.md) for prerequisites, environment configuration, service URLs, and shutdown instructions.

## Run the scientific workflow

The scientific workflow requires:

- macOS and the latest [Toone release](https://trytoone.com)
- Codex or Claude Code configured as the coding-agent client
- Python 3.11 or later
- the language runtimes required by the study, such as R

To try the included example:

1. Create an empty project in Toone.
2. Copy the included **Moura et al. 2023, Heliconius Experiment 1** bundle from `examples/moura-2023-heliconius-exp1/` into the project.
3. Open **Explore Workflows**, select Peer2Paper under its current compatibility listing, **micro1/ClaimBounty**, and add it to the workspace.
4. Bind `case-bundle` to the example directory and provide the required audit, scientific, and execution policies.
5. Review the bindings and run the parent Peer2Paper workflow.

The export installs the complete workflow graph, schemas, templates, and local runner. Follow the [reproduction guide](docs/REPRODUCE.md) for exact commands and the repository-package fallback.

## Development

Install the JavaScript workspace and run the fast local gate:

```sh
pnpm install --frozen-lockfile
make check-fast
```

Useful validation targets:

```sh
make public-release
make test-system
make check-ci
```

`make test-system` requires Docker. `make check-ci` runs the complete deterministic contributor gate and may require additional time and disk space.

The [Improvement Changelog](IMPROVEMENT_CHANGELOG.md) records the major product and workflow iterations together with the evidence that motivated each change.

## Repository map

| Path                                    | Purpose                                                      |
| --------------------------------------- | ------------------------------------------------------------ |
| `apps/web`                              | Researcher intake and administration interface               |
| `apps/api`                              | Go API and background worker processes                       |
| `contracts`                             | HTTP and data contracts                                      |
| `infra`                                 | Compose, Kubernetes, and object-storage configuration        |
| `examples/moura-2023-heliconius-exp1`   | Licensed example paper, R code, and data                     |
| `workflow/claimbounty-scientific-audit` | Installable Peer2Paper workflow package                      |
| `submission/recordings`                 | Product walkthrough and screened source footage              |
| `submission/reviewer`                   | Product guide and generated integrity manifest               |
| `submission/evidence`                   | Public engineering and evaluation evidence                   |
| `docs`                                  | Setup, architecture, reproduction, and product documentation |

## Data handling

The public repository contains no customer research data, credentials, session metadata, internal organization configuration, hidden answer keys, or restricted case material. Uploaded research bundles are governed by explicit permissions, privacy classification, retention rules, source immutability, and release policy.

Read [SECURITY.md](SECURITY.md) before reporting a vulnerability or handling research data. Dependency and asset attribution is recorded in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and [docs/ASSET_PROVENANCE.md](docs/ASSET_PROVENANCE.md). Repository licensing is defined by [LICENSE](LICENSE).
