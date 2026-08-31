#!/usr/bin/env node

import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';

const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

function checkExactFields(value, expected, label) {
  check(value !== null && typeof value === 'object' && !Array.isArray(value), `${label} object`);
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return;
  check(
    JSON.stringify(Object.keys(value).sort()) === JSON.stringify(Object.keys(expected).sort()),
    `${label} complete field set`,
  );
  for (const [field, expectedValue] of Object.entries(expected)) {
    check(value[field] === expectedValue, `${label} ${field}`);
  }
}

function parseJsonRejectingDuplicateKeys(source, file) {
  let offset = 0;

  function fail(message) {
    throw new Error(`${file}:${offset + 1}: ${message}`);
  }

  function skipWhitespace() {
    while (/\s/.test(source[offset] ?? '')) offset += 1;
  }

  function parseString() {
    if (source[offset] !== '"') fail('expected JSON string');
    const start = offset;
    offset += 1;
    while (offset < source.length) {
      if (source[offset] === '\\') {
        offset += 2;
        continue;
      }
      if (source[offset] === '"') {
        offset += 1;
        return JSON.parse(source.slice(start, offset));
      }
      offset += 1;
    }
    fail('unterminated JSON string');
  }

  function parseValue() {
    skipWhitespace();
    if (source[offset] === '{') {
      offset += 1;
      skipWhitespace();
      const keys = new Set();
      if (source[offset] === '}') {
        offset += 1;
        return;
      }
      while (offset < source.length) {
        skipWhitespace();
        const key = parseString();
        if (keys.has(key)) fail(`duplicate JSON key ${JSON.stringify(key)}`);
        keys.add(key);
        skipWhitespace();
        if (source[offset] !== ':') fail('expected colon after JSON key');
        offset += 1;
        parseValue();
        skipWhitespace();
        if (source[offset] === '}') {
          offset += 1;
          return;
        }
        if (source[offset] !== ',') fail('expected comma between JSON properties');
        offset += 1;
      }
      fail('unterminated JSON object');
    }
    if (source[offset] === '[') {
      offset += 1;
      skipWhitespace();
      if (source[offset] === ']') {
        offset += 1;
        return;
      }
      while (offset < source.length) {
        parseValue();
        skipWhitespace();
        if (source[offset] === ']') {
          offset += 1;
          return;
        }
        if (source[offset] !== ',') fail('expected comma between JSON values');
        offset += 1;
      }
      fail('unterminated JSON array');
    }
    if (source[offset] === '"') {
      parseString();
      return;
    }
    const primitive = source
      .slice(offset)
      .match(/^(?:-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null)/);
    if (!primitive) fail('invalid JSON value');
    offset += primitive[0].length;
  }

  parseValue();
  skipWhitespace();
  if (offset !== source.length) fail('trailing content after JSON value');
  return JSON.parse(source);
}

async function filesBelow(root, extension) {
  const entries = await readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const target = path.join(root, entry.name);
    if (entry.isDirectory()) files.push(...(await filesBelow(target, extension)));
    else if (entry.isFile() && target.endsWith(extension)) files.push(target);
  }
  return files;
}

const publicJsonFiles = (
  await Promise.all(
    ['submission/evidence', 'submission/reviewer', 'workflow/claimbounty-scientific-audit'].map(
      (root) => filesBelow(root, '.json'),
    ),
  )
).flat();
const parsedPublicJson = new Map();
for (const file of publicJsonFiles) {
  const source = await readFile(file, 'utf8');
  try {
    parsedPublicJson.set(file, parseJsonRejectingDuplicateKeys(source, file));
  } catch (error) {
    failures.push(`duplicate-key-safe JSON parse: ${error.message}`);
  }
}

let duplicateKeySelfTestRejected = false;
try {
  parseJsonRejectingDuplicateKeys('{"duplicate":false,"duplicate":true}', 'self-test');
} catch {
  duplicateKeySelfTestRejected = true;
}
check(duplicateKeySelfTestRejected, 'duplicate-key parser self-test');

