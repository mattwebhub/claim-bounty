#!/usr/bin/env node

import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';

const archivalPath = 'apps/web/public/claimbounty-archival-projection-v3.png';
const expectedHash = 'bf4f4154c7c10f75aa382a516facffcc44354d5c33ebf066aa01dc61f33a28ef';
const expectedIgnored = [
  '/apps/web/public/claimbounty-anatomical-brain.png',
  '/apps/web/public/claimbounty-proof-loupe-icon.png',
];
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

const archival = await readFile(archivalPath);
check(archival.subarray(1, 4).toString('ascii') === 'PNG', 'archival projection PNG signature');
check(archival.subarray(12, 16).toString('ascii') === 'IHDR', 'archival projection IHDR');
check(archival.readUInt32BE(16) === 1448, 'archival projection width');
check(archival.readUInt32BE(20) === 1086, 'archival projection height');
check(createHash('sha256').update(archival).digest('hex') === expectedHash, 'archival SHA-256');

const provenance = await readFile('docs/ASSET_PROVENANCE.md', 'utf8');
for (const fact of [
  archivalPath,
  '1448 by 1086 pixels',
  expectedHash,
  'built-in image-generation tool',
  'Wellcome public-domain page scan',
  'Public Domain Mark',
  'No Getty or VintageMedStock raster was supplied to this generation',
  'pending direct user confirmation before publication',
]) {
  check(provenance.includes(fact), `asset provenance fact: ${fact}`);
}

const notices = await readFile('THIRD_PARTY_NOTICES.md', 'utf8');
check(notices.includes('active archival projection'), 'archival projection notice');
check(notices.includes('Public Domain Mark'), 'archival public-domain notice');
check(notices.includes('excluded from the public application'), 'excluded prior derivative notice');

const route = await readFile('apps/web/src/routes/home/home.route.tsx', 'utf8');
const styles = await readFile('apps/web/src/shared/ui/styles.css', 'utf8');
check(route.includes('/claimbounty-archival-projection-v3.png'), 'active archival route usage');
for (const styleFact of [
  '.brain-archival-projection',
  'blur(',
  'contrast(',
  'brightness(',
  'animation:',
]) {
  check(styles.includes(styleFact), `archival browser presentation: ${styleFact}`);
}

const ignoreLines = new Set((await readFile('.gitignore', 'utf8')).split(/\r?\n/));
for (const ignored of expectedIgnored) {
  check(ignoreLines.has(ignored), `exact rejected-asset ignore entry: ${ignored}`);
}

for (const document of [
  'README.md',
  'IMPROVEMENT_CHANGELOG.md',
  'THIRD_PARTY_NOTICES.md',
  'docs/ASSET_PROVENANCE.md',
  'docs/VIDEO_SCRIPT.md',
  'submission/reviewer/README.md',
]) {
  const contents = await readFile(document, 'utf8');
  for (const ignored of expectedIgnored) {
    check(!contents.includes(ignored.slice(1)), `no rejected-asset dependency in ${document}`);
  }
}

if (failures.length > 0) {
  for (const failure of failures) console.error(`public-assets: invalid ${failure}`);
  process.exit(1);
}

console.log(
  'public-assets: selected assets, provenance, and rejected-asset exclusions are consistent',
);
