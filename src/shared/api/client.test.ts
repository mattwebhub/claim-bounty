import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { z } from 'zod';
import { ApiError, apiRequest } from '@/shared/api';
import { mockServer } from '@/shared/test/mocks/server';

describe('apiRequest', () => {
  it('unwraps and validates a successful response', async () => {
    mockServer.use(
      http.get('http://localhost:8080/api/v1/example', () =>
        HttpResponse.json({ data: { id: 'example-1' } }),
      ),
    );

    await expect(
      apiRequest({ path: '/example', schema: z.object({ id: z.string() }) }),
    ).resolves.toEqual({ id: 'example-1' });
  });

  it('normalizes a structured API failure', async () => {
    mockServer.use(
      http.get('http://localhost:8080/api/v1/example', () =>
        HttpResponse.json(
          {
            error: {
              code: 'example_missing',
              message: 'Example not found.',
              requestId: 'req-1',
              details: [{ path: 'name', code: 'required', message: 'Name is required.' }],
            },
          },
          { status: 404 },
        ),
      ),
    );

    const error = await apiRequest({ path: '/example', schema: z.unknown() }).catch(
      (cause: unknown) => cause,
    );

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      kind: 'http',
      status: 404,
      code: 'example_missing',
      requestId: 'req-1',
      retryable: false,
      fieldIssues: [{ path: 'name', code: 'required', message: 'Name is required.' }],
    });
  });

  it('rejects a success payload that violates the runtime schema', async () => {
    mockServer.use(
      http.get('http://localhost:8080/api/v1/example', () =>
        HttpResponse.json({ data: { id: 42 } }),
      ),
    );

    await expect(
      apiRequest({ path: '/example', schema: z.object({ id: z.string() }) }),
    ).rejects.toMatchObject({ kind: 'decode', retryable: false });
  });
});
