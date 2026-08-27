import { randomUUID } from 'node:crypto';
import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page } from '@playwright/test';

async function expectNoSeriousAccessibilityViolations(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  expect(
    results.violations.filter(({ impact }) => impact === 'critical' || impact === 'serious'),
  ).toEqual([]);
}

test('creates a project and persists a workspace through the real stack', async ({ page }) => {
  const projectName = `System project ${randomUUID().slice(0, 8)}`;

  await page.goto('/projects');
  await expect(page.getByRole('heading', { level: 1, name: 'Projects' })).toBeVisible();

  await page.getByLabel('Project name').fill(projectName);
  const createResponsePromise = page.waitForResponse(
    (response) =>
      response.request().method() === 'POST' &&
      new URL(response.url()).pathname === '/api/v1/projects',
  );
  await page.getByRole('button', { name: 'Create project' }).click();

  const createResponse = await createResponsePromise;
  expect(createResponse.status()).toBe(201);
  await expect(page.getByRole('status')).toContainText(`${projectName} was created.`);

  const projectCard = page
    .getByRole('article')
    .filter({ has: page.getByRole('heading', { name: projectName }) });
  await expect(projectCard).toBeVisible();
  await expectNoSeriousAccessibilityViolations(page);

  await projectCard.getByRole('link', { name: 'View project' }).click();
  await expect(page.getByRole('heading', { name: projectName })).toBeVisible();
  await page.getByRole('link', { name: 'Open workspace' }).click();

  await expect(page.getByRole('heading', { name: 'Your workspace is ready' })).toBeVisible();
  await page.getByRole('button', { name: /Add Note/ }).click();
  await page.keyboard.press('ArrowRight');

  const saveResponsePromise = page.waitForResponse(
    (response) =>
      response.request().method() === 'PUT' &&
      new URL(response.url()).pathname.endsWith('/workspace'),
  );
  await page.getByRole('button', { name: 'Save', exact: true }).click();

  const saveResponse = await saveResponsePromise;
  expect(saveResponse.status()).toBe(200);
  expect(saveResponse.request().headers()['if-match']).toBe('"1"');
  expect(saveResponse.headers().etag).toBe('"2"');
  await expect(page.getByText('Changes saved')).toBeVisible();

  await page.reload();
  await expect(page.getByRole('button', { name: 'Note 1, note' })).toBeVisible();
  await expectNoSeriousAccessibilityViolations(page);
});
