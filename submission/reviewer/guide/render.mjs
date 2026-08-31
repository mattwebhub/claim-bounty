import { chromium } from '../../../apps/web/node_modules/@playwright/test/index.mjs';
import { access } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const guideDirectory = path.dirname(fileURLToPath(import.meta.url));
const source = path.join(guideDirectory, 'index.html');
const output = path.resolve(guideDirectory, '..', 'peer2paper-product-guide.pdf');
const executablePath = process.env.PEER2PAPER_PDF_CHROME ?? process.env.CLAIMBOUNTY_PDF_CHROME;

await access(source);
if (executablePath) await access(executablePath);

const browser = await chromium.launch({
  ...(executablePath ? { executablePath } : {}),
  headless: true,
});

try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  await page.goto(pathToFileURL(source).href, { waitUntil: 'networkidle' });
  await page.emulateMedia({ media: 'print' });
  await page.pdf({
    path: output,
    format: 'A4',
    printBackground: true,
    preferCSSPageSize: true,
    tagged: true,
    outline: true,
  });
  console.log(`Rendered ${output}`);
} finally {
  await browser.close();
}
