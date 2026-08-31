# ROUTINE: Route Software Work

## Metadata
- **Routine Schema:** 2
- **Department:** software-engineering
- **Agent:** SWE Orchestrator
- **Cadence:** OnDemand
- **Duration:** ~5min
- **Proceed:** end
- **Status:** Production

## Purpose
Classify an incoming software request and return the least bureaucratic safe route based on ownership, uncertainty, dependencies, and risk.

## Pre-Requisites
Read organization://software-engineering/knowledge/team-constitution.md. Identify the target repository and apply its root and nearest scoped AGENTS.md, architecture, contract, testing, security, and contribution rules.

## Inputs
| ID | Kind | Source | Binding | Required | Type | Description |
|----|------|--------|---------|----------|------|-------------|
| input-request | parameter | Caller | - | dispatch | string | Requested software outcome, constraints, and known acceptance criteria. |
| input-target-project | parameter | Caller | - | optional | string | Target repository or project when already known. |

## Workflow
1. **Assess the request**: Identify the requested outcome, target project, acceptance criteria, affected boundaries, uncertainty, risk, and authority limits. Ask only for missing information that would materially change routing or scope.
   - id: step-assess-request
   - routine-input: input-request
   - routine-input: input-target-project
   - completion: The request has a usable outcome and an explicit boundary or a documented material ambiguity.
   - completion: Known authority-sensitive actions and risk triggers are identified.
   - execution: inherit
2. **Select the smallest safe route**: Classify the work as direct, discovery, or coordinated under the team constitution. Name the owning agent or agents, dependency order, required quality and review gates, and any decision that needs user authority. Direct work must not create extra coordination records.
   - id: step-select-route
   - routine-input: input-request
   - routine-input: input-target-project
   - output-artefact: artefact-routing-decision
   - completion: The routing decision names one of the three constitution modes.
   - completion: Every assigned boundary has one clear owner.
   - completion: Quality and review agents are included only when a constitution trigger or user request requires them.
   - completion: The routing decision is written to the declared JSON artefact.
   - execution: inherit

## Artefacts
| ID | Artifact | Path | Format |
|----|----------|------|--------|
| artefact-routing-decision | Software work routing decision | organization://software-engineering/active/{runId}/routing-decision.json | JSON |

## Escalation
Escalate when the request requires destructive data operations, production deployment, secret access, external publication or messaging, commits, pushes, or a material scope decision the user has not authorized.
