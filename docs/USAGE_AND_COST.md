# Usage and Cost Disclosure

> **Status**: Active | **Updated**: 2026-08-31 | **Scope**: Verified CLI session, token, runtime, and monetary-cost accounting

## Usage totals

The verified workflow usage audit identified two included CLI runs on provider `openai` using model `gpt-5.6-sol`. This label does not identify the separate ChatGPT Chat-mode comparator, which exposed no model name.

| Run                                              | Unique CLI sessions | Input tokens | Cached input tokens | Output tokens | Reasoning output tokens | Total tokens |
| ------------------------------------------------ | ------------------: | -----------: | ------------------: | ------------: | ----------------------: | -----------: |
| Primary `97A93515-89D6-4292-BB32-64A98F64BE20`   |                  91 |  157,669,522 |         148,934,912 |     1,697,629 |                 425,605 |  159,367,151 |
| Secondary `99AB2FDF-9953-4FA8-B499-7FE75256EC9E` |                  18 |   22,184,313 |          20,737,024 |       251,200 |                  63,907 |   22,435,513 |
| Combined                                         |                 109 |  179,853,835 |         169,671,936 |     1,948,829 |                 489,512 |  181,802,664 |

Cached input is included within input, and reasoning output is included within output. Those component values must not be added to the totals again. Each unique CLI session is counted once. Retries and correction rounds are included; repeated cumulative events and later assessment sessions are excluded.

## Runtime

The headline runtime is the primary scientific workflow wall time of **6h49m06s**. The wider parent-thread activity envelope was **8h12m30s** and includes concurrency, pauses, retries, and post-workflow work. The parent envelope is supporting operational context, not the scientific workflow runtime.

Neither duration is the sealed ChatGPT comparator duration or a same-case ClaimBounty result, and neither establishes benchmark quality or a measured comparison with human work.

## Monetary cost

Monetary cost is **unmeasured**. The audited records identify a Pro plan but contain no contemporaneous subscription allocation, invoice, API rate, or per-run dollar charge. ClaimBounty therefore does not report a zero-dollar cost and does not apply current API prices retroactively.

The authoritative machine-readable disclosure is `submission/evidence/usage-and-cost.json`.
