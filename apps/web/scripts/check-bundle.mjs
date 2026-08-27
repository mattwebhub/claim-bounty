import { brotliCompressSync } from 'node:zlib';
import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';

const kibibyte = 1024;
const limits = {
  totalJavaScript: Number(process.env.BUNDLE_MAX_JS_KIB ?? 250) * kibibyte,
  largestJavaScript: Number(process.env.BUNDLE_MAX_CHUNK_KIB ?? 175) * kibibyte,
  totalCss: Number(process.env.BUNDLE_MAX_CSS_KIB ?? 50) * kibibyte,
};

async function listFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nested = await Promise.all(
    entries.map((entry) => {
      const resolved = path.join(directory, entry.name);
      return entry.isDirectory() ? listFiles(resolved) : [resolved];
    }),
  );
  return nested.flat();
}

function formatSize(bytes) {
  return `${(bytes / kibibyte).toFixed(1)} KiB`;
}

async function measure(file) {
  const contents = await readFile(file);
  return {
    file: path.relative(process.cwd(), file),
    raw: contents.byteLength,
    brotli: brotliCompressSync(contents).byteLength,
  };
}

let files;
try {
  files = await listFiles(path.resolve('dist/assets'));
} catch (error) {
  console.error('BUNDLE-001: dist/assets is missing. Run `pnpm build` before `pnpm bundle:check`.');
  throw error;
}

const measured = await Promise.all(
  files.filter((file) => file.endsWith('.js') || file.endsWith('.css')).map(measure),
);
const javascript = measured.filter(({ file }) => file.endsWith('.js'));
const styles = measured.filter(({ file }) => file.endsWith('.css'));

for (const asset of measured.sort((left, right) => right.brotli - left.brotli)) {
  console.log(`${asset.file}: ${formatSize(asset.brotli)} Brotli (${formatSize(asset.raw)} raw)`);
}

const totalJavaScript = javascript.reduce((total, asset) => total + asset.brotli, 0);
const totalCss = styles.reduce((total, asset) => total + asset.brotli, 0);
const largestJavaScript = Math.max(0, ...javascript.map(({ brotli }) => brotli));
const failures = [
  ['total JavaScript', totalJavaScript, limits.totalJavaScript],
  ['largest JavaScript chunk', largestJavaScript, limits.largestJavaScript],
  ['total CSS', totalCss, limits.totalCss],
].filter(([, actual, limit]) => actual > limit);

if (failures.length > 0) {
  for (const [label, actual, limit] of failures) {
    console.error(
      `BUNDLE-002: ${label} is ${formatSize(actual)}, above the ${formatSize(limit)} budget. Lazy-load heavy routes or record an intentional budget change.`,
    );
  }
  process.exitCode = 1;
} else {
  console.log(
    `Bundle budgets passed: JS ${formatSize(totalJavaScript)}, largest chunk ${formatSize(largestJavaScript)}, CSS ${formatSize(totalCss)}.`,
  );
}
