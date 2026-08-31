#!/usr/bin/env node

import { readFile } from 'node:fs/promises';

const changelog = await readFile('IMPROVEMENT_CHANGELOG.md', 'utf8');
const readme = await readFile('README.md', 'utf8');
const video = await readFile('docs/VIDEO_SCRIPT.md', 'utf8');
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

const experimentMatches = [...changelog.matchAll(/^### Experiment \d+:/gm)];
check(experimentMatches.length === 7, 'seven material experiment entries');

const requiredFields = [
  '- Approach / hypothesis:',
  '- Fixed evaluation or review method:',
  '- Observed result:',
  '- Keep / change / remove decision:',
  '- Public evidence path:',
];
for (let index = 0; index < experimentMatches.length; index += 1) {
  const start = experimentMatches[index].index;
  const end = experimentMatches[index + 1]?.index ?? changelog.indexOf('\n## Main failure mode');
  const section = changelog.slice(start, end);
  for (const field of requiredFields) {
    check(section.includes(field), `${field} in experiment ${index + 1}`);
  }
}

const requiredFacts = [
  '785.539676 seconds',
  '`truncated_or_malformed_output`',
  'Alpha scored it 1/100',
  'Beta 2/100',
  'decision `invalid_incomplete`',
  'no follow-up',
  '6h49m06s',
  '`audit_completed_with_limits`',
  'research completion-envelope failure',
  'verification-before-delivery sequence',
  'renderer-selection defect',
  'clean replay',
  '`partial_reproduction`',
  'continuation gate',
  'Scientific execution remains outside the hosted boundary',
  '7-routine/19-current-agent/47-step',
  '13-page PDF',
  'qualitative user feedback',
  '3799.936556 seconds',
  '`case_frozen_with_limits`',
  '`not_executable`',
  '1,275-byte task contract omitted the frozen 1,276-byte prompt file',
  'two case-specific values from outside the frozen participant bundle',
  'values themselves are omitted',
  'out-of-contract contamination',
  'G2, G3, and G4 are `BLOCKED`',
];
for (const fact of requiredFacts) {
  check(changelog.includes(fact), `required disclosure: ${fact}`);
}

check(changelog.includes('## Main failure mode'), 'main failure mode section');
check(changelog.includes('## Hot take'), 'hot take section');
check(
  changelog.includes('Analysis that never becomes a traceable reviewer report is missing work'),
  'evidence-grounded hot take',
);
const internalOrganizationPattern = new RegExp(`\\.${'to' + 'one'}\\/`, 'i');
const absoluteLocalPattern = new RegExp(
  `(?:^|[\\s"'\`])(?:/${'Users'}/|/home/|/var/|/tmp/|/private/|/Applications/|[A-Za-z]:\\\\)`,
  'm',
);
check(!internalOrganizationPattern.test(changelog), 'no internal organization path');
check(!absoluteLocalPattern.test(changelog), 'no absolute local path');
check(
  !/\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b/i.test(changelog),
  'no internal run or invocation UUID',
);

for (const heading of [
  '## Who it is for',
  '## The problem',
  '## How Peer2Paper works',
  '## What the product delivers',
  '## Product architecture',
]) {
  check(readme.includes(heading), `README narrative heading: ${heading}`);
}
check(
  readme.includes('[Improvement Changelog](IMPROVEMENT_CHANGELOG.md)'),
  'README changelog link',
);
check(readme.includes('Researcher intake'), 'README product intake boundary');
check(readme.includes('Local scientific workflow'), 'README local execution boundary');
check(readme.includes('decision package'), 'README product output');
check(!/hackathon/i.test(readme), 'README excludes event framing');

const shotPattern = /^\| (\d):(\d{2})–(\d):(\d{2}) \|/gm;
const shots = [...video.matchAll(shotPattern)].map((match) => ({
  start: Number(match[1]) * 60 + Number(match[2]),
  end: Number(match[3]) * 60 + Number(match[4]),
}));
check(shots.length === 8, 'eight video shots');
check(shots[0]?.start === 0, 'video starts at zero');
check(shots.at(-1)?.end <= 300, 'video ends within five minutes');
check(
  shots.every((shot, index) => index === 0 || shot.start === shots[index - 1].end),
  'video shots are contiguous',
);
for (const fact of [
  'On screen',
  'Narration',
  'final public protocol-deviation card',
  'two-deviation blocked-gate summary',
  'evidence-gated stages with explicit continuation checks',
  'Removed direction: overly detailed tactile loupe',
  'docs/REPRODUCE.md',
  '7-routine/19-current-agent trajectory index',
  'submission/reviewer/peer2paper-product-guide.pdf',
  '`make public-release` passing',
]) {
  check(video.includes(fact), `video requirement: ${fact}`);
}

if (failures.length > 0) {
  for (const failure of failures) console.error(`improvement-changelog: invalid ${failure}`);
  process.exit(1);
}

console.log(
  'improvement-changelog: 7 evidence-linked experiments, README narrative, and 4:50 video plan',
);
