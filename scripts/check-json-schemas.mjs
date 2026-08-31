import { readFile } from 'node:fs/promises';
import process from 'node:process';

import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';

const pairs = [
  ['contracts/schemas/v1/case-manifest.schema.json', 'contracts/examples/v1/CASE-MANIFEST.json'],
  ['contracts/schemas/v1/audit-request.schema.json', 'contracts/examples/v1/audit-request.json'],
  [
    'contracts/schemas/v1/scientific-policy.schema.json',
    'contracts/examples/v1/scientific-policy.json',
  ],
  [
    'contracts/schemas/v1/execution-policy.schema.json',
    'contracts/examples/v1/execution-policy.json',
  ],
];

const unsafeMutations = new Map([
  [
    'contracts/schemas/v1/case-manifest.schema.json',
    [
      (value) => {
        value.files[0].path = 'case-bundle/../escape.pdf';
      },
      (value) => {
        value.manifestPath = 'CASE-MANIFEST.json';
      },
      (value) => {
        value.routineContract.validation.status = 'exception';
      },
      (value) => {
        delete value.files[0].objectVersion;
      },
      (value) => {
        value.files[0].storageImmutability = 'mutable';
      },
      (value) => {
        value.authority.externalRedistributionAuthorized = true;
      },
    ],
  ],
  [
    'contracts/schemas/v1/audit-request.schema.json',
    [
      (value) => {
        value.releaseScope = 'public';
      },
      (value) => {
        value.retention.piiDisposition = 'retain';
      },
      (value) => {
        delete value.authority.adminAllowlistVersion;
      },
      (value) => {
        value.targetClaim.source.artifact = 'case-bundle/paper/study.pdf';
      },
      (value) => {
        value.authority.externalRedistributionAuthorized = true;
      },
    ],
  ],
  [
    'contracts/schemas/v1/scientific-policy.schema.json',
    [
      (value) => {
        value.estimand = { customerSelected: true };
      },
    ],
  ],
  [
    'contracts/schemas/v1/execution-policy.schema.json',
    [
      (value) => {
        value.command = 'customer-supplied-command';
      },
    ],
  ],
]);

const readJson = async (path) => JSON.parse(await readFile(path, 'utf8'));
const ajv = new Ajv2020({ allErrors: true, strict: true });
addFormats(ajv);

let failed = false;

for (const [schemaPath, examplePath] of pairs) {
  try {
    const schema = await readJson(schemaPath);
    const example = await readJson(examplePath);
    const validate = ajv.compile(schema);

    if (!validate(example)) {
      failed = true;
      console.error(`${examplePath} does not satisfy ${schemaPath}`);
      console.error(ajv.errorsText(validate.errors, { separator: '\n' }));
      continue;
    }

    const missingVersion = structuredClone(example);
    delete missingVersion.schemaVersion;
    if (validate(missingVersion)) {
      failed = true;
      console.error(`${schemaPath} accepted an instance without schemaVersion`);
      continue;
    }

    for (const [index, mutate] of unsafeMutations.get(schemaPath).entries()) {
      const unsafe = structuredClone(example);
      mutate(unsafe);
      if (validate(unsafe)) {
        failed = true;
        console.error(`${schemaPath} accepted unsafe negative case ${index + 1}`);
      }
    }

    console.log(`schema: ${schemaPath} and its example are valid`);
  } catch (error) {
    failed = true;
    console.error(`schema: failed to compile or read ${schemaPath}`);
    console.error(error instanceof Error ? error.message : error);
  }
}

if (failed) process.exit(1);
