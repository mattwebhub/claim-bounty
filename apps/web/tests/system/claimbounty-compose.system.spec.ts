import { execFile } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';
import { access, mkdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext, type Page, type TestInfo } from '@playwright/test';

const execFileAsync = promisify(execFile);
const composeMode = Boolean(process.env.SYSTEM_TEST_WEB_ORIGIN);
const claimBountyRequired = process.env.SYSTEM_TEST_REQUIRE_CLAIMBOUNTY === '1';
const mailpitOrigin = process.env.SYSTEM_TEST_MAILPIT_ORIGIN ?? 'http://127.0.0.1:8025';
const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../..');

if (claimBountyRequired && !composeMode) {
  throw new Error(
    'The required ClaimBounty system gate must run through make test-system and the Compose profile.',
  );
}

interface MailpitAddress {
  Address: string;
}

interface MailpitMessageSummary {
  ID: string;
  To: MailpitAddress[];
}

interface MailpitMessageList {
  messages: MailpitMessageSummary[];
}

interface AdminOrderSnapshot {
  id: string;
  purpose: string;
  targetClaim: { text: string; sourceLocation?: string | null };
  permissions: { executeSuppliedCode: boolean; externalSearch: boolean };
  privacy: { containsParticipantLevelData: boolean; containsDirectIdentifiers: boolean };
  piiRetention: { sourceDeleteAfter: string; piiDeleteAfter: string };
  submittedAt: string;
  files: { id: string; role: string; originalDisplayName: string; status: string }[];
}

async function listMail(request: APIRequestContext) {
  const response = await request.get(`${mailpitOrigin}/api/v1/messages?limit=100`);
  expect(response.ok()).toBe(true);
  return (await response.json()) as MailpitMessageList;
}

async function waitForNewCode(
  request: APIRequestContext,
  recipient: string,
  knownMessageIds: Set<string>,
) {
  let code = '';
  await expect
    .poll(
      async () => {
        const list = await listMail(request);
        const message = list.messages.find(
          ({ ID, To }) =>
            !knownMessageIds.has(ID) &&
            To.some(({ Address }) => Address.toLowerCase() === recipient.toLowerCase()),
        );
        if (!message) return '';
        const response = await request.get(`${mailpitOrigin}/api/v1/message/${message.ID}`);
        expect(response.ok()).toBe(true);
        const content = JSON.stringify(await response.json());
        code = /\b[0-9]{6}\b/.exec(content)?.[0] ?? '';
        return code;
      },
      { timeout: 30_000, intervals: [250, 500, 1_000] },
    )
    .toMatch(/^[0-9]{6}$/);
  return code;
}

async function signInWithMailpit(page: Page, request: APIRequestContext, email: string) {
  const existing = await listMail(request);
  const knownMessageIds = new Set(existing.messages.map(({ ID }) => ID));
  await page.getByLabel('Email address').fill(email);
  await page.getByRole('button', { name: 'Send verification code' }).click();
  const code = await waitForNewCode(request, email, knownMessageIds);
  await page.getByLabel('Verification code').fill(code);
  await page.getByRole('button', { name: 'Verify and continue' }).click();
}

async function fillIntakeDocument(page: Page, summary: string, label: string, document: unknown) {
  const details = page
    .locator('details')
    .filter({ has: page.locator('summary', { hasText: summary }) });
  if (!(await details.getAttribute('open'))) await details.locator('summary').click();
  await page.getByLabel(label).fill(JSON.stringify(document, null, 2));
}

