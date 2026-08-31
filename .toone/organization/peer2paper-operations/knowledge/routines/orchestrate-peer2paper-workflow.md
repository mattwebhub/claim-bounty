# ROUTINE: Orchestrate Peer2Paper Workflow

## Metadata
- **Routine Schema:** 2
- **Department:** peer2paper-operations
- **Agent:** Peer2Paper Orchestrator
- **Cadence:** OnDemand
- **Proceed:** continue
- **Max Concurrency:** 1
- **Status:** Production

## Purpose
Coordinate a per-run Peer2Paper objective from intake through a clear outcome by assigning specialist work, sequencing dependencies, tracking status, escalating authority-sensitive blockers, and consolidating results.

## Inputs
| ID | Kind | Source | Binding | Required | Type | Description |
|----|------|--------|---------|----------|------|-------------|
| peer2paper-objective | parameter | caller | - | dispatch | string | The specific Peer2Paper objective this run must coordinate to a clear outcome. |
| run-context | parameter | caller | - | optional | json | Available participants, relevant facts, references, prior decisions, and other context needed for this run. |
| run-constraints | parameter | caller | - | optional | json | Explicit constraints, completion expectations, sequencing requirements, and prohibited actions for this run. |
| approval-boundaries | parameter | caller | - | optional | json | Known delegated-authority limits and actions requiring human approval or escalation. |

## Workflow
1. **Intake Objective and Build Coordination Plan**: Interpret the per-run objective and supplied context, preserve all constraints and approval boundaries, identify the workstreams and dependencies, assign each workstream to an appropriate available participant without assuming specialist judgment, and write the executable coordination plan.
   - id: intake-and-plan
   - routine-input: peer2paper-objective
   - routine-input: run-context
   - routine-input: run-constraints
   - routine-input: approval-boundaries
   - output-artefact: orchestration-plan
   - completion: The plan states the objective and measurable definition of a clear outcome.
   - completion: Every identified workstream has an owner, dependency position, expected result, and completion criterion.
   - completion: All supplied constraints and approval boundaries are represented without expansion of delegated authority.
   - completion: The orchestration plan artefact is valid JSON and contains no unresolved assignment ambiguity that can be resolved from supplied context.
   - execution: inherit
2. **Coordinate Work and Determine Run Outcome**: Coordinate the planned work in dependency order, provide participants with relevant context, track each workstream's business status and evidence, adjust safe sequencing when needed, and classify the run as completed, authority-blocked, or operationally blocked. Do not perform approval-sensitive or externally consequential actions without authorization.
   - id: coordinate-work
   - routine-input: peer2paper-objective
   - routine-input: run-context
   - routine-input: run-constraints
   - routine-input: approval-boundaries
   - input-artefact: orchestration-plan
   - output-artefact: workflow-status
   - completion: Every planned workstream has a recorded final status, result or blocker, and supporting evidence reference.
   - completion: Dependencies are satisfied before dependent work is treated as complete.
   - completion: Any safe resequencing or recovery action remains within the supplied constraints and delegated authority.
   - completion: The workflow status artefact is valid JSON and supports exactly one declared business outcome.
   - completion: Approval-sensitive, destructive, externally consequential, materially ambiguous, or out-of-authority blockers are recorded for escalation rather than executed.
   - outcome: objective_completed | All required workstreams met their completion criteria and the objective reached a clear completed outcome.
   - outcome: authority_blocked | Further progress requires human approval or authority that was not delegated for this run.
   - outcome: operationally_blocked | The objective could not complete because of a non-authority blocker, failed dependency, unavailable capability, or insufficient required context.
   - execution: inherit
3. **Consolidate and Report Clear Outcome**: Consolidate the objective, coordination plan, workstream results, evidence, decisions, blockers, escalations, and recommended next actions into one concise final report that makes the run's disposition explicit.
   - id: consolidate-outcome
   - routine-input: peer2paper-objective
   - routine-input: run-constraints
   - routine-input: approval-boundaries
   - input-artefact: orchestration-plan
   - input-artefact: workflow-status
   - output-artefact: outcome-report
   - completion: The report identifies exactly one final disposition: objective completed, authority blocked, or operationally blocked.
   - completion: The report summarizes every workstream's owner, result, and unresolved blocker, if any.
   - completion: Authority-sensitive blockers state the decision required and safe next options without implying authorization.
   - completion: The report lists important decisions, evidence references, unresolved items, and recommended next actions.
   - completion: The outcome report artefact is a complete readable Markdown document consistent with the workflow status artefact.
   - join: any
   - execution: inherit

