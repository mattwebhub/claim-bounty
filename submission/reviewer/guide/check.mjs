import AxeBuilder from '../../../apps/web/node_modules/@axe-core/playwright/dist/index.mjs';
import { chromium } from '../../../apps/web/node_modules/@playwright/test/index.mjs';
import { access } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const guideDirectory = path.dirname(fileURLToPath(import.meta.url));
const source = path.join(guideDirectory, 'index.html');
const executablePath = process.env.PEER2PAPER_PDF_CHROME ?? process.env.CLAIMBOUNTY_PDF_CHROME;

await access(source);
if (executablePath) await access(executablePath);

const browser = await chromium.launch({
  ...(executablePath ? { executablePath } : {}),
  args: ['--allow-file-access-from-files'],
  headless: true,
});

try {
  const context = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  const page = await context.newPage();
  const consoleErrors = [];
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text());
  });
  await page.goto(pathToFileURL(source).href, { waitUntil: 'networkidle' });

  const sheetMetrics = await page.locator('.sheet').evaluateAll((sheets) =>
    sheets.map((sheet, index) => ({
      page: index + 1,
      clientHeight: sheet.clientHeight,
      scrollHeight: sheet.scrollHeight,
      clientWidth: sheet.clientWidth,
      scrollWidth: sheet.scrollWidth,
    })),
  );
  const overflow = sheetMetrics.filter(
    ({ clientHeight, scrollHeight, clientWidth, scrollWidth }) =>
      scrollHeight > clientHeight || scrollWidth > clientWidth,
  );
  const links = await page
    .locator('a[href]')
    .evaluateAll((anchors) => anchors.map((anchor) => anchor.href));
  const forbiddenLinks = links.filter(
    (link) =>
      link.startsWith('file:') ||
      /(?:\/Users\/|\/home\/|\/var\/|\/tmp\/|\/private\/|\/Applications\/|[A-Za-z]:\\)/.test(link),
  );
  const accessibility = await new AxeBuilder({ page }).analyze();
  const summary = {
    sheets: sheetMetrics.length,
    overflow,
    links,
    forbiddenLinks,
    consoleErrors,
    accessibilityViolations: accessibility.violations.map(({ id, impact, nodes }) => ({
      id,
      impact,
      nodes: nodes.map(({ failureSummary, target }) => ({ failureSummary, target })),
    })),
  };

  console.log(JSON.stringify(summary, null, 2));
  if (
    sheetMetrics.length !== 13 ||
    overflow.length ||
    forbiddenLinks.length ||
    consoleErrors.length ||
    accessibility.violations.length
  ) {
    process.exitCode = 1;
  }
} finally {
  await browser.close();
}