function buildAdminDocuments(order: AdminOrderSnapshot) {
  const primary = order.files.find(({ role }) => role === 'primary_paper');
  if (!primary) throw new Error('The scanned primary paper was not returned to the administrator.');
  const sourceLocation = order.targetClaim.sourceLocation ?? 'Primary paper';
  const primaryPath = `paper/${primary.id}-${primary.originalDisplayName}`;
  const auditRequest = {
    schemaVersion: '1.0.0',
    caseId: order.id,
    purpose: order.purpose,
    targetClaim: {
      claimId: 'primary-result',
      text: order.targetClaim.text,
      source: { artifact: primaryPath, location: sourceLocation },
      status: 'frozen',
    },
    permissions: {
      readUploadedFiles: true,
      executeSuppliedCode: order.permissions.executeSuppliedCode,
      createDerivedFiles: true,
      externalSearch: order.permissions.externalSearch,
      openAccessSourcesOnly: true,
      externalRedistributionAuthorized: false,
    },
    privacy: {
      classification: 'restricted_research',
      containsParticipantLevelData: order.privacy.containsParticipantLevelData,
      containsDirectIdentifiers: order.privacy.containsDirectIdentifiers,
      redactRowLevelDataFromReports: true,
    },
    retention: {
      policyVersion: 'claimbounty-p0.1',
      sourceDeleteAfter: order.piiRetention.sourceDeleteAfter,
      piiDeleteAfter: order.piiRetention.piiDeleteAfter,
      piiDisposition: 'hard_delete',
      preserveRunOutputs: true,
    },
    authority: {
      uploadsAuthorized: true,
      analysisUseAuthorized: true,
      externalRedistributionAuthorized: false,
      termsVersion: 'claimbounty-p0.1',
      customerConfirmedAt: order.submittedAt,
      frozenBy: order.id,
      frozenAt: order.submittedAt,
      authorizationPolicyVersion: 'admin-policy-v1',
      adminAllowlistVersion: 'admin-allowlist-v1',
    },
    releaseScope: 'internal',
  };
  const scientificPolicy = {
    schemaVersion: '1.0.0',
    policyVersion: 'scientific-intake-v1',
    defaultsVersion: 'claimbounty-defaults-v1',
    targetFreeze: {
      inferMissingScientificChoices: false,
      ambiguity: 'preserve_conflict_and_continue_with_limits',
    },
    reproduction: {
      comparisonProfile: 'reported-precision-v1',
      scientificChangesCountAsExact: false,
    },
    sensitivity: { maximumCandidates: 5, resultsBlindReview: true, reviewerCount: 2 },
    evidence: { maximumQuestions: 3, maximumDeepSources: 3 },
    verification: { independentRerun: true, maximumCorrectionRounds: 2 },
  };
  const executionPolicy = {
    schemaVersion: '1.0.0',
    policyVersion: 'local-operator-v1',
    runClass: 'manual_local_operator',
    releaseScope: 'internal',
    resources: {
      maximumCpuCores: 4,
      maximumMemoryMiB: 8192,
      maximumWorkingStorageMiB: 5120,
    },
    sandbox: {
      isolationRequired: true,
      networkDuringAnalysis: 'disabled',
      dependencyAcquisition: 'operator_approved',
      expandArchivesAutomatically: false,
    },
    sourceAccess: {
      externalSearch: order.permissions.externalSearch,
      openAccessOnly: true,
      paywallBypass: false,
    },
    privacy: {
      publishParticipantRows: false,
      publishDirectIdentifiers: false,
      reportsMayIncludeAggregateResults: true,
    },
    replay: {
      requireCleanEnvironment: true,
      requireInputAndOutputHashes: true,
      requireDependencyVersions: true,
    },
  };
  const routineContract = {
    routineId: 'claim-bounty-operations/run-claimbounty-scientific-audit',
    revision: `sha256:${'a'.repeat(64)}`,
    validation: {
      status: 'validated',
      validatedAt: '2026-08-30T11:50:00Z',
      evidenceSha256: 'b'.repeat(64),
    },
  };
  return { auditRequest, scientificPolicy, executionPolicy, routineContract };
}

