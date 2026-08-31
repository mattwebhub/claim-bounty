import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page } from '@playwright/test';

const session = {
  audience: 'submitter',
  csrfToken: 'c'.repeat(32),
  authorizationPolicyVersion: 'policy.1',
  expiresAt: '2026-08-30T18:00:00Z',
};

const baseOrder = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  publicReference: 'CB-ABCDEF123456',
  status: 'draft',
  version: 1,
  title: 'Retention intervention',
  purpose: 'Check the reported effect',
  targetClaim: { text: 'Retention increased', sourceLocation: 'Table 2' },
  permissions: { executeSuppliedCode: false, externalSearch: false },
  privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
  files: [],
  piiRetention: {
    policyVersion: 'policy.1',
    disposition: 'hard_delete',
    sourceDeleteAfter: '2026-09-15T10:00:00Z',
    piiDeleteAfter: '2026-09-30T10:00:00Z',
  },
  createdAt: '2026-08-30T10:00:00Z',
  updatedAt: '2026-08-30T10:00:00Z',
};

const uploadedFile = {
  id: '223e4567-e89b-12d3-a456-426614174000',
  role: 'primary_paper',
  originalDisplayName: 'paper.pdf',
  sizeBytes: 13,
  sha256: 'a'.repeat(64),
  storage: { objectVersion: 'generation-1', sha256: 'a'.repeat(64), immutability: 'write_once' },
  declaredMediaType: 'application/pdf',
  status: 'uploaded',
  createdAt: '2026-08-30T10:01:00Z',
  updatedAt: '2026-08-30T10:01:00Z',
};

const uploadedDataFile = {
  ...uploadedFile,
  id: '323e4567-e89b-12d3-a456-426614174000',
  role: 'data',
  originalDisplayName: 'results.csv',
  declaredMediaType: 'text/csv',
};

async function expectNoSeriousViolations(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  expect(
    results.violations.filter(({ impact }) => impact === 'critical' || impact === 'serious'),
  ).toEqual([]);
}

test('email challenge progressively opens the accessible intake', async ({ page }) => {
  let verified = false;
  await page.route('**/api/v1/session', async (route) => {
    if (verified) await route.fulfill({ json: { data: session } });
    else
      await route.fulfill({
        status: 401,
        json: { error: { code: 'unauthorized', message: 'Sign in required.' } },
      });
  });
  await page.route('**/api/v1/email-challenges', (route) =>
    route.fulfill({ status: 202, json: { data: { accepted: true } } }),
  );
  await page.route('**/api/v1/email-challenges/verify', async (route) => {
    verified = true;
    await route.fulfill({ json: { data: session } });
  });

  await page.goto('/');
  await expect(
    page.getByRole('heading', {
      name: 'Find the weakness in your paper before reviewers do',
    }),
  ).toBeVisible();
  await expect(page.getByLabel('Email address')).toHaveCount(0);
  await page.getByLabel('Manuscript and research files').setInputFiles({
    name: 'paper.pdf',
    mimeType: 'application/pdf',
    buffer: Buffer.from('%PDF-1.7 test'),
  });
  await page.getByLabel('Email address').fill('researcher@example.org');
  await page.getByRole('button', { name: 'Send verification code' }).click();
  await page.getByLabel('Verification code').fill('123456');
  await page.getByRole('button', { name: 'Verify and continue' }).click();

  await expect(page.getByRole('heading', { name: 'Tell us what to test' })).toBeVisible();
  await expect(page.getByLabel('Manuscript and research files')).toHaveCount(0);
  await expect(page.getByRole('group', { name: 'Evidence files' })).toBeVisible();
  await expectNoSeriousViolations(page);
});

