import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '@/shared/api';
import { mockServer } from '@/shared/test';
import { createExport, downloadExport } from './admin-order.service';
import type { ExportRecord } from '../model/admin-order.schema';

const readyExport: ExportRecord = {
  id: '323e4567-e89b-12d3-a456-426614174000',
  orderId: '123e4567-e89b-12d3-a456-426614174000',
  status: 'ready',
  routineContract: {
    routineId: 'claim-bounty-operations/run-claimbounty-scientific-audit',
    revision: `sha256:${'b'.repeat(64)}`,
    validation: {
      status: 'validated',
      validatedAt: '2026-08-30T10:00:00Z',
      evidenceSha256: 'c'.repeat(64),
    },
  },
  inputs: [
    {
      fileId: '223e4567-e89b-12d3-a456-426614174000',
      objectVersion: 'generation-1',
      sha256: 'd'.repeat(64),
    },
  ],
  sha256: 'a'.repeat(64),
  sizeBytes: 7,
  contentPath: '/api/v1/admin/exports/323e4567-e89b-12d3-a456-426614174000/download',
  createdAt: '2026-08-30T10:05:00Z',
  completedAt: '2026-08-30T10:06:00Z',
};

describe('createExport', () => {
  it('sends the cached order version and preserves conflict request metadata', async () => {
    let ifMatch: string | null = null;
    let idempotencyKey: string | null = null;
    mockServer.use(
      http.post('*/api/v1/admin/orders/:orderId/exports', ({ request }) => {
        ifMatch = request.headers.get('if-match');
        idempotencyKey = request.headers.get('idempotency-key');
        return HttpResponse.json(
          {
            error: {
              code: 'version_conflict',
              message: 'The order changed in another session.',
              requestId: 'req-conflict-1',
            },
          },
          { status: 409 },
        );
      }),
    );

    const error = await createExport(
      '123e4567-e89b-12d3-a456-426614174000',
      7,
      'c'.repeat(32),
      {
        retentionPolicyVersion: 'policy.1',
        preserveRunOutputs: true,
      },
      'stable-export-key',
    ).catch((cause: unknown) => cause);

    expect(ifMatch).toBe('"7"');
    expect(idempotencyKey).toBe('stable-export-key');
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 409,
      code: 'version_conflict',
      requestId: 'req-conflict-1',
    });
  });
});

describe('downloadExport', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('preserves the response Content-Digest after matching it to export metadata', async () => {
    const digest = 'sha-256=:qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqo=:';
    vi.stubGlobal('showSaveFilePicker', createSaveFilePicker());
    mockServer.use(
      http.get(
        '*/api/v1/admin/exports/:exportId/download',
        () =>
          new HttpResponse(new Uint8Array([80, 75, 3, 4]), {
            headers: { 'Content-Digest': digest, 'Content-Type': 'application/zip' },
          }),
      ),
    );

    await expect(downloadExport(readyExport)).resolves.toMatchObject({
      contentDigest: digest,
      delivery: 'streamed',
      filename: `claimbounty-${readyExport.id}.zip`,
      sha256: readyExport.sha256,
    });
  });

  it('blocks a download whose response digest differs from the frozen export metadata', async () => {
    vi.stubGlobal('showSaveFilePicker', createSaveFilePicker());
    mockServer.use(
      http.get(
        '*/api/v1/admin/exports/:exportId/download',
        () =>
          new HttpResponse(new Uint8Array([80, 75, 3, 4]), {
            headers: {
              'Content-Digest': 'sha-256=:u7u7u7u7u7u7u7u7u7u7u7u7u7u7u7u7u7u7u7u7s=:',
            },
          }),
      ),
    );

    await expect(downloadExport(readyExport)).rejects.toMatchObject({
      kind: 'decode',
      retryable: false,
    });
  });
});

function createSaveFilePicker() {
  return vi.fn().mockResolvedValue({
    createWritable: vi.fn().mockResolvedValue(new WritableStream<Uint8Array>()),
  });
}