async function verifyExportOffline(
  archivePath: string,
  expectedSha256: string,
  testInfo: TestInfo,
) {
  if (process.env.SYSTEM_TEST_VERIFY_EXPORT !== '1') return;
  const composeProject = process.env.SYSTEM_TEST_COMPOSE_PROJECT_NAME;
  if (!composeProject) throw new Error('SYSTEM_TEST_COMPOSE_PROJECT_NAME is required.');
  const verifiedRoot = testInfo.outputPath('offline-verified');
  await mkdir(verifiedRoot, { recursive: true });
  const composeFile = path.join(repositoryRoot, 'infra/compose.yaml');
  const archiveDirectory = path.dirname(archivePath);
  const archiveName = path.basename(archivePath);
  const { stdout, stderr } = await execFileAsync(
    'docker',
    [
      'compose',
      '--project-name',
      composeProject,
      '--file',
      composeFile,
      '--profile',
      'operator',
      'run',
      '--rm',
      '--entrypoint',
      '/api',
      '-v',
      `${archiveDirectory}:/system-downloads:ro`,
      '-v',
      `${verifiedRoot}:/system-verified`,
      'verify-export',
      'verify-export',
      `/system-downloads/${archiveName}`,
      expectedSha256,
      '/system-verified/claimbounty-export',
    ],
    { cwd: repositoryRoot, encoding: 'utf8', timeout: 180_000 },
  );
  expect(`${stdout}${stderr}`).toContain('case-bundle');
  const extracted = path.join(verifiedRoot, 'claimbounty-export');
  await access(path.join(extracted, 'case-bundle', 'CASE-MANIFEST.json'));
  await access(path.join(extracted, 'dispatch', 'audit-request.json'));
  await access(path.join(extracted, 'dispatch', 'scientific-policy.json'));
  await access(path.join(extracted, 'dispatch', 'execution-policy.json'));
}

