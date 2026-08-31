#!/usr/bin/env node

import { readFile } from 'node:fs/promises';

const estimate = JSON.parse(await readFile('submission/evidence/human-estimate.json', 'utf8'));
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

check(estimate.classification === 'estimate_not_observation', 'estimate classification');
check(estimate.unit === 'active_person_hours', 'active-person-hour unit');
check(estimate.personCount === 1, 'one-person scope');
check(estimate.workPackages.length === 8, 'eight work packages');

for (const scenario of ['low', 'base', 'high']) {
  const total = estimate.workPackages.reduce((sum, item) => sum + item[scenario], 0);
  check(total === estimate.scenarios[scenario], `${scenario} work-package sum`);
}

check(estimate.scenarios.low === 21, '21-hour low estimate');
check(estimate.scenarios.base === 56, '56-hour base estimate');
check(estimate.scenarios.high === 124, '124-hour high estimate');
check(estimate.exclusions.includes('unattended computation'), 'unattended computation exclusion');
check(estimate.interpretation.observedDuration === false, 'observed-duration prohibition');
check(estimate.interpretation.humanQualityScore === false, 'human-quality-score prohibition');
check(estimate.interpretation.speedupClaimAllowed === false, 'speedup prohibition');
check(estimate.interpretation.unattendedComputeExcluded === true, 'unattended-compute marker');
check(estimate.sourcesAccessedOn === '2026-08-31', 'source access date');
check(estimate.sources.length === 9, 'nine cited sources');
check(
  estimate.sources.every((source) => source.url.startsWith('https://')),
  'linked sources',
);
check(
  estimate.sources.every((source) => source.accessedOn === '2026-08-31'),
  'per-source access dates',
);

const expectedSourceIds = [
  'aczel-2021',
  'prc-2016',
  'management-science-reproducibility',
  'colliard-cascad',
  'krafczyk-2021',
  'hoogeveen-2022',
  'brodeur-2026',
  'schindler-2024',
  'economic-journal-faq',
];
check(
  expectedSourceIds.every((id) => estimate.sources.some((source) => source.id === id)),
  'required source set',
);

const linkedDocuments = [
  'docs/BENCHMARK_RESULTS.md',
  'submission/reviewer/README.md',
  'submission/evidence/README.md',
];
for (const document of linkedDocuments) {
  const contents = await readFile(document, 'utf8');
  check(
    contents.includes('HUMAN_BASELINE.md') || contents.includes('human-estimate.json'),
    `human-estimate link in ${document}`,
  );
}

if (failures.length > 0) {
  for (const failure of failures) console.error(`human-estimate: invalid ${failure}`);
  process.exit(1);
}

console.log('human-estimate: eight work packages sum to 21/56/124 with nine cited sources');