const comparator = parsedPublicJson.get('submission/evidence/chatgpt-comparator.json');
const benchmark = parsedPublicJson.get('submission/evidence/benchmark-status.json');
const claimBountyAttempt = parsedPublicJson.get(
  'submission/evidence/claimbounty-exploratory-attempt.json',
);

for (const [record, value] of [
  ['comparator', comparator],
  ['benchmark status', benchmark],
  ['ClaimBounty attempt', claimBountyAttempt],
]) {
  check(value !== undefined, `${record} has duplicate-free valid JSON`);
}
if (!comparator || !benchmark || !claimBountyAttempt) {
  for (const failure of failures) console.error(`comparator-disclosure: invalid ${failure}`);
  process.exit(1);
}

const publicDisclosureFiles = [
  'README.md',
  'IMPROVEMENT_CHANGELOG.md',
  ...(await filesBelow('docs', '.md')),
  ...(await filesBelow('submission', '.md')),
  ...(await filesBelow('submission', '.html')),
  ...(await filesBelow('workflow/claimbounty-scientific-audit', '.md')),
  ...publicJsonFiles,
];
for (const file of new Set(publicDisclosureFiles)) {
  const source = await readFile(file, 'utf8');
  check(!/\b0\.\d{3}\b/.test(source), `no restricted case-specific numeric literal in ${file}`);
}

check(comparator.status === 'sealed', 'sealed comparator status');
check(comparator.comparator.product === 'ChatGPT', 'comparator product');
check(comparator.comparator.mode === 'standard_chat', 'standard Chat mode');
check(
  comparator.comparator.chatContext === 'non_personalized_temporary_chat',
  'temporary non-personalized context',
);
check(comparator.comparator.visibleSetting === 'High', 'visible High setting');
check(comparator.comparator.exposedModelName === null, 'unexposed model name');
check(comparator.comparator.modelInferenceAllowed === false, 'model inference prohibition');
check(comparator.frozenInputs.fileCount === 5, 'five frozen files');
check(comparator.frozenInputs.manifestOrderPreserved === true, 'manifest file order');
check(comparator.frozenInputs.allHashesVerified === true, 'frozen file hashes');
check(
  comparator.prompt.sha256 === 'da014c3be55d7a37a35a57816d72adf5ce112a54d4af6c306dd2134f762f4ae2',
  'frozen prompt hash',
);
check(comparator.prompt.submittedCount === 1, 'one submitted prompt');
check(comparator.preSendStaging.filePickerSelectorFailures === 1, 'one pre-send selector failure');
check(comparator.preSendStaging.hiddenFileInputUsed === true, 'hidden file input staging');
check(
  comparator.preSendStaging.sessionHadPriorSubmittedPrompt === false,
  'unused pre-send session',
);

for (const field of [
  'followUps',
  'continueActions',
  'regenerations',
  'edits',
  'branches',
  'retries',
]) {
  check(comparator.interactionCounts[field] === 0, `zero ${field}`);
}