test('uploads a primary PDF and produces a receipt', async ({ page }) => {
  await page.route('**/api/v1/session', (route) => route.fulfill({ json: { data: session } }));
  await page.route('**/api/v1/orders**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (request.method() === 'POST' && path.endsWith('/files')) {
      await route.fulfill({
        status: 201,
        headers: { ETag: '"2"' },
        json: { data: uploadedFile },
      });
      return;
    }
    if (request.method() === 'POST' && path.endsWith('/submit')) {
      await route.fulfill({
        status: 202,
        json: {
          data: {
            ...baseOrder,
            version: 3,
            status: 'submitted',
            files: [uploadedFile],
            submittedAt: '2026-08-30T10:02:00Z',
          },
        },
      });
      return;
    }
    await route.fulfill({ status: 201, headers: { ETag: '"1"' }, json: { data: baseOrder } });
  });

  await page.goto('/');
  await page.getByLabel('Study title').fill(baseOrder.title);
  await page.getByLabel('What should the review establish?').fill(baseOrder.purpose);
  await page.getByLabel('Exact target claim').fill(baseOrder.targetClaim.text);
  await page.getByLabel('Claim location').fill(baseOrder.targetClaim.sourceLocation);
  await page.locator('#evidence-files').setInputFiles({
    name: 'paper.pdf',
    mimeType: 'application/pdf',
    buffer: Buffer.from('%PDF-1.7 test'),
  });
  await page.getByRole('button', { name: 'Create intake and upload' }).click();
  await expect(page.getByText(/complete$/)).toBeVisible();
  await page.getByLabel(/I accept the ClaimBounty/).check();
  await page.getByLabel(/retain and privately inspect/).check();
  await page.getByLabel(/private derived analysis files/).check();
  await page.getByRole('button', { name: 'Submit for review' }).click();

  await expect(page.getByRole('heading', { name: 'Your evidence is in review' })).toBeVisible();
  await expect(page.getByText('CB-ABCDEF123456')).toBeVisible();
  await expect(page).not.toHaveURL(/draft=/);
});

test('refresh restores one successful upload and lets a failed upload resume on the same draft', async ({
  page,
}) => {
  let createCount = 0;
  let uploadCount = 0;
  await page.route('**/api/v1/session', (route) => route.fulfill({ json: { data: session } }));
  await page.route('**/api/v1/orders**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (request.method() === 'GET') {
      await route.fulfill({
        headers: { ETag: '"2"' },
        json: {
          data: {
            ...baseOrder,
            status: 'uploading',
            version: 2,
            files: uploadCount > 0 ? [uploadedFile] : [],
          },
        },
      });
      return;
    }
    if (path.endsWith('/files')) {
      uploadCount += 1;
      if (uploadCount === 2) {
        await route.fulfill({
          status: 503,
          json: { error: { message: 'Upload connection interrupted.', requestId: 'req-upload-2' } },
        });
        return;
      }
      await route.fulfill({
        status: 201,
        headers: { ETag: `"${uploadCount + 1}"` },
        json: { data: uploadCount === 1 ? uploadedFile : uploadedDataFile },
      });
      return;
    }
    createCount += 1;
    await route.fulfill({
      status: 201,
      headers: { ETag: '"1"' },
      json: { data: baseOrder },
    });
  });

  await page.goto('/');
  await page.getByLabel('Study title').fill(baseOrder.title);
  await page.getByLabel('What should the review establish?').fill(baseOrder.purpose);
  await page.getByLabel('Exact target claim').fill(baseOrder.targetClaim.text);
  await page.getByLabel('Claim location').fill(baseOrder.targetClaim.sourceLocation);
  await page.locator('#evidence-files').setInputFiles([
    {
      name: 'paper.pdf',
      mimeType: 'application/pdf',
      buffer: Buffer.from('%PDF-1.7 test'),
    },
    { name: 'results.csv', mimeType: 'text/csv', buffer: Buffer.from('x,y\n1,2') },
  ]);
  await page.getByRole('button', { name: 'Create intake and upload' }).click();

  await expect(page.getByRole('alert')).toContainText('Upload connection interrupted.');
  await expect(page).toHaveURL(new RegExp(`draft=${baseOrder.id}`));
  expect(createCount).toBe(1);
  expect(uploadCount).toBe(2);

  await page.reload();
  const restored = page.getByRole('list', { name: 'Restored evidence files' });
  await expect(restored.getByText('paper.pdf')).toBeVisible();
  await expect(page.getByText(/did not finish uploading cannot be restored/)).toBeVisible();

  await page.locator('#evidence-files').setInputFiles({
    name: 'results.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from('x,y\n1,2'),
  });
  await page.getByRole('button', { name: 'Upload selected files' }).click();
  await expect(page.getByRole('listitem').filter({ hasText: 'results.csv' })).toContainText(
    'complete',
  );
  expect(createCount).toBe(1);
  expect(uploadCount).toBe(3);
});

test('protected admin list is responsive and accessible', async ({ page }) => {
  await page.route('**/api/v1/session', (route) =>
    route.fulfill({ json: { data: { ...session, audience: 'administrator' } } }),
  );
  await page.route('**/api/v1/admin/orders**', (route) =>
    route.fulfill({
      json: { data: { items: [{ ...baseOrder, status: 'scanning' }] } },
    }),
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/admin?status=scanning');

  await expect(page.getByRole('heading', { name: 'Research orders' })).toBeVisible();
  await expect(page.getByText('CB-ABCDEF123456')).toBeVisible();
  await expect(page.getByLabel('Status')).toHaveValue('scanning');
  await expectNoSeriousViolations(page);
});
