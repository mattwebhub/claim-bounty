import { describe, expect, it } from 'vitest';
import { parseEnvironment } from '@/shared/config/env';

describe('parseEnvironment', () => {
  it('applies safe local defaults', () => {
    expect(parseEnvironment({})).toEqual({
      VITE_APP_NAME: 'React Frontend Template',
      VITE_API_BASE_URL: 'http://localhost:8080/api/v1',
      VITE_API_TIMEOUT_MS: 10_000,
    });
  });

  it('rejects an invalid API base URL with an actionable message', () => {
    expect(() => parseEnvironment({ VITE_API_BASE_URL: 'not-a-url' })).toThrow(/VITE_API_BASE_URL/);
  });
});
