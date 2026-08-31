#!/usr/bin/env node

import { createHash } from 'node:crypto';
import { mkdir, readFile, readdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const root = 'workflow/claimbounty-scientific-audit/trajectories';
const checkOnly = process.argv.includes('--check');
const redactions = [
  'research_content_removed',
  'identity_details_removed',
  'runtime_metadata_removed',
  'interaction_content_removed',
  'local_identifiers_replaced',
  'integrity_values_replaced',
  'uncleared_artifacts_removed',
];

const routineDefinitions = [
  {
    id: 'run-claimbounty-scientific-audit',
    grade: 'equivalent_predecessor',
    relation: 'predecessor',
    summary:
      'Representative predecessor orchestration recovered a bounded stage failure, enforced verification ordering, joined corrections, and reached limited delivery.',
    status: 'completed_with_limits',
    outcome: 'audit_completed_with_limits',
    steps: [
      'invoke-build-study-case',
      'invoke-reproduce-result',
      'decide-reproduction-continuation',
      'invoke-stress-test',
      'invoke-research-evidence',
      'invoke-verify-findings',
      'invoke-deliver-audit',
    ],
    corrections: 2,
    received: [],
    delivered: [
      'build-and-freeze-study-case',
      'reproduce-original-result',
      'stress-test-analysis',
      'research-methods-and-evidence',
      'verify-and-adjudicate-findings',
      'assemble-final-audit',
    ],
    inputs: ['audit_request', 'execution_policy', 'scientific_policy', 'case_bundle'],
    errors: [
      ['stage_failure_recovered', 'A bounded stage failure was recovered before continuation.'],
      [
        'delivery_sequence_rejected',
        'Delivery was held until verification and adjudication joined.',
      ],
    ],
    outputs: ['audit_package'],
    limitations: ['predecessor_revision_only', 'current_revision_partial'],
  },
  {
    id: 'build-and-freeze-study-case',
    grade: 'exact',
    relation: 'current',
    summary:
      'The current revision mapped the bounded target, preserved a document conflict, and froze the case with visible limits.',
    status: 'completed_with_limits',
    outcome: 'case_frozen_with_limits',
    started: '2026-08-30T19:33:32Z',
    finished: '2026-08-30T19:55:51Z',
    steps: [
      'validate-claim-map-documents',
      'map-target-data',
      'map-target-code',
      'verify-document-consistency',
      'bind-and-freeze-analysis',
    ],
    received: ['run-claimbounty-scientific-audit'],
    delivered: ['reproduce-original-result'],
    inputs: ['audit_request', 'scientific_policy', 'case_bundle'],
    outputs: ['frozen_study_case'],
    limitations: ['nonblocking_document_conflict'],
    exactSteps: true,
  },
  {
    id: 'reproduce-original-result',
    grade: 'exact',
    relation: 'current',
    summary:
      'The current revision mapped execution, applied one bounded repair, ran one secondary target, and packaged a partial reproduction.',
    status: 'partial',
    outcome: 'partial_reproduction',
    started: '2026-08-30T19:55:51Z',
    finished: '2026-08-30T20:41:35Z',
    steps: [
      'freeze-reproduction-inputs',
      'preflight-execution',
      'run-primary-unchanged',
      'compare-primary-result',
      'diagnose-freeze-repairs',
      'test-repair-1',
      'test-repair-2',
      'test-repair-3',
      'run-secondary-target-1',
      'run-secondary-target-2',
      'run-secondary-target-3',
      'package-reproduction',
    ],
    received: ['build-and-freeze-study-case'],
    delivered: ['run-claimbounty-scientific-audit'],
    inputs: ['frozen_study_case', 'execution_policy', 'scientific_policy', 'case_bundle'],
    errors: [['nonclean_entrypoint', 'The untouched entry point required one bounded repair.']],
    outputs: ['reproduction_package'],
    limitations: ['partial_reproduction', 'current_revision_not_end_to_end'],
  },
  {
    id: 'stress-test-analysis',
    grade: 'equivalent_predecessor',
    relation: 'predecessor',
    summary:
      'Representative predecessor evidence covers result-blind design lanes, candidate freeze, two blind reviews, admissibility, isolated execution, and result mapping.',
    status: 'completed_with_limits',
    outcome: 'robustness_ready_with_limits',
    started: '2026-08-29T15:26:51Z',
    finished: '2026-08-29T16:47:45Z',
    steps: [
      'freeze-sensitivity-contract',
      'design-data-measurement-lane',
      'design-model-lane',
      'design-inference-lane',
      'rank-sensitivity-candidates',
      'review-candidates-alpha',
      'review-candidates-beta',
      'resolve-candidate-admissibility',
      'execute-approved-candidates',
      'build-robustness-map',
    ],
    corrections: 1,
    received: ['reproduce-original-result'],
    delivered: ['verify-and-adjudicate-findings'],
    inputs: ['frozen_study_case', 'reproduction_package', 'scientific_policy'],
    outputs: ['robustness_map'],
    limitations: ['predecessor_revision_only', 'visible_nonblocking_gaps'],
  },
  {
    id: 'research-methods-and-evidence',
    grade: 'equivalent_predecessor',
    relation: 'predecessor',
    summary:
      'Representative predecessor evidence resumed a bounded extraction after timeout, repaired stale provenance, and assembled a limited evidence package.',
    status: 'completed_with_limits',
    outcome: 'evidence_ready_with_limits',
    finished: '2026-08-29T17:27:05.264Z',
    steps: [
      'freeze-research-brief',
      'search-screen-slice-sources',
      'map-full-text-source-1',
      'map-full-text-source-2',
      'map-full-text-source-3',
      'assemble-targeted-evidence',
    ],
    retries: 1,
    corrections: 1,
    received: ['reproduce-original-result'],
    delivered: ['verify-and-adjudicate-findings'],
    inputs: ['frozen_study_case', 'reproduction_package', 'research_policy'],
    errors: [
      [
        'bounded_extraction_timeout',
        'A bounded extraction timed out and resumed from the frozen map.',
      ],
      ['stale_provenance_repaired', 'A stale publication chain was corrected through one writer.'],
    ],
    outputs: ['literature_evidence_package'],
    limitations: ['predecessor_revision_only', 'visible_source_limits'],
  },
  {
    id: 'verify-and-adjudicate-findings',
    grade: 'equivalent_predecessor',
    relation: 'predecessor',
    summary:
      'Representative predecessor evidence joined independent statistical and source checks, preserved disagreements, and denied delivery until adjudication completed.',
    status: 'completed_with_limits',
    outcome: 'verification_incomplete',
    steps: [
      'freeze-verification-inputs',
      'verify-statistical-evidence',
      'verify-source-evidence',
      'adjudicate-verified-findings',
    ],
    corrections: 2,
    received: ['stress-test-analysis', 'research-methods-and-evidence'],
    delivered: ['assemble-final-audit'],
    inputs: ['reproduction_package', 'robustness_map', 'literature_evidence_package'],
    errors: [
      [
        'control_profile_rejected',
        'An invalid verification preflight was rejected before analysis.',
      ],
      ['premature_delivery_denied', 'Delivery was denied until both independent checks joined.'],
    ],
    outputs: ['adjudication_package', 'manuscript_recommendations'],
    limitations: ['predecessor_revision_only', 'unresolved_disagreements'],
  },
  {
    id: 'assemble-final-audit',
    grade: 'equivalent_predecessor',
    relation: 'predecessor',
    summary:
      'Representative predecessor evidence built canonical and rendered outputs, corrected renderer selection, confirmed matching renders, and completed a clean replay.',
    status: 'completed_with_limits',
    outcome: 'audit_completed_with_limits',
    steps: ['build-canonical-audit', 'render-researcher-html', 'check-and-package-audit'],
    corrections: 1,
    received: ['verify-and-adjudicate-findings'],
    delivered: ['run-claimbounty-scientific-audit'],
    inputs: ['adjudication_package', 'manuscript_recommendations', 'audit_schema'],
    errors: [['renderer_selection_corrected', 'A renderer selection mismatch was corrected.']],
    outputs: ['audit_json', 'audit_html', 'replay_package', 'audit_package'],
    schemaStatus: 'reported_valid',
    integrity: {
      algorithm: 'sha256',
      evidence_id: 'trajectory-evidence://routine/assemble-render-match',
      hash_match: true,
    },
    limitations: ['predecessor_revision_only', 'visible_release_limits'],
  },
];

const agentDefinitions = [
  [
    'claim-bounty-operations/claimbounty-orchestrator',
    'equivalent_predecessor',
    'predecessor',
    'Gated bounded stages, corrected ordering, joined verification, and reached limited predecessor delivery.',
    [
      'invoke-build-study-case',
      'decide-reproduction-continuation',
      'invoke-verify-findings',
      'invoke-deliver-audit',
    ],
    2,
    null,
    [],
    [
      'claim-bounty-intake/claim-mapper',
      'claim-bounty-reproduction/reproduction-engineer',
      'claim-bounty-verification/verification-lead',
      'claim-bounty-delivery/audit-report-builder',
    ],
    ['audit_package'],
    ['predecessor_revision_only', 'current_revision_partial'],
  ],
  [
    'claim-bounty-intake/claim-mapper',
    'exact',
    'current',
    'Validated bounded documents, preserved a conflict, and bound the frozen study case.',
    ['validate-claim-map-documents', 'bind-and-freeze-analysis'],
    null,
    null,
    [
      'claim-bounty-operations/claimbounty-orchestrator',
      'claim-bounty-intake/variable-mapper',
      'claim-bounty-verification/source-evidence-verifier',
    ],
    ['claim-bounty-reproduction/reproduction-engineer'],
    ['frozen_study_case'],
    ['nonblocking_document_conflict'],
  ],
  [
    'claim-bounty-intake/variable-mapper',
    'exact',
    'current',
    'Mapped the target-scoped dependency closure and handed it to final case binding.',
    ['map-target-data'],
    null,
    null,
    ['claim-bounty-intake/claim-mapper'],
    ['claim-bounty-intake/claim-mapper'],
    ['target_scoped_map'],
    [],
  ],
  [
    'claim-bounty-reproduction/reproduction-engineer',
    'exact',
    'current',
    'Mapped execution, diagnosed a nonclean entry point, applied bounded repair, ran one secondary target, and packaged partial reproduction.',
    [
      'map-target-code',
      'preflight-execution',
      'run-primary-unchanged',
      'diagnose-freeze-repairs',
      'test-repair-1',
      'run-secondary-target-1',
      'package-reproduction',
    ],
    1,
    null,
    ['claim-bounty-intake/claim-mapper'],
    ['claim-bounty-operations/claimbounty-orchestrator'],
    ['reproduction_package'],
    ['partial_reproduction'],
  ],
  [
    'claim-bounty-verification/source-evidence-verifier',
    'exact',
    'current',
    'Produced current supplied-document consistency findings under the bounded intake scope.',
    ['verify-document-consistency'],
    null,
    null,
    ['claim-bounty-reproduction/reproduction-engineer'],
    ['claim-bounty-intake/claim-mapper'],
    ['document_consistency_findings'],
    ['later_verification_represented_by_predecessor'],
  ],
  [
    'claim-bounty-robustness/robustness-lead',
    'equivalent_predecessor',
    'predecessor',
    'Froze estimand rules, joined result-blind lanes, resolved admissibility, built the robustness map, and accepted one bounded correction.',
    [
      'freeze-sensitivity-contract',
      'rank-sensitivity-candidates',
      'resolve-candidate-admissibility',
      'build-robustness-map',
    ],
    null,
    1,
    [
      'claim-bounty-robustness/data-choices-analyst',
      'claim-bounty-robustness/model-and-covariate-analyst',
      'claim-bounty-robustness/inference-analyst',
      'claim-bounty-verification/blind-method-reviewer-alpha',
      'claim-bounty-verification/blind-method-reviewer-beta',
    ],
    ['claim-bounty-verification/verification-lead'],
    ['robustness_map'],
    ['predecessor_revision_only'],
  ],
  [
    'claim-bounty-robustness/data-choices-analyst',
    'reconstructed',
    'predecessor',
    'Produced result-blind measurement and sample decision records for ranking.',
    ['design-data-measurement-lane'],
    null,
    null,
    ['claim-bounty-robustness/robustness-lead'],
    ['claim-bounty-robustness/robustness-lead'],
    ['data_measurement_candidates'],
    ['reconstructed_from_sanitized_inventory'],
  ],
  [
    'claim-bounty-robustness/model-and-covariate-analyst',
    'reconstructed',
    'predecessor',
    'Registered result-blind model and covariate options for candidate ranking.',
    ['design-model-lane'],
    null,
    null,
    ['claim-bounty-robustness/robustness-lead'],
    ['claim-bounty-robustness/robustness-lead'],
    ['model_specification_candidates'],
    ['reconstructed_from_sanitized_inventory'],
  ],
  [
    'claim-bounty-robustness/inference-analyst',
    'reconstructed',
    'predecessor',
    'Produced result-blind uncertainty and multiplicity decision lanes.',
    ['design-inference-lane'],
    null,
    null,
    ['claim-bounty-robustness/robustness-lead'],
    ['claim-bounty-robustness/robustness-lead'],
    ['inference_candidates'],
    ['reconstructed_from_sanitized_inventory'],
  ],
  [
    'claim-bounty-verification/blind-method-reviewer-alpha',
    'exact',
    'predecessor',
    'Independently identified duplicate candidate families and execution-gate issues before admissibility resolution.',
    ['review-candidates-alpha'],
    null,
    null,
    ['claim-bounty-robustness/robustness-lead'],
    ['claim-bounty-robustness/robustness-lead'],
    ['blind_review_alpha'],
    ['predecessor_revision_only'],
  ],
  [
    'claim-bounty-verification/blind-method-reviewer-beta',
    'exact',
    'predecessor',
    'Performed a second independent result-blind review without inspecting the first review or outcomes.',
    ['review-candidates-beta'],
    null,
    null,
    ['claim-bounty-robustness/robustness-lead'],
    ['claim-bounty-robustness/robustness-lead'],
    ['blind_review_beta'],
    ['predecessor_revision_only'],
  ],
  [
    'research-and-insights/researcher',
    'exact',
    'predecessor',
    'Froze the brief, reviewed extraction, corrected stale provenance through one writer, and assembled the evidence package.',
    ['freeze-research-brief', 'assemble-targeted-evidence'],
    null,
    1,
    ['research-and-insights/literature-searcher', 'research-and-insights/source-extractor'],
    ['claim-bounty-verification/verification-lead'],
    ['literature_evidence_package'],
    ['predecessor_revision_only'],
  ],
  [
    'research-and-insights/literature-searcher',
    'reconstructed',
    'predecessor',
    'Ran three bounded search lanes and handed a screened registry to extraction.',
    ['search-screen-slice-sources'],
    null,
    null,
    ['research-and-insights/researcher'],
    ['research-and-insights/source-extractor'],
    ['screened_registry'],
    ['reconstructed_from_sanitized_inventory'],
  ],
  [
    'research-and-insights/source-extractor',
    'exact',
    'predecessor',
    'Resumed after one bounded extraction timeout, preserved concurrent updates, corrected provenance, and completed extraction.',
    ['map-full-text-source-1', 'map-full-text-source-2', 'map-full-text-source-3'],
    1,
    1,
    ['research-and-insights/literature-searcher'],
    ['research-and-insights/researcher'],
    ['source_extractions'],
    ['predecessor_revision_only'],
  ],
  [
    'claim-bounty-verification/verification-lead',
    'exact',
    'predecessor',
    'Froze the packet, reconciled independent checks, preserved material disagreements, and enforced the verification join.',
    ['freeze-verification-inputs'],
    null,
    null,
    ['claim-bounty-robustness/robustness-lead', 'research-and-insights/researcher'],
    [
      'claim-bounty-verification/statistical-evidence-verifier',
      'claim-bounty-verification/source-evidence-verifier',
      'claim-bounty-verification/scientific-adjudicator',
    ],
    ['verification_inputs'],
    ['predecessor_revision_only', 'unresolved_disagreements'],
  ],
  [
    'claim-bounty-verification/statistical-evidence-verifier',
    'equivalent_predecessor',
    'predecessor',
    'Rejected an invalid preflight and completed an independent replay on the corrected attempt.',
    ['verify-statistical-evidence'],
    1,
    null,
    ['claim-bounty-verification/verification-lead'],
    ['claim-bounty-verification/scientific-adjudicator'],
    ['statistical_verification'],
    ['predecessor_revision_only'],
  ],
  [
    'claim-bounty-verification/scientific-adjudicator',
    'exact',
    'predecessor',
    'Denied premature delivery, required two bounded corrections, and authorized limited delivery with unresolved disagreements.',
    ['adjudicate-verified-findings'],
    null,
    2,
    [
      'claim-bounty-verification/statistical-evidence-verifier',
      'claim-bounty-verification/source-evidence-verifier',
    ],
    ['claim-bounty-delivery/audit-report-builder'],
    ['adjudication_package', 'manuscript_recommendations'],
    ['predecessor_revision_only', 'verification_incomplete'],
  ],
  [
    'claim-bounty-delivery/audit-report-builder',
    'exact',
    'predecessor',
    'Built canonical and rendered outputs, corrected renderer selection, checked deterministic rendering, and ran clean replay.',
    ['build-canonical-audit', 'render-researcher-html'],
    null,
    1,
    ['claim-bounty-verification/scientific-adjudicator'],
    ['claim-bounty-delivery/release-quality-reviewer'],
    ['audit_json', 'audit_html', 'replay_package'],
    ['predecessor_revision_only'],
  ],
  [
    'claim-bounty-delivery/release-quality-reviewer',
    'exact',
    'predecessor',
    'Checked completeness, format consistency, replay, accessibility, privacy, licensing, and release scope.',
    ['check-and-package-audit'],
    null,
    null,
    ['claim-bounty-delivery/audit-report-builder'],
    ['claim-bounty-operations/claimbounty-orchestrator'],
    ['release_gates', 'audit_package'],
    ['predecessor_revision_only', 'visible_release_limits'],
  ],
];

function baseRecord(type, definition) {
  const integrity = definition.integrity ?? {
    algorithm: null,
    evidence_id: null,
    hash_match: null,
  };
  return {
    schema_version: '1.0.0',
    record_type: type,
    public_id: definition.id,
    record_uri: `trajectory://${type}/${definition.id}`,
    evidence_grade: definition.grade,
    source_revision_relation: definition.relation,
    summary: definition.summary,
    status: definition.status ?? 'represented',
    outcome: definition.outcome ?? null,
    proves_current_revision_end_to_end_completion: false,
    timestamps: {
      started_at_utc: definition.started ?? null,
      finished_at_utc: definition.finished ?? null,
    },
    step_order: definition.steps.map((public_step_id, index) => ({
      position: index + 1,
      public_step_id,
      status: definition.exactSteps ? 'completed' : 'represented',
    })),
    retry_count: definition.retries ?? null,
    correction_count: definition.corrections ?? null,
    handoffs: {
      received_from: definition.received,
      delivered_to: definition.delivered,
    },
    input_classes: definition.inputs ?? ['sanitized_workflow_state'],
    bounded_errors: (definition.errors ?? []).map(([code, summary]) => ({ code, summary })),
    output_artifact_kinds: definition.outputs,
    schema_status: definition.schemaStatus ?? 'not_reported',
    integrity,
    release_limitations: definition.limitations,
    redactions_applied: redactions,
    contains_research_payload: false,
    contains_session_metadata: false,
  };
}

const routineRecords = routineDefinitions.map((definition) => baseRecord('routine', definition));
const agentRecords = agentDefinitions.map(
  ([
    id,
    grade,
    relation,
    summary,
    steps,
    retries,
    corrections,
    received,
    delivered,
    outputs,
    limitations,
  ]) =>
    baseRecord('agent', {
      id,
      grade,
      relation,
      summary,
      steps,
      retries,
      corrections,
      received,
      delivered,
      outputs,
      limitations,
    }),
);

const workflowStepCount = routineRecords.reduce((sum, record) => sum + record.step_order.length, 0);
if (routineRecords.length !== 7 || agentRecords.length !== 19 || workflowStepCount !== 47) {
  throw new Error('trajectory allowlist must contain 7 routines, 19 agents, and 47 workflow steps');
}

const generated = new Map();
for (const record of routineRecords) {
  generated.set(`routines/${record.public_id}.json`, `${JSON.stringify(record, null, 2)}\n`);
}
for (const record of agentRecords) {
  const filename = record.public_id.replace('/', '__');
  generated.set(`agents/${filename}.json`, `${JSON.stringify(record, null, 2)}\n`);
}

const index = {
  schema_version: '1.0.0',
  package_status: 'representative_sanitized_projection',
  routine_count: routineRecords.length,
  agent_count: agentRecords.length,
  workflow_step_count: workflowStepCount,
  current_revision_end_to_end_run_claimed: false,
  allowed_evidence_grades: ['exact', 'equivalent_predecessor', 'reconstructed'],
  allowed_source_revision_relations: ['current', 'predecessor'],
  records: [
    ...routineRecords.map((record) => ({
      record_type: record.record_type,
      public_id: record.public_id,
      evidence_grade: record.evidence_grade,
      source_revision_relation: record.source_revision_relation,
      path: `routines/${record.public_id}.json`,
      record_uri: record.record_uri,
    })),
    ...agentRecords.map((record) => ({
      record_type: record.record_type,
      public_id: record.public_id,
      evidence_grade: record.evidence_grade,
      source_revision_relation: record.source_revision_relation,
      path: `agents/${record.public_id.replace('/', '__')}.json`,
      record_uri: record.record_uri,
    })),
  ],
  release_limitations: [
    'no_current_revision_end_to_end_completion_claim',
    'predecessor_records_are_representative_only',
    'research_payload_excluded',
  ],
};
generated.set('index.json', `${JSON.stringify(index, null, 2)}\n`);

function digest(contents) {
  return createHash('sha256').update(contents).digest('hex');
}

async function expectedManifest() {
  const payload = new Map(generated);
  for (const manual of ['README.md', 'sanitization-policy.md', 'trajectory.schema.json']) {
    payload.set(manual, await readFile(path.join(root, manual)));
  }
  return `${[...payload.entries()]
    .map(([filename, contents]) => `${digest(contents)}  ${filename}`)
    .sort()
    .join('\n')}\n`;
}

await mkdir(path.join(root, 'routines'), { recursive: true });
await mkdir(path.join(root, 'agents'), { recursive: true });

let failed = false;
for (const [filename, contents] of generated) {
  const destination = path.join(root, filename);
  if (checkOnly) {
    const current = await readFile(destination, 'utf8').catch(() => '');
    if (current !== contents) {
      console.error(`public-trajectories: generated drift in ${destination}`);
      failed = true;
    }
  } else {
    await writeFile(destination, contents, 'utf8');
  }
}

for (const directory of ['routines', 'agents']) {
  const expected = [...generated.keys()]
    .filter((filename) => filename.startsWith(`${directory}/`))
    .map((filename) => path.basename(filename))
    .sort();
  const actual = (await readdir(path.join(root, directory)))
    .filter((name) => name.endsWith('.json'))
    .sort();
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    console.error(`public-trajectories: unexpected JSON record in ${path.join(root, directory)}`);
    failed = true;
  }
}

const manifest = await expectedManifest();
const manifestPath = path.join(root, 'MANIFEST.sha256');
if (checkOnly) {
  const current = await readFile(manifestPath, 'utf8').catch(() => '');
  if (current !== manifest) {
    console.error(`public-trajectories: generated drift in ${manifestPath}`);
    failed = true;
  }
} else {
  await writeFile(manifestPath, manifest, 'utf8');
}

if (failed) process.exit(1);
console.log(
  checkOnly
    ? 'public-trajectories: allowlisted records and manifest are current'
    : 'public-trajectories: generated 7 routine and 19 current-agent records',
);
