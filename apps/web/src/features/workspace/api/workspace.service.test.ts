import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { mockServer } from '@/shared/test/mocks/server';
import { saveWorkspace } from './workspace.service';

const projectId = '00000000-0000-4000-8000-000000000001';

describe('workspace service', () => {
  it('sends the expected version only as a strong If-Match header', async () => {
    let requestBody: unknown;
    let ifMatch: string | null = null;
    mockServer.use(
      http.put(
        `http://127.0.0.1:8080/api/v1/projects/${projectId}/workspace`,
        async ({ request }) => {
          requestBody = await request.json();
          ifMatch = request.headers.get('if-match');
          return HttpResponse.json({
            data: {
              projectId,
              version: 2,
              createdAt: '2026-08-27T10:00:00.000Z',
              updatedAt: '2026-08-27T10:01:00.000Z',
              document: { schemaVersion: 1, objects: [] },
            },
          });
        },
      ),
    );

    const result = await saveWorkspace(projectId, {
      expectedVersion: 1,
      document: { schemaVersion: 1, objects: [] },
    });

    expect(ifMatch).toBe('"1"');
    expect(requestBody).toEqual({ document: { schemaVersion: 1, objects: [] } });
    expect(result.version).toBe(2);
  });
});
