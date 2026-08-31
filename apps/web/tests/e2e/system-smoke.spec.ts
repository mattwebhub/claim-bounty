import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page } from '@playwright/test';

const orderId = '123e4567-e89b-42d3-a456-426614174000';
const now = '2026-08-30T15:00:00Z';
const submitterSession = {
  audience: 'submitter',
  csrfToken: 'c'.repeat(32),
  authorizationPolicyVersion: 'claimbounty-p0.1',
  expiresAt: '2026-08-30T18:00:00Z',
};
const administratorSession = { ...submitterSession, audience: 'administrator' };
const order = {
  id: orderId,
  publicReference: 'CB-ABC123DEF456',
  status: 'needs_information',
  version: 4,
  title: 'Replication of a treatment effect',
  purpose: 'Check whether the reported effect reproduces.',
  targetClaim: { text: 'The treatment improved scores.', sourceLocation: 'Page 4, Table 2' },
  permissions: { executeSuppliedCode: false, externalSearch: true },
  privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
  files: [],
  piiRetention: {
    policyVersion: 'claimbounty-p0.1',
    disposition: 'hard_delete',
    sourceDeleteAfter: '2026-09-15T15:00:00Z',
    piiDeleteAfter: '2026-09-30T15:00:00Z',
  },
  createdAt: now,
  updatedAt: now,
  submittedAt: now,
};

async function mockSignedOutIntake(page: Page) {
  let verified = false;
  await page.route('**/api/v1/session', async (route) => {
    if (verified) {
      await route.fulfill({ json: { data: submitterSession } });
      return;
    }
    await route.fulfill({
      status: 401,
      json: { error: { code: 'unauthorized', message: 'Sign in required.' } },
    });
  });
  await page.route('**/api/v1/email-challenges', (route) =>
    route.fulfill({ status: 202, json: { data: { accepted: true } } }),
  );
  await page.route('**/api/v1/email-challenges/verify', async (route) => {
    expect(route.request().postDataJSON()).toEqual({
      email: 'researcher@example.org',
      audience: 'submitter',
      code: '123456',
    });
    verified = true;
    await route.fulfill({ json: { data: submitterSession } });
  });
}

async function mockAdmin(page: Page) {
  await page.route('**/api/v1/session', (route) =>
    route.fulfill({ json: { data: administratorSession } }),
  );
  await page.route(`**/api/v1/admin/orders/${orderId}`, (route) =>
    route.fulfill({
      json: {
        data: {
          ...order,
          submitterEmail: 'researcher@example.org',
          permissions: { executeSuppliedCode: false, externalSearch: true },
          privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
          frozenIntake: null,
          readinessIssues: [
            {
              code: 'intake_required',
              path: 'frozenIntake',
              message: 'Freeze the local handoff intake before export.',
            },
          ],
          events: [],
          exports: [],
        },
      },
    }),
  );
  await page.route('**/api/v1/admin/orders?**', (route) =>
    route.fulfill({ json: { data: { items: [order] } } }),
  );
}

test('public intake verifies email progressively and exposes one accessible file surface', async ({
  page,
}) => {
  await mockSignedOutIntake(page);
  await page.goto('/');

  await expect(
    page.getByRole('heading', {
      name: 'Find the weakness in your paper before reviewers do',
    }),
  ).toBeVisible();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('link', { name: /skip to main content/i })).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByLabel('Email address')).toHaveCount(0);
  await page.getByLabel('Manuscript and research files').setInputFiles([
    { name: 'paper.pdf', mimeType: 'application/pdf', buffer: Buffer.from('paper') },
    { name: 'data.csv', mimeType: 'text/csv', buffer: Buffer.from('a,b') },
  ]);
  await expect(
    page.getByRole('list', { name: 'Selected manuscript and research files' }),
  ).toBeVisible();
  await page.getByLabel('Email address').fill('researcher@example.org');
  await page.getByRole('button', { name: 'Send verification code' }).click();
  await expect(page.getByRole('heading', { name: 'Check your inbox' })).toBeFocused();
  await page.getByLabel('Verification code').fill('123456');
  await page.getByRole('button', { name: 'Verify and continue' }).click();

  await expect(page.getByRole('heading', { name: 'Tell us what to test' })).toBeVisible();
  await expect(page.getByLabel('Manuscript and research files')).toHaveCount(0);
  await expect(page.getByRole('group', { name: 'Evidence files' })).toBeVisible();
  await expect(page.getByText('paper.pdf')).toBeVisible();
  await expect(page.getByText('data.csv')).toBeVisible();
  await page.getByRole('button', { name: 'Remove data.csv' }).click();
  await expect(page.getByText('data.csv')).toBeHidden();

  const results = await new AxeBuilder({ page }).analyze();
  expect(
    results.violations.filter(({ impact }) => impact === 'critical' || impact === 'serious'),
  ).toEqual([]);
});

test('admin list and detail expose readiness without dispatch controls', async ({ page }) => {
  await mockAdmin(page);
  await page.goto('/admin');

  await expect(page.getByRole('heading', { name: 'Research orders' })).toBeVisible();
  await page.getByRole('link', { name: 'Review' }).click();
  await expect(page.getByRole('heading', { name: order.title })).toBeVisible();
  await expect(page.getByText('Export is not ready')).toBeVisible();
  await expect(page.getByText('Freeze the local handoff intake before export.')).toBeVisible();
  await expect(page.getByRole('button', { name: /dispatch/i })).toHaveCount(0);
  await expect(page.getByText(/payment/i)).toHaveCount(0);

  const results = await new AxeBuilder({ page }).analyze();
  expect(
    results.violations.filter(({ impact }) => impact === 'critical' || impact === 'serious'),
  ).toEqual([]);
});