test('completes ClaimBounty intake and offline handoff through the real Compose stack', async ({
  page,
  request,
}, testInfo) => {
  test.skip(
    !composeMode,
    'Requires make test-system and the complete ClaimBounty Compose profile.',
  );
  test.setTimeout(300_000);
  const suffix = randomUUID().slice(0, 8);
  const submitterEmail = `submitter-${suffix}@example.test`;
  const title = `System audit ${suffix}`;
  const sourceBytes = Buffer.from(
    '%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 0>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF',
  );

  await page.goto('/');
  await signInWithMailpit(page, request, submitterEmail);
  await expect(
    page.getByRole('heading', { name: 'Describe the claim and add evidence' }),
  ).toBeVisible();

  await page.getByLabel('Study title').fill(title);
  await page.getByLabel('What should the review establish?').fill('Verify the primary result.');
  await page
    .getByLabel('Exact target claim')
    .fill('The reported intervention increases retention.');
  await page.getByLabel('Claim location').fill('Abstract and Table 2');
  await page.locator('#evidence-files').setInputFiles({
    name: 'study.pdf',
    mimeType: 'application/pdf',
    buffer: sourceBytes,
  });
  await page.getByRole('button', { name: 'Create intake and upload' }).click();
  await expect(page.getByRole('listitem').filter({ hasText: 'study.pdf' })).toContainText(
    'complete',
  );
  const draftUrl = new URL(page.url());
  const orderId = draftUrl.searchParams.get('draft');
  expect(orderId).toMatch(/^[0-9a-f-]{36}$/);
  await page.getByLabel(/I accept the ClaimBounty/).check();
  await page.getByLabel(/retain and privately inspect/).check();
  await page.getByLabel(/private derived analysis files/).check();
  await page.getByRole('button', { name: 'Submit for review' }).click();
  await expect(page.getByRole('heading', { name: 'Your evidence is in review' })).toBeVisible();

  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(page).toHaveURL('/');
  await page.getByRole('link', { name: 'Admin' }).click();
  await signInWithMailpit(page, request, 'admin@example.test');
  await expect(page.getByRole('heading', { name: 'Research orders' })).toBeVisible();
  const orderRow = page.getByRole('row').filter({ hasText: title });
  await expect(orderRow).toBeVisible();
  await orderRow.getByRole('link', { name: 'Review' }).click();
  await expect(page.getByRole('heading', { name: title })).toBeVisible();

  await expect
    .poll(
      async () => {
        await page.reload();
        return page.getByRole('listitem').filter({ hasText: 'study.pdf' }).textContent();
      },
      { timeout: 180_000, intervals: [2_000] },
    )
    .toContain('clean');

  const sourceDownloadPromise = page.waitForEvent('download');
  await page.getByRole('link', { name: /^Download$/ }).click();
  const sourceDownload = await sourceDownloadPromise;
  const sourcePath = testInfo.outputPath('downloaded-study.pdf');
  await sourceDownload.saveAs(sourcePath);
  const downloadedSource = await readFile(sourcePath);
  expect(downloadedSource.byteLength).toBeGreaterThan(0);
  expect(downloadedSource.equals(sourceBytes)).toBe(true);

  if (!orderId) throw new Error('The public draft URL did not contain an order identifier.');
  const snapshotResponse = await page.request.get(`/api/v1/admin/orders/${orderId}`);
  expect(snapshotResponse.ok()).toBe(true);
  const snapshot = (await snapshotResponse.json()) as { data: AdminOrderSnapshot };
  const documents = buildAdminDocuments(snapshot.data);
  await fillIntakeDocument(page, 'Audit request', 'Audit request JSON', documents.auditRequest);
  await fillIntakeDocument(
    page,
    'Scientific policy',
    'Scientific policy JSON',
    documents.scientificPolicy,
  );
  await fillIntakeDocument(
    page,
    'Execution policy',
    'Execution policy JSON',
    documents.executionPolicy,
  );
  await fillIntakeDocument(
    page,
    'Validated routine contract',
    'Validated routine contract JSON',
    documents.routineContract,
  );
  await page.getByRole('button', { name: 'Validate and freeze intake' }).click();
  await expect(page.getByRole('status')).toContainText('Intake frozen and readiness recalculated.');

  await page.getByRole('checkbox', { name: /I reviewed the claim/ }).check();
  await page.getByRole('checkbox', { name: /I confirmed every included file/ }).check();
  await page.getByRole('button', { name: 'Create export' }).click();
  const exportLink = page.getByRole('link', { name: 'Download export' });
  await expect(exportLink).toBeVisible({ timeout: 180_000 });
  const exportHref = await exportLink.getAttribute('href');
  const exportId = /\/admin\/exports\/([0-9a-f-]+)\/download$/.exec(
    new URL(exportHref ?? '', page.url()).pathname,
  )?.[1];
  if (!exportId) throw new Error(`Export download URL did not contain an export ID: ${exportHref}`);
  const exportMetadataResponse = await page.request.get(
    `/api/v1/admin/orders/${orderId}/exports/${exportId}`,
  );
  expect(exportMetadataResponse.ok()).toBe(true);
  const exportMetadata = (await exportMetadataResponse.json()) as {
    data: { sha256?: string | null };
  };
  const expectedSha256 = exportMetadata.data.sha256;
  expect(expectedSha256).toMatch(/^[a-f0-9]{64}$/);
  if (!expectedSha256) throw new Error('Ready export metadata did not include sha256.');
  await expect(page.getByText(expectedSha256, { exact: true })).toBeVisible();
  const exportDownloadPromise = page.waitForEvent('download');
  const exportResponsePromise = page.waitForResponse(
    (response) => response.url().endsWith('/download') && response.status() === 200,
  );
  await exportLink.click();
  const [exportDownload, exportResponse] = await Promise.all([
    exportDownloadPromise,
    exportResponsePromise,
  ]);
  const contentDigest = exportResponse.headers()['content-digest'];
  const digestMatch = /^sha-256=:([A-Za-z0-9+/]{43}=):$/.exec(contentDigest ?? '');
  expect(digestMatch, `Content-Digest header ${contentDigest}`).not.toBeNull();
  const encodedDigest = digestMatch?.[1];
  if (!encodedDigest) throw new Error(`Invalid export Content-Digest: ${contentDigest}`);
  expect(Buffer.from(encodedDigest, 'base64').toString('hex')).toBe(expectedSha256);
  const archivePath = testInfo.outputPath('claimbounty-export.zip');
  await exportDownload.saveAs(archivePath);
  const downloadedArchive = await readFile(archivePath);
  expect(downloadedArchive.byteLength).toBeGreaterThan(0);
  expect(createHash('sha256').update(downloadedArchive).digest('hex')).toBe(expectedSha256);
  await verifyExportOffline(archivePath, expectedSha256, testInfo);
});
