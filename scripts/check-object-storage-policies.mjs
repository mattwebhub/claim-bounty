import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const apiPolicyPath = 'infra/object-storage/api-policy.json';
const demoPoliciesPath = 'infra/kubernetes/overlays/demo/object-storage-policies.yaml';

const expectedApiPolicy = {
  Version: '2012-10-17',
  Statement: [
    {
      Effect: 'Allow',
      Action: ['s3:PutObject', 's3:DeleteObject', 's3:DeleteObjectVersion'],
      Resource: ['arn:aws:s3:::claimbounty-private/quarantine/*'],
    },
    {
      Effect: 'Allow',
      Action: ['s3:GetObject', 's3:GetObjectVersion'],
      Resource: [
        'arn:aws:s3:::claimbounty-private/accepted/*',
        'arn:aws:s3:::claimbounty-private/exports/*',
      ],
    },
  ],
};

function embeddedJson(document, key) {
  const lines = document.split('\n');
  const marker = `  ${key}: |`;
  const start = lines.indexOf(marker);
  assert.notEqual(start, -1, `${demoPoliciesPath} is missing ${key}`);

  const content = [];
  for (const line of lines.slice(start + 1)) {
    if (line.startsWith('    ')) {
      content.push(line.slice(4));
      continue;
    }
    if (line === '') {
      content.push(line);
      continue;
    }
    break;
  }
  return JSON.parse(content.join('\n'));
}

const apiPolicy = JSON.parse(await readFile(apiPolicyPath, 'utf8'));
const demoPolicies = await readFile(demoPoliciesPath, 'utf8');
const demoApiPolicy = embeddedJson(demoPolicies, 'api-policy.json');

assert.deepEqual(
  apiPolicy,
  expectedApiPolicy,
  `${apiPolicyPath} must grant only quarantine write/delete and accepted/export read`,
);
assert.deepEqual(
  demoApiPolicy,
  expectedApiPolicy,
  `${demoPoliciesPath} API policy must match ${apiPolicyPath}`,
);

console.log('object-storage-policy: API scope is synchronized and least privilege');
