import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page } from '@playwright/test';

const project = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Hackathon workspace',
  createdAt: '2026-08-27T09:00:00Z',
  updatedAt: '2026-08-27T09:00:00Z',
};

const emptyWorkspace = {
  projectId: project.id,
  version: 1,
  createdAt: project.createdAt,
  updatedAt: project.updatedAt,
  document: { schemaVersion: 1, objects: [] },
};

async function mockProjectApi(page: Page) {
  await page.route('**/api/v1/projects**', async (route) => {
    const request = route.request();

    if (request.method() === 'POST') {
      await route.fulfill({ status: 201, json: { data: project } });
      return;
    }

    if (new URL(request.url()).pathname.endsWith(`/${project.id}`)) {
      await route.fulfill({ json: { data: project } });
      return;
    }

    await route.fulfill({ json: { data: { items: [project] } } });
  });
}

test('application shell is keyboard reachable and has no serious accessibility violations', async ({
  page,
}) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('link', { name: /skip to main content/i })).toBeFocused();

  const results = await new AxeBuilder({ page }).analyze();
  expect(
    results.violations.filter(({ impact }) => impact === 'critical' || impact === 'serious'),
  ).toEqual([]);
});

test('project list and create flow work against the public API contract', async ({ page }) => {
  await mockProjectApi(page);
  await page.goto('/projects');

  await expect(page.getByRole('heading', { level: 1, name: 'Projects' })).toBeVisible();
  await expect(page.getByRole('heading', { name: project.name })).toBeVisible();

  await page.getByLabel('Project name').fill(project.name);
  await page.getByRole('button', { name: 'Create project' }).click();
  await expect(page.getByRole('status')).toContainText(`${project.name} was created.`);
});

test('workspace edits save with optimistic concurrency and survive reload', async ({ page }) => {
  let workspace = emptyWorkspace;
  let receivedIfMatch: string | undefined;

  await page.route(`**/api/v1/projects/${project.id}/workspace`, async (route) => {
    const request = route.request();
    if (request.method() === 'PUT') {
      receivedIfMatch = request.headers()['if-match'];
      const body = request.postDataJSON() as { document: typeof emptyWorkspace.document };
      workspace = {
        ...workspace,
        version: workspace.version + 1,
        updatedAt: '2026-08-27T10:00:00Z',
        document: body.document,
      };
    }
    await route.fulfill({
      json: { data: workspace },
      headers: { ETag: `"${workspace.version}"` },
    });
  });

  await page.goto(`/workspace/${project.id}`);
  await page.getByRole('button', { name: /Add Note/ }).click();
  await page.keyboard.press('ArrowRight');
  await page.getByRole('button', { name: 'Save' }).click();

  await expect.poll(() => receivedIfMatch).toBe('"1"');
  await expect(page.getByText('Changes saved')).toBeVisible();
  await page.reload();
  await expect(page.getByRole('button', { name: 'Note 1, note' })).toBeVisible();

  const results = await new AxeBuilder({ page }).analyze();
  expect(
    results.violations.filter(({ impact }) => impact === 'critical' || impact === 'serious'),
  ).toEqual([]);
});
