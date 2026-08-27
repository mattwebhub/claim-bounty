import '@testing-library/jest-dom/vitest';
// Intentional defense against leaked portals and concurrent trees between feature tests.
// eslint-disable-next-line testing-library/no-manual-cleanup
import { cleanup, configure } from '@testing-library/react';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { mockServer } from '@/shared/test/mocks/server';

// Coverage instrumentation and contended CI runners can make otherwise immediate MSW
// transitions exceed Testing Library's one-second default.
configure({ asyncUtilTimeout: 3_000 });

beforeAll(() => {
  mockServer.listen({ onUnhandledRequest: 'error' });
});
afterEach(() => {
  cleanup();
  mockServer.resetHandlers();
  window.localStorage.clear();
});
afterAll(() => {
  mockServer.close();
});
