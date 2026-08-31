# Peer2Paper validation plan

## Objective

Measure reproduction, scientific sensitivity, source support, replay, and researcher usefulness separately, then report one case-level score without hiding component failures.

## Dataset split

- Development set: six public cases used to debug the workflow and its fixtures.
- Locked evaluation set: at least ten untouched cases selected before development ends.
- Fresh-manuscript set: three to five manuscripts or preprints that were not used to design or debug the workflow.

Cases must not move from development into locked evaluation. Gold answers remain outside every routine-visible case bundle.

## Development order

1. SocSci case 27, Elite party cues: complete R reproduction smoke test with four known targets.
2. Steegen et al., fertility and religiosity: finite multiverse and known fragility.
3. SocSci case 23, Airbnb endorsements: Python reproduction.
4. SocSci case 3, Facebook vaccine content: expected partial and missing-data behavior.
5. Liu et al., financial benefits: stable open-world control.
6. Luttrell et al., ambivalence and attitude stability: sensitive open-world case.

## Platform execution order

Schema 2 child routines are callable only through their parent. Dispatch the production parent routine for every case:

1. Build and freeze the study case.
2. Reproduce the original result.
3. Run stress testing and methods/evidence research in parallel.
4. Verify and adjudicate findings.
5. If verification requests a correction, rerun the affected branch for no more than two correction rounds.
6. Assemble the final HTML, PDF, JSON, replay package, and manifest.

Do not try to dispatch a child routine directly. Component checks use frozen fixtures and independent scoring outside the production scheduler.

## Case scoring

Each complete case receives a Verified Audit Score from 0 to 10:

- Reproduction or correct blocked classification: 0 to 2.
- Scientifically valid sensitivity findings: 0 to 4.
- Methods and source support checks: 0 to 2.
- Clean replay from a fresh environment: 0 to 2.

A fabricated or seriously invalid conclusion caps the total at 5.

Record runtime, cost, human minutes, target reproduction rate, valid and invalid findings, search coverage, correction rounds, and clean replay success as secondary measures.

## First dispatch gate

Dispatch development case 1 only after:

- paper, data, code, supplements, and preregistration are present;
- source hashes and authority are recorded;
- the target claim and estimand are frozen;
- the hidden answer key is outside the case bundle;
- the local R runtime is available;
- required R dependencies are either pinned in the execution environment or installation is allowed before network-isolated execution;
- audit, scientific, and execution policies parse as JSON;
- the case bundle contains no gold answers.

