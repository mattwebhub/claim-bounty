# Human Baseline Estimate

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Estimated qualified-human effort for the frozen benchmark scope

The base estimate is **56 active person-hours** for one qualified human to complete the frozen scientific-audit scope. The low and high scenarios are 21 and 124 active person-hours. These are planning estimates, not observed duration, monetary cost, a human quality score, or evidence of a speedup. Unattended computation and waiting time are excluded.

## Work-package estimate

| Work package                  | Task                                                            |    Low |   Base |    High |
| ----------------------------- | --------------------------------------------------------------- | -----: | -----: | ------: |
| Claim map                     | Read manuscript and map target claim                            |      3 |      5 |       8 |
| Inventory                     | Inventory evidence and inspect data and code                    |      2 |      4 |       8 |
| Reproduce                     | Rebuild environment and reproduce result                        |      3 |      8 |      18 |
| Robustness design             | Design robustness and sensitivity plan                          |      2 |      6 |      12 |
| Robustness execution          | Execute and interpret robustness suite                          |      4 |     14 |      32 |
| Verify                        | Verify sources and cross-artifact consistency                   |      2 |      6 |      14 |
| Adjudicate                    | Adjudicate discrepancies and draft corrections                  |      2 |      5 |      12 |
| Report                        | Produce report, evidence table, provenance, and quality control |      3 |      8 |      20 |
| **Total active person-hours** |                                                                 | **21** | **56** | **124** |

## Scenario assumptions

- **Low:** The replication package is complete, documented, compatible with the available environment, and produces the target result with few discrepancies. The robustness plan is narrow and no substantial repair is needed.
- **Base:** One qualified researcher encounters ordinary environment reconstruction, source checking, analysis decisions, discrepancy handling, and reporting work across the full frozen scope.
- **High:** Dependency repair, ambiguous methods, restricted or difficult data access, material discrepancies, and an extensive robustness suite require repeated investigation and correction.

The scope covers claim mapping, package inspection, environment reconstruction, exact reproduction, robustness analysis, cross-source verification, discrepancy adjudication, corrections, reporting, provenance, and quality control for the same frozen task.

## Accounting rules and exclusions

Each row is an additive work package. A task is assigned to the row that owns its output so the same labor is not counted again in another row. Work can occur in parallel, but this estimate reports one person's active effort, so parallel scheduling does not reduce the total person-hours. When an activity spans two packages, its time belongs to the package where the resulting evidence or decision becomes final.

The estimate excludes unattended compute time, scheduler waits, blocked time, data-access delays, procurement, institutional approvals, publication administration, and time contributed by additional specialists. It also excludes work outside the frozen claim and requested report. Cost is not inferred from person-hours.

The largest sensitivity drivers are environment and dependency repair, data and licence access, ambiguity in the reported methods, the number and cost of robustness specifications, the number of discrepancies requiring adjudication, and the completeness of the submitted package. These drivers explain the wide low-to-high range.

## Interpretation boundary

The literature below anchors task complexity, reproducibility practice, and plausible effort categories. It does not provide an empirical duration for this exact case or validate the three scenario totals. A measured human comparison would require a future observed, blinded human run scored under the frozen rubric.

The 56-hour estimate must not be divided by either the comparator duration or the unscored ClaimBounty exploratory-attempt duration to claim speedup. The current evaluation contains one frozen case. The ClaimBounty attempt failed exact prompt identity and introduced out-of-contract case-specific values before scientific execution.

## Sources

All sources were accessed on 2026-08-31.

1. Aczel, B., Szaszi, B., and Holcombe, A. O. (2021). [A billion-dollar donation: estimating the cost of researchers' time spent on peer review](https://doi.org/10.1186/s41073-021-00118-2). _Research Integrity and Peer Review_, 6, article 14.
2. Ware, M., and Publishing Research Consortium (2016). [Peer Review Survey 2015](https://www.elsevier.com/__data/assets/pdf_file/0007/655756/PRC-peer-review-survey-report-Final-2016-05-19.pdf). Mark Ware Consulting, May 2016.
3. Fišar, M., Greiner, B., Huber, C., Katok, E., Ozkes, A. I., and the Management Science Reproducibility Collaboration (2024). [Reproducibility in Management Science](https://doi.org/10.1287/mnsc.2023.03556). _Management Science_, 70(3), 1343–1356.
4. Colliard, J.-E., Hurlin, C., and Pérignon, C. (2022). [The Economics of Research Reproducibility](https://www.cascad.tech/download-article-front/110). Working paper.
5. Krafczyk, M. S., Shi, A., Bhaskar, A., Marinov, D., and Stodden, V. (2021). [Learning from reproducing computational results: introducing three principles and the Reproduction Package](https://doi.org/10.1098/rsta.2020.0069). _Philosophical Transactions of the Royal Society A_, 379, 20200069.
6. Hoogeveen, S. et al. (2023, published online 2022). [A many-analysts approach to the relation between religiosity and well-being](https://doi.org/10.1080/2153599X.2022.2070255). _Religion, Brain & Behavior_, 13(3), 237–283.
7. Brodeur, A. et al. (2026). [Reproducibility and robustness of economics and political science research](https://doi.org/10.1038/s41586-026-10251-x). _Nature_, 652, 151–156.
8. Schindler, D., Hossain, T., Spors, S., and Krüger, F. (2024). [A multilevel analysis of data quality for formal software citation](https://doi.org/10.1162/qss_a_00309). _Quantitative Science Studies_, 5(3), 637–667.
9. Economic Journal Data Editor. [Frequently Asked Questions](https://ejdataeditor.github.io/faqs.html).

The machine-readable estimate is [submission/evidence/human-estimate.json](../submission/evidence/human-estimate.json).
