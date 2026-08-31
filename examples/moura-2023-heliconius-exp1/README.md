# Example Case Bundle: Moura et al. 2023, Experiment 1

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Publicly licensed ClaimBounty demonstration bundle

This folder is the included example for the ClaimBounty reproduction path. It contains the two-page paper, the authors' Experiment 1 R script, and both data files read by that script.

## Source and attribution

Priscila A. Moura, Fletcher J. Young, Monica Monllor, Marcio Z. Cardoso, and Stephen H. Montgomery, “Long-term spatial memory across large spatial scales in Heliconius butterflies,” _Current Biology_ 33 (2023), R797–R798.

- Article DOI: <https://doi.org/10.1016/j.cub.2023.06.009>
- Data and code DOI: <https://doi.org/10.5281/zenodo.7985236>
- License: Creative Commons Attribution 4.0 International

The article states that it is open access under CC BY 4.0. The Zenodo record identifies the deposited data and code as open access under CC BY 4.0. The full license text is included as `LICENSE-CC-BY-4.0.txt`.

## Included files

| File                        | Purpose                               | SHA-256                                                            |
| --------------------------- | ------------------------------------- | ------------------------------------------------------------------ |
| `moura-2023-heliconius.pdf` | Primary academic paper                | `2de6257494b23d7401e12082b1fcb9d79ac34669af04ac02d12a4ebc01f8da8b` |
| `exp1.R`                    | Author-supplied Experiment 1 analysis | `92fc0cad2274cd013d9125807c2c43b27c1b45229e3d7c7cc8471e887c723841` |
| `exp1.csv`                  | Main Experiment 1 data                | `87d4145b5babf43d026e44bf044410fad9d22450463929b3cd8f30e87fc524b3` |
| `Exp1byDay.csv`             | Experiment 1 day-level data           | `6af21960673cd3953a47754fa43b33344847f5ddc9e64793f4e76ae6c61b946b` |

## Run the source analysis

Install R, then run from this directory:

```sh
Rscript -e 'install.packages(c("lme4", "DHARMa", "car"), repos="https://cloud.r-project.org")'
Rscript exp1.R > exp1-output.txt 2>&1
```

The script fits the authors' binomial mixed-effects models, emits summaries and analysis-of-deviance results, and writes the default R plot output. With R 4.6.1 and the packages already installed, the checked source run completed in 1.41 seconds and produced 228 output lines plus a 25 KB `Rplots.pdf`. Package installation time varies by platform and may require build tools.

## Use with ClaimBounty

Copy these five files into `claimbounty/input/case-bundle/` in a new Toone project. In **Explore Workflows**, select **micro1/ClaimBounty**, bind `case-bundle` to that directory, review the request and policy documents, and run the parent workflow.
