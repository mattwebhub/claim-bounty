#!/usr/bin/env node

import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';

const root = 'workflow/claimbounty-scientific-audit/trajectories';
const schema = JSON.parse(await readFile(path.join(root, 'trajectory.schema.json'), 'utf8'));
const index = JSON.parse(await readFile(path.join(root, 'index.json'), 'utf8'));
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

const expectedRoutines = [
  'run-claimbounty-scientific-audit',
  'build-and-freeze-study-case',
  'reproduce-original-result',
  'stress-test-analysis',
  'research-methods-and-evidence',
  'verify-and-adjudicate-findings',
  'assemble-final-audit',
];
const expectedAgents = [
  'claim-bounty-operations/claimbounty-orchestrator',
  'claim-bounty-intake/claim-mapper',
  'claim-bounty-intake/variable-mapper',
  'claim-bounty-reproduction/reproduction-engineer',
  'claim-bounty-verification/source-evidence-verifier',
  'claim-bounty-robustness/robustness-lead',
  'claim-bounty-robustness/data-choices-analyst',
  'claim-bounty-robustness/model-and-covariate-analyst',
  'claim-bounty-robustness/inference-analyst',
  'claim-bounty-verification/blind-method-reviewer-alpha',
  'claim-bounty-verification/blind-method-reviewer-beta',
  'research-and-insights/researcher',
  'research-and-insights/literature-searcher',
  'research-and-insights/source-extractor',
  'claim-bounty-verification/verification-lead',
  'claim-bounty-verification/statistical-evidence-verifier',
  'claim-bounty-verification/scientific-adjudicator',
  'claim-bounty-delivery/audit-report-builder',
  'claim-bounty-delivery/release-quality-reviewer',
];
const removedPredecessorAgents = [
  'claim-bounty-intake/intake-mapper',
  'claim-bounty-robustness/skeptical-analyst',
];
const allowedGrades = new Set(['exact', 'equivalent_predecessor', 'reconstructed']);
const allowedRelations = new Set(['current', 'predecessor']);
const forbiddenKeys = new Set([
  'claim_text',
  'scientific_values',
  'manuscript_passages',
  'source_identities',
  'research_data',
  'variable_names',
  'participant_information',
  'hidden_answers',
  'local_path',
  'internal_path',
  'raw_hash',
  'original_hash',
  'run_id',
  'invocation_id',
  'provider',
  'account',
  'email',
  'authentication',
  'prompt',
  'message_body',
  'tool_input',
  'tool_output',
  'shell_command',
  'environment',
  'browser_material',
  'screenshot',
  'secret',
  'token',
  'cookie',
  'headers',
  'username',
  'hostname',
  'raw_artifact',
]);
const forbiddenValuePatterns = [
  [
    'absolute local path',
    /(?:^|[\s"'`])(?:\/Users\/|\/home\/|\/var\/|\/tmp\/|\/private\/|\/Applications\/|[A-Za-z]:\\)/,
  ],
  ['internal organization path', new RegExp(`\\.${'to' + 'one'}\\/`, 'i')],
  ['file URL', /file:\/\//i],
  ['email address', /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/],
  [
    'credential-shaped value',
    /(?:Bearer\s+[A-Za-z0-9._~+/-]{12,}|gh[pousr]_[A-Za-z0-9_]{20,}|sk-[A-Za-z0-9_-]{20,})/i,
  ],
  ['private key material', /-----BEGIN .*PRIVATE KEY-----/i],
  ['raw SHA-256 digest', /\b[0-9a-f]{64}\b/i],
  ['raw UUID', /\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b/i],
  ['external URL', /https?:\/\//i],
  [
    'forbidden interaction material',
    /\b(?:prompt|tool output|shell command|browser|screenshot|cookie|authorization|authentication|access token|session token|provider account)\b/i,
  ],
];

const ajv = new Ajv2020({ allErrors: true, strict: true });
addFormats(ajv);
const validate = ajv.compile(schema);

async function readRecords(directory) {
  const names = (await readdir(path.join(root, directory)))
    .filter((name) => name.endsWith('.json'))
    .sort();
  return Promise.all(
    names.map(async (name) => ({
      name,
      record: JSON.parse(await readFile(path.join(root, directory, name), 'utf8')),
    })),
  );
}

function inspectValue(value, location) {
  if (Array.isArray(value)) {
    value.forEach((item, index) => inspectValue(item, `${location}[${index}]`));
    return;
  }
  if (value && typeof value === 'object') {
    for (const [key, item] of Object.entries(value)) {
      check(!forbiddenKeys.has(key), `forbidden field ${key} at ${location}`);
      inspectValue(item, `${location}.${key}`);
    }
    return;
  }
  if (typeof value !== 'string') return;
  for (const [label, pattern] of forbiddenValuePatterns) {
    check(!pattern.test(value), `${label} at ${location}`);
  }
}

const routines = await readRecords('routines');
const agents = await readRecords('agents');
const all = [...routines, ...agents];

for (const { name, record } of all) {
  if (!validate(record)) {
    for (const error of validate.errors ?? []) {
      failures.push(`schema ${record.record_type}/${name}${error.instancePath}: ${error.message}`);
    }
  }
  check(allowedGrades.has(record.evidence_grade), `evidence grade in ${name}`);
  check(allowedRelations.has(record.source_revision_relation), `revision relation in ${name}`);
  check(
    record.proves_current_revision_end_to_end_completion === false,
    `current-revision completion prohibition in ${name}`,
  );
  check(record.contains_research_payload === false, `research payload exclusion in ${name}`);
  check(record.contains_session_metadata === false, `session metadata exclusion in ${name}`);
  inspectValue(record, `${record.record_type}/${record.public_id}`);
}

const routineIds = routines.map(({ record }) => record.public_id);
const agentIds = agents.map(({ record }) => record.public_id);
check(routines.length === 7, 'seven routine records');
check(agents.length === 19, 'nineteen current-agent records');
check(
  JSON.stringify([...routineIds].sort()) === JSON.stringify([...expectedRoutines].sort()),
  'exact public routine membership',
);
check(
  JSON.stringify([...agentIds].sort()) === JSON.stringify([...expectedAgents].sort()),
  'exact current-agent membership',
);
check(
  removedPredecessorAgents.every((id) => !agentIds.includes(id)),
  'removed predecessor agents excluded from current membership',
);
const workflowStepCount = routines.reduce((sum, { record }) => sum + record.step_order.length, 0);
check(workflowStepCount === 47, '47 ordered routine steps');

check(index.routine_count === 7, 'index routine count');
check(index.agent_count === 19, 'index agent count');
check(index.workflow_step_count === 47, 'index workflow step count');
check(index.current_revision_end_to_end_run_claimed === false, 'index completion prohibition');
check(index.records.length === 26, 'index record count');
check(
  JSON.stringify(index.allowed_evidence_grades) ===
    JSON.stringify(['exact', 'equivalent_predecessor', 'reconstructed']),
  'index evidence-grade allowlist',
);
check(
  index.records.every((entry) =>
    all.some(
      ({ record }) =>
        record.record_type === entry.record_type &&
        record.public_id === entry.public_id &&
        record.evidence_grade === entry.evidence_grade &&
        record.source_revision_relation === entry.source_revision_relation,
    ),
  ),
  'index-to-record identity',
);

if (failures.length > 0) {
  for (const failure of failures) console.error(`public-trajectories: invalid ${failure}`);
  process.exit(1);
}

console.log(
  'public-trajectories: 7 routines, 19 current agents, 47 steps, schema and security checks passed',
);