check(comparator.timing.confirmedSendAt === '2026-08-31T03:52:52.532Z', 'confirmed send timestamp');
check(comparator.timing.terminalDetectedAt === '2026-08-31T04:05:58.072Z', 'terminal timestamp');
check(comparator.timing.elapsedSeconds === 785.539676, 'elapsed seconds');
check(
  comparator.terminalClassification === 'truncated_or_malformed_output',
  'terminal classification',
);
check(
  JSON.stringify(comparator.completePreservedOutput) ===
    JSON.stringify([
      'Reviewed PDF pages, audited feeder data, and reproduced GLMM results',
      'Computed robust binomial models, tests, and phase-by-reward effects',
    ]),
  'complete preserved output',
);
check(comparator.requiredReviewerReport.status === 'absent', 'absent reviewer report');
check(comparator.requiredReviewerReport.requestedSections === 8, 'eight requested sections');
check(comparator.requiredReviewerReport.presentSections === 0, 'zero present sections');
check(comparator.requiredReviewerReport.missingSections === 8, 'eight missing sections');
check(comparator.cost.amount === null, 'null comparator cost');
check(comparator.cost.status === 'unmeasured', 'unmeasured comparator cost');
check(comparator.independentScoring.alphaScore === 1, 'Alpha score');
check(comparator.independentScoring.betaScore === 2, 'Beta score');
check(comparator.adjudication.score === 1, 'adjudicated score');
check(comparator.adjudication.capApplied === false, 'no adjudication cap');
check(comparator.adjudication.decision === 'invalid_incomplete', 'adjudication decision');
check(comparator.qualityScore.value === 1, 'published comparator score');
check(comparator.speedup === null, 'absent speedup');
checkExactFields(
  comparator.publicProjection,
  {
    restrictedScreenshotsIncluded: false,
    rawBrowserTracesIncluded: false,
    authenticationOrSessionMetadataIncluded: false,
    absoluteInternalPathsIncluded: false,
  },
  'comparator public projection',
);

check(benchmark.candidate.articleDoi === '10.1016/j.cub.2023.06.009', 'article DOI');
check(
  benchmark.candidate.dataCodeRecordDoi === '10.5281/zenodo.7985236',
  'data and code record DOI',
);
check(benchmark.freeze.status === 'complete_validated', 'validated case freeze');
check(benchmark.chatGptComparator.status === 'completed', 'completed comparator');
check(
  benchmark.claimBountyExecution.status === 'protocol_deviation_unscored',
  'unscored exploratory ClaimBounty attempt',
);
check(
  benchmark.claimBountyExecution.qualificationStatus === 'blocked_two_independent_deviations',
  'ClaimBounty qualification blocked by two deviations',
);
check(benchmark.claimBountyExecution.independentDeviationCount === 2, 'two deviations');
check(
  benchmark.independentScoring.status === 'comparator_complete_claimbounty_not_scored',
  'ClaimBounty not independently scored',
);
check(
  benchmark.adjudication.status === 'comparator_complete_claimbounty_not_adjudicated',
  'ClaimBounty not adjudicated',
);
check(benchmark.qualityResult.value === null, 'absent benchmark quality result');
check(
  benchmark.evaluationSample.challengeRecommendedMinimumWhenFeasible === 10,
  'recommended evaluation sample',
);
check(benchmark.evaluationSample.frozenCaseCount === 1, 'single frozen evaluation case');
check(
  benchmark.evaluationSample.defensibleTenCaseEvaluationCompleted === false,
  'incomplete defensible ten-case evaluation',
);
check(
  benchmark.evaluationSample.generalPerformanceClaimAllowed === false,
  'general performance prohibition',
);
check(
  benchmark.evaluationSample.humanQualitySuperiorityClaimAllowed === false,
  'human-quality superiority prohibition',
);
check(benchmark.evaluationSample.broadSpeedupClaimAllowed === false, 'broad speedup prohibition');

