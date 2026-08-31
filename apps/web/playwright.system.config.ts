import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig, devices } from '@playwright/test';

const webDirectory = path.dirname(fileURLToPath(import.meta.url));
const apiDirectory = path.resolve(webDirectory, '../api');
const externalWebOrigin = process.env.SYSTEM_TEST_WEB_ORIGIN;
const webOrigin = externalWebOrigin ?? 'http://127.0.0.1:4173';
const apiOrigin = 'http://127.0.0.1:18080';

export default defineConfig({
  testDir: './tests/system',
  outputDir: './test-results/system',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI
    ? [['line'], ['html', { open: 'never', outputFolder: 'playwright-report/system' }]]
    : [['list'], ['html', { open: 'never', outputFolder: 'playwright-report/system' }]],
  use: {
    baseURL: webOrigin,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'system-chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  ...(externalWebOrigin
    ? {}
    : {
        webServer: [
          {
            name: 'api',
            command:
              'system_api_binary=$(mktemp /tmp/micro1-system-api.XXXXXX); trap \'rm -f "$system_api_binary"\' EXIT; go build -trimpath -o "$system_api_binary" ./cmd/api; "$system_api_binary"',
            cwd: apiDirectory,
            url: `${apiOrigin}/health/ready`,
            reuseExistingServer: false,
            timeout: 120_000,
            gracefulShutdown: { signal: 'SIGTERM', timeout: 15_000 },
            stdout: 'pipe',
            env: {
              APP_ENV: 'test',
              SERVER_HOST: '127.0.0.1',
              SERVER_PORT: '18080',
              CORS_ALLOWED_ORIGINS: webOrigin,
              DATABASE_URL:
                process.env.SYSTEM_TEST_DATABASE_URL ??
                'postgres://postgres:postgres@127.0.0.1:5432/app?sslmode=disable',
              DATABASE_AUTO_MIGRATE: 'true',
              LOG_FORMAT: 'text',
              LOG_LEVEL: 'info',
              GOCACHE:
                process.env.SYSTEM_TEST_GOCACHE ?? path.join(os.tmpdir(), 'micro1-system-go-cache'),
            },
          },
          {
            name: 'web',
            command: 'pnpm preview --host 127.0.0.1',
            cwd: webDirectory,
            url: webOrigin,
            reuseExistingServer: false,
            timeout: 120_000,
            gracefulShutdown: { signal: 'SIGTERM', timeout: 5_000 },
            env: {
              SYSTEM_TEST_API_ORIGIN: apiOrigin,
            },
          },
        ],
      }),
});
