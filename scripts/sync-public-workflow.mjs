#!/usr/bin/env node

import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const routineNames = [
  'run-claimbounty-scientific-audit.md',
  'build-and-freeze-study-case.md',
  'reproduce-original-result.md',
  'research-methods-and-evidence.md',
  'stress-test-analysis.md',
  'verify-and-adjudicate-findings.md',
  'assemble-final-audit.md',
];

const resourceNames = [
  'config/audit-schema.json',
  'config/audit-template.html',
  'config/canonical-schemas.json',
  'execution/README.md',
  'execution/execution-manifest.schema.json',
  'execution/execution-profile.schema.json',
  'execution/runner.py',
  'execution/tests/test_runner.py',
];

const options = new Map();
for (let index = 2; index < process.argv.length; index += 2) {
  options.set(process.argv[index], process.argv[index + 1]);
}

const routineSource = options.get('--routine-source');
const resourceSource = options.get('--resource-source');
const outputRoot = options.get('--output') ?? 'workflow/claimbounty-scientific-audit';

if (!routineSource || !resourceSource) {
  console.error(
    'usage: sync-public-workflow.mjs --routine-source <directory> --resource-source <claimbounty-directory> [--output <directory>]',
  );
  process.exit(2);
}

const metadata =
  '> **Status**: Production | **Updated**: 2026-08-31 | **Scope**: Public ClaimBounty workflow export\n';

function addMetadataHeader(contents, expectedPrefix) {
  if (!contents.startsWith(expectedPrefix)) {
    throw new Error(`source does not start with ${expectedPrefix}`);
  }
  const firstNewline = contents.indexOf('\n');
  const title = contents.slice(0, firstNewline);
  const body = contents
    .slice(firstNewline + 1)
    .replace(/^\n> \*\*Status\*\*:.*\n/, '')
    .replace(/^- \*\*Routine Schema:\*\* \d+\n/m, '')
    .replace(/^\n+/, '');
  return `${title}\n\n${metadata}\n${body}`;
}

for (const name of routineNames) {
  const source = path.join(routineSource, name);
  const destination = path.join(outputRoot, 'routines', name);
  const contents = await readFile(source, 'utf8');
  await mkdir(path.dirname(destination), { recursive: true });
  await writeFile(destination, addMetadataHeader(contents, '# ROUTINE:'), 'utf8');
}

for (const name of resourceNames) {
  const source = path.join(resourceSource, name);
  const destination = path.join(outputRoot, 'project-template', 'claimbounty', name);
  const contents = await readFile(source);
  await mkdir(path.dirname(destination), { recursive: true });
  if (name.endsWith('.md')) {
    await writeFile(destination, addMetadataHeader(contents.toString('utf8'), '# '), 'utf8');
  } else {
    await writeFile(destination, contents);
  }
}

console.log(
  `workflow: synchronized ${routineNames.length} routines and ${resourceNames.length} resources`,
);
