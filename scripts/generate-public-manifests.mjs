#!/usr/bin/env node

import { createHash } from 'node:crypto';
import { readdir, readFile, stat, writeFile } from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const packageRoots = [
  'submission/reviewer',
  'submission/evidence',
  'workflow/claimbounty-scientific-audit',
];
const check = process.argv.includes('--check');

function excludedPythonCache(relativePath, isDirectory) {
  const normalized = relativePath.split(path.sep).join('/');
  return isDirectory
    ? normalized.split('/').includes('__pycache__')
    : /\.(?:pyc|pyo|pyd)$/i.test(normalized);
}

for (const [candidate, isDirectory] of [
  ['execution/__pycache__', true],
  ['execution/__pycache__/runner.cpython-314.pyc', false],
  ['execution/runner.pyc', false],
  ['execution/runner.pyo', false],
  ['execution/runner.pyd', false],
]) {
  if (!excludedPythonCache(candidate, isDirectory)) {
    throw new Error(`public-package: Python cache exclusion regression for ${candidate}`);
  }
}

async function filesBelow(root, current = root) {
  const entries = await readdir(current, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const absolute = path.join(current, entry.name);
    const relative = path.relative(root, absolute);
    if (excludedPythonCache(relative, entry.isDirectory())) continue;
    if (entry.isDirectory()) files.push(...(await filesBelow(root, absolute)));
    if (entry.isFile()) files.push(relative.split(path.sep).join('/'));
  }
  return files.sort();
}

function digest(contents) {
  return createHash('sha256').update(contents).digest('hex');
}

async function expectedPackage(root) {
  const files = (await filesBelow(root)).filter(
    (file) => file !== 'manifest.json' && file !== 'MANIFEST.sha256',
  );
  const entries = [];
  for (const file of files) {
    const absolute = path.join(root, file);
    const contents = await readFile(absolute);
    const info = await stat(absolute);
    entries.push({ path: file, bytes: info.size, sha256: digest(contents) });
  }
  const manifest = `${JSON.stringify(
    {
      schemaVersion: '1.0.0',
      packageRoot: root,
      pathPolicy: 'repository-relative',
      entries,
    },
    null,
    2,
  )}\n`;
  const checksumLines = [
    ...entries.map(({ path: file, sha256 }) => `${sha256}  ${file}`),
    `${digest(manifest)}  manifest.json`,
  ];
  return { manifest, checksums: `${checksumLines.sort().join('\n')}\n` };
}

let failed = false;
for (const root of packageRoots) {
  const expected = await expectedPackage(root);
  if (check) {
    const currentManifest = await readFile(path.join(root, 'manifest.json'), 'utf8').catch(
      () => '',
    );
    const currentChecksums = await readFile(path.join(root, 'MANIFEST.sha256'), 'utf8').catch(
      () => '',
    );
    if (currentManifest !== expected.manifest || currentChecksums !== expected.checksums) {
      console.error(`public-package: generated manifest drift in ${root}`);
      failed = true;
    }
  } else {
    await writeFile(path.join(root, 'manifest.json'), expected.manifest, 'utf8');
    await writeFile(path.join(root, 'MANIFEST.sha256'), expected.checksums, 'utf8');
    console.log(`public-package: generated ${root}`);
  }
}

if (failed) process.exit(1);
if (check) console.log('public-package: manifests and SHA-256 files are current');
