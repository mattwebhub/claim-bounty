#!/usr/bin/env node

import { readFile } from 'node:fs/promises';

const workflow = await readFile('.github/workflows/release.yml', 'utf8');
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

const strictSemver =
  /^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/;
for (const valid of ['v1.2.3', 'v1.2.3-rc.1', 'v1.2.3+build.7', 'v0.0.0-a+z']) {
  check(strictSemver.test(valid), `strict SemVer accepts ${valid}`);
}
for (const invalid of ['v01.2.3', 'v1.02.3', 'v1.2.03', 'v1.2.3-01', 'v1.2.3+', '1.2.3']) {
  check(!strictSemver.test(invalid), `strict SemVer rejects ${invalid}`);
}
const identityJob = workflow.indexOf('\n  release-identity:');
const containersJob = workflow.indexOf('\n  containers:');
const digestOnlyPush = workflow.indexOf('push-by-digest=true,name-canonical=true,push=true');
const publishJob = workflow.indexOf('\n  publish:');
const existingReleaseRefusal = workflow.indexOf(
  'already exists; refusing container registry mutation',
);

check(identityJob >= 0, 'release identity job exists');
check(containersJob > identityJob, 'release identity job is declared before containers');
check(digestOnlyPush > identityJob, 'release identity guard precedes digest-only image push');
check(
  workflow.includes('needs: [scientific-runner, release-identity]'),
  'containers depend on release identity guard',
);
check(
  existingReleaseRefusal > identityJob && existingReleaseRefusal < containersJob,
  'existing release blocks registry mutation',
);
check(
  workflow.includes('[0-9]*[A-Za-z-][0-9A-Za-z-]*'),
  'SemVer prerelease numeric identifiers reject leading zeroes',
);
check(
  workflow.includes('${#RELEASE_TAG}') && workflow.includes('-gt 80'),
  'SemVer input length remains within the injective Docker-tag encoding domain',
);

const releaseIdentityBlock = workflow.slice(identityJob, containersJob);
const containerBlock = workflow.slice(containersJob, publishJob);
const publishBlock = workflow.slice(publishJob);
for (const [name, block] of [
  ['release identity', releaseIdentityBlock],
  ['release publication', publishBlock],
]) {
  check(!block.includes('|| true'), `${name} probes do not discard failures with || true`);
  check(!block.includes('2>/dev/null'), `${name} probes preserve registry and API stderr`);
}
check(
  (workflow.match(/cannot prove GitHub release absence/g) ?? []).length === 2,
  'both GitHub release existence probes fail closed on non-404 errors',
);
for (const forbidden of [
  'imagetools create',
  '--tag',
  '\n          tags:',
  'type=semver',
  'type=sha',
  'docker/metadata-action',
]) {
  check(
    !containerBlock.includes(forbidden),
    `container job does not publish tags via ${forbidden}`,
  );
}
check(
  containerBlock.includes('printf \'%s@%s\\n\' "$IMAGE" "$SUBJECT_DIGEST"'),
  'release records an exact pull-by-digest reference',
);
check(
  containerBlock.includes('name: release-container-${{ matrix.image }}'),
  'pull-by-digest reference is uploaded as a release artifact',
);
check(
  containerBlock.includes('subject-digest: ${{ steps.build.outputs.digest }}'),
  'container provenance attestation binds the content digest',
);
check(
  publishBlock.includes('! -name SHA256SUMS'),
  'release checksums include digest-reference artifacts',
);

if (failures.length > 0) {
  for (const failure of failures) console.error(`release-workflow: invalid ${failure}`);
  process.exit(1);
}

console.log('release-workflow: identity guard and digest-only container publication are valid');
