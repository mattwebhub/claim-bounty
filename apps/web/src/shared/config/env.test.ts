import { describe, expect, it } from 'vitest';
import { parseEnvironment } from '@/shared/config/env';

describe('parseEnvironment', () => {
  it('applies safe local defaults', () => {
    expect(parseEnvironment({})).toEqual({
      VITE_APP_NAME: 'Micro1 Template',
      VITE_API_BASE_URL: '/api/v1',
      VITE_API_TIMEOUT_MS: 10_000,
    });
  });

  it('accepts explicit same-origin and HTTP(S) API locations', () => {
    expect(parseEnvironment({ VITE_API_BASE_URL: '/gateway/api/v1' }).VITE_API_BASE_URL).toBe(
      '/gateway/api/v1',
    );
    expect(
      parseEnvironment({ VITE_API_BASE_URL: 'https://api.example.test/api/v1' }).VITE_API_BASE_URL,
    ).toBe('https://api.example.test/api/v1');
  });

  it.each(['not-a-url', '//attacker.example/api', 'ftp://api.example.test', '/api/v1?token=x'])(
    'rejects unsafe API base URL %s with an actionable message',
    (value) => {
      expect(() => parseEnvironment({ VITE_API_BASE_URL: value })).toThrow(/VITE_API_BASE_URL/);
    },
  );
});