check(claimBountyAttempt.status === 'protocol_deviation_unscored', 'attempt unscored status');
check(claimBountyAttempt.classification === 'exploratory_same_case_attempt', 'exploratory class');
check(
  claimBountyAttempt.protocolConformance.qualificationStatus === 'blocked',
  'blocked qualification',
);
check(
  claimBountyAttempt.protocolConformance.independentDeviationCount === 2,
  'two independent attempt deviations',
);
const deviations = claimBountyAttempt.protocolConformance.independentDeviations;
check(Array.isArray(deviations) && deviations.length === 2, 'two deviation records');
const promptDeviation = deviations?.find((item) => item.code === 'prompt_final_lf_mismatch');
check(promptDeviation?.classification === 'exact_input_identity_failure', 'input identity failure');
check(promptDeviation?.dispatchedTaskContractBytes === 1275, '1,275-byte dispatch');
check(promptDeviation?.frozenPromptFileBytes === 1276, '1,276-byte frozen prompt');
const contamination = deviations?.find(
  (item) => item.code === 'case_specific_values_not_in_participant_inputs',
);
checkExactFields(
  contamination,
  {
    code: 'case_specific_values_not_in_participant_inputs',
    classification: 'out_of_contract_contamination',
    projectedValueCount: 2,
    caseSpecificProjectedValuesIncluded: false,
    contaminationDisclosed: true,
    presentInFrozenPrompt: false,
    presentInParticipantFiles: false,
    summary:
      'The comparison routine injected two case-specific values from outside the frozen participant bundle. They were absent from the frozen prompt and every participant file and are omitted from this public record.',
  },
  'out-of-contract contamination disclosure',
);
check(!('projectedValuesPublished' in contamination), 'no ambiguous projected-values field');
check(!('projectedValues' in contamination), 'no restricted projected values in public record');
const claimBountyAttemptSource = await readFile(
  'submission/evidence/claimbounty-exploratory-attempt.json',
  'utf8',
);
for (const field of [
  'projectedValueCount',
  'caseSpecificProjectedValuesIncluded',
  'contaminationDisclosed',
]) {
  check(
    (claimBountyAttemptSource.match(new RegExp(`"${field}"`, 'g')) ?? []).length === 1,
    `single unambiguous ${field} field`,
  );
}
check(
  claimBountyAttempt.interpretation.scoringAllowed === false,
  'ClaimBounty scoring prohibition',
);
check(
  claimBountyAttempt.interpretation.qualifiedHeadToHeadComparisonAllowed === false,
  'head-to-head prohibition',
);
check(claimBountyAttempt.interpretation.speedupClaimAllowed === false, 'speedup prohibition');
checkExactFields(
  claimBountyAttempt.publicProjection,
  {
    runIdentifierIncluded: false,
    absoluteInternalPathsIncluded: false,
    restrictedCaseFilesIncluded: false,
    graderOnlyMaterialIncluded: false,
    rawDiagnosticsIncluded: false,
  },
  'attempt public projection',
);
check(
  claimBountyAttempt.stageOutcomes.scientificCommandsExecuted === 0,
  'zero scientific commands',
);
check(claimBountyAttempt.stageOutcomes.repairsApplied === 0, 'zero repairs');
check(claimBountyAttempt.stageOutcomes.downstreamStagesSkipped === 5, 'five downstream stages');
check(claimBountyAttempt.execution.parentInvocations === 1, 'one parent invocation');
check(claimBountyAttempt.execution.externalRestarts === 0, 'no external restart');
check(claimBountyAttempt.execution.elapsedSeconds === 3799.936556, 'attempt elapsed seconds');
check(
  JSON.stringify(claimBountyAttempt.protocolConformance.blockedGates) ===
    JSON.stringify(['G2', 'G3', 'G4']),
  'G2 G3 G4 blocked',
);
check(
  !/(run|evidence|gate)(Sha256|Hash)/i.test(JSON.stringify(claimBountyAttempt)),
  'no superseded internal seal hashes',
);
const attemptProjection = JSON.stringify(claimBountyAttempt);
check(!attemptProjection.includes('EC8359'), 'no raw attempt identifier');
check(
  !/\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b/i.test(
    attemptProjection,
  ),
  'no UUID in attempt projection',
);
check(
  !/(?:\/Users\/|\/home\/|\/var\/|\/tmp\/|\/private\/|\/Applications\/)/.test(attemptProjection),
  'no absolute path in attempt projection',
);

if (failures.length > 0) {
  for (const failure of failures) console.error(`comparator-disclosure: invalid ${failure}`);
  process.exit(1);
}

console.log(
  'comparator-disclosure: duplicate-free records, restricted-value redaction, and two-deviation exclusion are consistent',
);
