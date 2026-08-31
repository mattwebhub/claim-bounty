#!/usr/bin/env node

import { access, readFile } from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const documents = [
  'README.md',
  'THIRD_PARTY_NOTICES.md',
  'IMPROVEMENT_CHANGELOG.md',
  'SECURITY.md',
  'docs/SETUP.md',
  'docs/REPRODUCE.md',
  'docs/ARCHITECTURE.md',
  'docs/BENCHMARK_PROTOCOL.md',
  'docs/BENCHMARK_RESULTS.md',
  'docs/LIMITATIONS.md',
  'docs/PREEXISTING_WORK.md',
  'docs/TRAJECTORIES.md',
  'docs/VIDEO_SCRIPT.md',
  'docs/SUBMISSION.md',
  'docs/ASSET_PROVENANCE.md',
  'docs/HUMAN_BASELINE.md',
  'docs/USAGE_AND_COST.md',
  'submission/reviewer/README.md',
  'submission/evidence/README.md',
  'workflow/claimbounty-scientific-audit/README.md',
  'workflow/claimbounty-scientific-audit/trajectories/README.md',
  'workflow/claimbounty-scientific-audit/trajectories/sanitization-policy.md',
];

let failed = false;
for (const document of documents) {
  const contents = await readFile(document, 'utf8').catch(() => null);
  if (contents === null) {
    console.error(`public-links: missing document ${document}`);
    failed = true;
    continue;
  }

  const links = contents.matchAll(/!?\[[^\]]*\]\(([^)]+)\)/g);
  for (const match of links) {
    let target = match[1].trim();
    if (target.startsWith('<') && target.endsWith('>')) target = target.slice(1, -1);
    target = target.split('#', 1)[0];
    if (!target || /^(?:https?:|mailto:|project:)/.test(target)) continue;
    if (target.startsWith('/')) {
      console.error(`public-links: absolute local link in ${document}: ${target}`);
      failed = true;
      continue;
    }
    const resolved = path.normalize(path.join(path.dirname(document), decodeURIComponent(target)));
    await access(resolved).catch(() => {
      console.error(`public-links: missing target in ${document}: ${target}`);
      failed = true;
    });
  }
}

if (failed) process.exit(1);
console.log(`public-links: checked ${documents.length} public documents`);
