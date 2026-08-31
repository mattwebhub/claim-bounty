import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createExport } from './admin-order.service';
import { adminOrderKeys, useCreateExport } from './admin-order.queries';
import type { AdminOrder, ExportReadinessInput, ExportRecord } from '../model/admin-order.schema';

vi.mock('./admin-order.service', () => ({
  createExport: vi.fn(),
  downloadExport: vi.fn(),
  getAdminOrder: vi.fn(),
  getExport: vi.fn(),
  listAdminOrders: vi.fn(),
  updateAdminIntake: vi.fn(),
}));

const orderId = '123e4567-e89b-12d3-a456-426614174000';
const fileId = '223e4567-e89b-12d3-a456-426614174000';
const exportId = '323e4567-e89b-12d3-a456-426614174000';
const routineContract = {
  routineId: 'claim-bounty-operations/run-claimbounty-scientific-audit' as const,
  revision: `sha256:${'a'.repeat(64)}`,
  validation: {
    status: 'validated' as const,
    validatedAt: '2026-08-30T10:00:00Z',
    evidenceSha256: 'b'.repeat(64),
  },
};
const exportRecord: ExportRecord = {
  id: exportId,
  orderId,
  status: 'queued',
  routineContract,
  inputs: [{ fileId, objectVersion: 'generation-1', sha256: 'a'.repeat(64) }],
  createdAt: '2026-08-30T10:05:00Z',
};
const order: AdminOrder = {
  id: orderId,
  publicReference: 'CB-ABCDEF123456',
  status: 'ready_for_export',
  version: 7,
  title: 'Study',
  purpose: 'Review the primary claim.',
  targetClaim: { text: 'The intervention changed the outcome.', sourceLocation: 'Table 2' },
  files: [
    {
      id: fileId,
      role: 'primary_paper',
      originalDisplayName: 'paper.pdf',
      sizeBytes: 5,
      sha256: 'a'.repeat(64),
      storage: {
        objectVersion: 'generation-1',
        sha256: 'a'.repeat(64),
        immutability: 'write_once',
      },
      declaredMediaType: 'application/pdf',
      detectedMediaType: 'application/pdf',
      status: 'clean',
      createdAt: '2026-08-30T10:00:00Z',
      updatedAt: '2026-08-30T10:01:00Z',
    },
  ],
  piiRetention: {
    policyVersion: 'policy.1',
    disposition: 'hard_delete',
    sourceDeleteAfter: '2026-09-15T10:00:00Z',
    piiDeleteAfter: '2026-09-30T10:00:00Z',
  },
  createdAt: '2026-08-30T10:00:00Z',
  updatedAt: '2026-08-30T10:01:00Z',
  submitterEmail: 'researcher@example.test',
  permissions: { executeSuppliedCode: false, externalSearch: false },
  privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
  frozenIntake: null,
  readinessIssues: [],
  events: [],
  exports: [],
};
const input: ExportReadinessInput = {
  metadataReviewed: true,
  filesReviewed: true,
  retentionPolicyVersion: 'claimbounty-p0.1',
  preserveRunOutputs: true,
};

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('useCreateExport', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reuses the exact export key after a lost response until success reconciles it', async () => {
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    queryClient.setQueryData(adminOrderKeys.detail(orderId), order);
    vi.mocked(createExport)
      .mockRejectedValueOnce(new TypeError('response lost'))
      .mockResolvedValue(exportRecord);
    const { result } = renderHook(() => useCreateExport(orderId, 'c'.repeat(32)), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync(input).catch(() => undefined);
    });
    await act(async () => {
      await result.current.mutateAsync(input);
    });

    const keys = vi.mocked(createExport).mock.calls.map((call) => call[4]);
    expect(keys).toHaveLength(2);
    expect(keys[0]).toMatch(/^[0-9a-f-]{36}$/);
    expect(keys[1]).toBe(keys[0]);
  });
});