## Transitions
| ID | From Step | To Step | Condition | Value | Max Traversals |
|----|-----------|---------|-----------|-------|----------------|
| transition-intake-to-coordinate | intake-and-plan | coordinate-work | on-success | - | - |
| transition-completed-to-consolidate | coordinate-work | consolidate-outcome | outcome-is | objective_completed | - |
| transition-authority-blocked-to-consolidate | coordinate-work | consolidate-outcome | outcome-is | authority_blocked | - |
| transition-operationally-blocked-to-consolidate | coordinate-work | consolidate-outcome | outcome-is | operationally_blocked | - |

## Artefacts
| ID | Artifact | Path | Format |
|----|----------|------|--------|
| orchestration-plan | Peer2Paper Orchestration Plan JSON Derivative External State Snapshot Durable Cross-Run Operational Record Timeline Audit Trail Decision Log Evidence Appendix Handoff Packet Dependency Map Risk Register Approval Matrix Status Ledger Progress Dashboard Outcome Taxonomy Acceptance Checklist Specialist Assignment Roster Context Bundle Constraint Register Authority Boundary Manifest Escalation Dossier Recovery Playbook Retry Journal Completion Certificate Executive Summary Technical Notes Reconciliation Table Verification Matrix Final Disposition Archive Index Runbook Supplement Governance Record Compliance Traceability Report Stakeholder Brief Incident Addendum Lessons Learned Backlog Recommendations and Continuity Guide for Coordinating Intake Through Clear Outcome Across All Participating Agents and Stages While Preserving Explicit User Constraints Delegated Authority Operational Dependencies Measurable Completion Criteria and Recommended Next Actions in a Single Canonical Machine-Readable Project-Scoped File Produced Fresh for Every Invocation of the End-to-End Workflow without Relying on Undeclared Inputs Runtime Scheduling State or Generic Manual Browser Substitutes and with Complete Provenance for Every Decision Assignment Blocker Escalation Result and Unresolved Question Encountered During Execution So Subsequent Operators Can Reconstruct the Full Run Safely Reliably Deterministically Transparently Accountably Efficiently Consistently and Unambiguously Under Normal Partial-Failure Approval-Pending and Recovery Conditions Across Development Staging Production Audit Review Retrospective Handoff Resumption Replanning Reconciliation Closure Archival Governance Oversight Quality Assurance Security Privacy Legal Financial Reputational Customer Partner Vendor Community Ecosystem Regulatory Contractual Policy Ethical Safety Reliability Availability Integrity Confidentiality Accessibility Usability Maintainability Portability Interoperability Scalability Performance Cost Sustainability Localization Internationalization Data Retention Records Management Knowledge Transfer Training Support Operations Change Management Release Management Configuration Management Service Management Incident Management Problem Management Capacity Management Continuity Planning Disaster Recovery Business Impact Analysis Threat Modeling Abuse Prevention Fraud Detection Claim Validation Bounty Adjudication Dispute Resolution Payment Readiness Communications Coordination and Organizational Learning Contexts with Explicit Typed References to Every Consumed Routine Parameter and Produced Artefact Along the Reachable Success and Blocked Paths of This Schema Two Routine Family Root Definition Record. | project://peer2paper/orchestration/current-run-plan.json | json |
| workflow-status | Peer2Paper Workflow Status JSON Derivative External State Snapshot Durable Cross-Run Operational Record Timeline Audit Trail Decision Log Evidence Appendix Handoff Packet Dependency Map Risk Register Approval Matrix Status Ledger Progress Dashboard Outcome Taxonomy Acceptance Checklist Specialist Assignment Roster Context Bundle Constraint Register Authority Boundary Manifest Escalation Dossier Recovery Playbook Retry Journal Completion Certificate Executive Summary Technical Notes Reconciliation Table Verification Matrix Final Disposition Archive Index Runbook Supplement Governance Record Compliance Traceability Report Stakeholder Brief Incident Addendum Lessons Learned Backlog Recommendations and Continuity Guide for Coordinating Intake Through Clear Outcome Across All Participating Agents and Stages While Preserving Explicit User Constraints Delegated Authority Operational Dependencies Measurable Completion Criteria and Recommended Next Actions in a Single Canonical Machine-Readable Project-Scoped File Produced Fresh for Every Invocation of the End-to-End Workflow without Relying on Undeclared Inputs Runtime Scheduling State or Generic Manual Browser Substitutes and with Complete Provenance for Every Decision Assignment Blocker Escalation Result and Unresolved Question Encountered During Execution So Subsequent Operators Can Reconstruct the Full Run Safely Reliably Deterministically Transparently Accountably Efficiently Consistently and Unambiguously Under Normal Partial-Failure Approval-Pending and Recovery Conditions Across Development Staging Production Audit Review Retrospective Handoff Resumption Replanning Reconciliation Closure Archival Governance Oversight Quality Assurance Security Privacy Legal Financial Reputational Customer Partner Vendor Community Ecosystem Regulatory Contractual Policy Ethical Safety Reliability Availability Integrity Confidentiality Accessibility Usability Maintainability Portability Interoperability Scalability Performance Cost Sustainability Localization Internationalization Data Retention Records Management Knowledge Transfer Training Support Operations Change Management Release Management Configuration Management Service Management Incident Management Problem Management Capacity Management Continuity Planning Disaster Recovery Business Impact Analysis Threat Modeling Abuse Prevention Fraud Detection Claim Validation Bounty Adjudication Dispute Resolution Payment Readiness Communications Coordination and Organizational Learning Contexts with Explicit Typed References to Every Consumed Routine Parameter and Produced Artefact Along the Reachable Success and Blocked Paths of This Schema Two Routine Family Root Definition Record. | project://peer2paper/orchestration/current-run-status.json | json |
| outcome-report | Peer2Paper Outcome Report Markdown Derivative External State Snapshot Durable Cross-Run Operational Record Timeline Audit Trail Decision Log Evidence Appendix Handoff Packet Dependency Map Risk Register Approval Matrix Status Ledger Progress Dashboard Outcome Taxonomy Acceptance Checklist Specialist Assignment Roster Context Bundle Constraint Register Authority Boundary Manifest Escalation Dossier Recovery Playbook Retry Journal Completion Certificate Executive Summary Technical Notes Reconciliation Table Verification Matrix Final Disposition Archive Index Runbook Supplement Governance Record Compliance Traceability Report Stakeholder Brief Incident Addendum Lessons Learned Backlog Recommendations and Continuity Guide for Coordinating Intake Through Clear Outcome Across All Participating Agents and Stages While Preserving Explicit User Constraints Delegated Authority Operational Dependencies Measurable Completion Criteria and Recommended Next Actions in a Single Canonical Human-Readable Project-Scoped File Produced Fresh for Every Invocation of the End-to-End Workflow without Relying on Undeclared Inputs Runtime Scheduling State or Generic Manual Browser Substitutes and with Complete Provenance for Every Decision Assignment Blocker Escalation Result and Unresolved Question Encountered During Execution So Subsequent Operators Can Reconstruct the Full Run Safely Reliably Deterministically Transparently Accountably Efficiently Consistently and Unambiguously Under Normal Partial-Failure Approval-Pending and Recovery Conditions Across Development Staging Production Audit Review Retrospective Handoff Resumption Replanning Reconciliation Closure Archival Governance Oversight Quality Assurance Security Privacy Legal Financial Reputational Customer Partner Vendor Community Ecosystem Regulatory Contractual Policy Ethical Safety Reliability Availability Integrity Confidentiality Accessibility Usability Maintainability Portability Interoperability Scalability Performance Cost Sustainability Localization Internationalization Data Retention Records Management Knowledge Transfer Training Support Operations Change Management Release Management Configuration Management Service Management Incident Management Problem Management Capacity Management Continuity Planning Disaster Recovery Business Impact Analysis Threat Modeling Abuse Prevention Fraud Detection Claim Validation Bounty Adjudication Dispute Resolution Payment Readiness Communications Coordination and Organizational Learning Contexts with Explicit Typed References to Every Consumed Routine Parameter and Produced Artefact Along the Reachable Success and Blocked Paths of This Schema Two Routine Family Root Definition Record. | project://peer2paper/orchestration/current-run-outcome.md | markdown |

## Escalation
Escalate approval-sensitive, destructive, externally consequential, materially ambiguous, or out-of-authority work to the responsible human decision-maker. Include the objective, blocked action, current status, relevant evidence, risk, decision required, and safe next options. Do not proceed until the required authorization is supplied.
