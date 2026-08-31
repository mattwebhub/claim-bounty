#!/usr/bin/env node

import { readFile } from 'node:fs/promises';

const usage = JSON.parse(await readFile('submission/evidence/usage-and-cost.json', 'utf8'));
const timing = JSON.parse(await readFile('submission/evidence/timing-observations.json', 'utf8'));
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

for (const run of usage.runs) {
  check(run.tokens.total === run.tokens.input + run.tokens.output, `${run.role} token total`);
  check(run.tokens.cachedInput <= run.tokens.input, `${run.role} cached input inclusion`);
  check(run.tokens.reasoningOutput <= run.tokens.output, `${run.role} reasoning output inclusion`);
}

const primary = usage.runs.find((run) => run.role === 'primary');
const secondary = usage.runs.find((run) => run.role === 'secondary');
check(Boolean(primary && secondary), 'primary and secondary run presence');

if (primary && secondary) {
  check(
    usage.combined.uniqueCliSessions === primary.uniqueCliSessions + secondary.uniqueCliSessions,
    'combined unique CLI sessions',
  );
  for (const field of ['input', 'cachedInput', 'output', 'reasoningOutput', 'total']) {
    check(
      usage.combined.tokens[field] === primary.tokens[field] + secondary.tokens[field],
      `combined ${field} tokens`,
    );
  }
}

check(usage.combined.tokens.total === 181802664, 'verified combined token total');
check(usage.combined.uniqueCliSessions === 109, 'verified combined session total');
check(
  usage.runtimeDisclosure.headlineScientificWorkflow.duration === 'PT6H49M6S',
  'headline workflow duration',
);
check(
  usage.runtimeDisclosure.parentThreadEnvelope.duration === 'PT8H12M30S',
  'parent-thread envelope duration',
);
check(timing.observedInternalWorkflow.headlineRuntime === true, 'timing headline marker');
check(timing.parentThreadEnvelope.headlineRuntime === false, 'timing envelope marker');
check(usage.monetaryCost.status === 'unmeasured', 'unmeasured monetary cost');
check(usage.monetaryCost.zeroDollarClaimSupported === false, 'zero-dollar claim prohibition');
check(
  usage.monetaryCost.retroactiveCurrentApiPricingAllowed === false,
  'retroactive pricing prohibition',
);

if (failures.length > 0) {
  for (const failure of failures) console.error(`usage-disclosure: invalid ${failure}`);
  process.exit(1);
}

console.log('usage-disclosure: verified session, token, runtime, and cost accounting');
