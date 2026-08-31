import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createOrder, sha256, submitOrder, uploadOrderFile } from './order.service';
import { orderKeys, useCreateOrder, useSubmitOrder, useUploadOrderFile } from './order.queries';
import type { Order, OrderFile, OrderIntakeInput } from '../model/order.schema';

vi.mock('./order.service', () => ({
  createOrder: vi.fn(),
  removeOrderFile: vi.fn(),
  sha256: vi.fn(),
  submitOrder: vi.fn(),
  uploadOrderFile: vi.fn(),
}));

const storedFile: OrderFile = {
  id: '223e4567-e89b-12d3-a456-426614174000',
  role: 'primary_paper',
  originalDisplayName: 'paper.pdf',
  sizeBytes: 5,
  sha256: 'a'.repeat(64),
  storage: { objectVersion: 'v1', sha256: 'a'.repeat(64), immutability: 'write_once' },
  declaredMediaType: 'application/pdf',
  detectedMediaType: 'application/pdf',
  status: 'uploaded',
  createdAt: '2026-08-30T10:00:00Z',
  updatedAt: '2026-08-30T10:00:00Z',
};

const order: Order = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  publicReference: 'CB-ABCDEF123456',
  status: 'uploading',
  version: 1,
  title: 'Study',
  purpose: 'Review the primary claim.',
  targetClaim: { text: 'The intervention changed the outcome.', sourceLocation: 'Table 2' },
  permissions: { executeSuppliedCode: false, externalSearch: false },
  privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
  files: [storedFile],
  piiRetention: {
    policyVersion: 'policy.1',
    disposition: 'hard_delete',
    sourceDeleteAfter: '2026-09-15T10:00:00Z',
    piiDeleteAfter: '2026-09-30T10:00:00Z',
  },
  createdAt: '2026-08-30T10:00:00Z',
  updatedAt: '2026-08-30T10:00:00Z',
};

const intake: OrderIntakeInput = {
  title: 'Study',
  purpose: 'Review the primary claim.',
  targetClaim: { text: 'The intervention changed the outcome.', sourceLocation: 'Table 2' },
  permissions: { executeSuppliedCode: false, externalSearch: false },
  privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
};

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('idempotent order mutations', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reuses the exact order-creation key after a lost response until success reconciles it', async () => {
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    vi.mocked(createOrder)
      .mockRejectedValueOnce(new TypeError('response lost'))
      .mockResolvedValue(order);
    const { result } = renderHook(() => useCreateOrder(), {
      wrapper: createWrapper(queryClient),
    });
    const variables = { input: intake, csrfToken: 'c'.repeat(32) };

    await act(async () => {
      await result.current.mutateAsync(variables).catch(() => undefined);
    });
    await act(async () => {
      await result.current.mutateAsync(variables);
    });

    const keys = vi.mocked(createOrder).mock.calls.map((call) => call[2]);
    expect(keys).toHaveLength(2);
    expect(keys[0]).toMatch(/^[0-9a-f-]{36}$/);
    expect(keys[1]).toBe(keys[0]);
  });

  it('reuses the exact submission key after a lost response until success reconciles it', async () => {
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    queryClient.setQueryData(orderKeys.detail(order.id), order);
    vi.mocked(submitOrder)
      .mockRejectedValueOnce(new TypeError('response lost'))
      .mockResolvedValue({ ...order, status: 'submitted' });
    const { result } = renderHook(() => useSubmitOrder(), {
      wrapper: createWrapper(queryClient),
    });
    const variables = {
      orderId: order.id,
      csrfToken: 'c'.repeat(32),
      termsAccepted: true,
      uploadsAuthorized: true,
      analysisUseAuthorized: true,
    };

    await act(async () => {
      await result.current.mutateAsync(variables).catch(() => undefined);
    });
    await act(async () => {
      await result.current.mutateAsync(variables);
    });

    const keys = vi.mocked(submitOrder).mock.calls.map((call) => call[3]);
    expect(keys).toHaveLength(2);
    expect(keys[0]).toMatch(/^[0-9a-f-]{36}$/);
    expect(keys[1]).toBe(keys[0]);
  });
});

describe('useUploadOrderFile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reuses the idempotency key on retry and does not duplicate a returned file', async () => {
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    queryClient.setQueryData(orderKeys.detail(order.id), order);
    vi.mocked(sha256).mockResolvedValue('a'.repeat(64));
    vi.mocked(uploadOrderFile).mockResolvedValue({
      file: { ...storedFile, status: 'clean', updatedAt: '2026-08-30T10:01:00Z' },
      version: 2,
    });
    const { result } = renderHook(() => useUploadOrderFile(), {
      wrapper: createWrapper(queryClient),
    });
    const input = {
      csrfToken: 'c'.repeat(32),
      file: new File(['paper'], 'paper.pdf', { type: 'application/pdf' }),
      idempotencyKey: 'stable-upload-key',
      onHashing: vi.fn(),
      onProgress: vi.fn(),
      orderId: order.id,
      role: 'primary_paper' as const,
      signal: new AbortController().signal,
    };

    await act(async () => {
      await result.current.mutateAsync(input);
      await result.current.mutateAsync(input);
    });

    expect(vi.mocked(uploadOrderFile)).toHaveBeenCalledTimes(2);
    expect(vi.mocked(uploadOrderFile).mock.calls.map(([call]) => call.idempotencyKey)).toEqual([
      'stable-upload-key',
      'stable-upload-key',
    ]);
    const updated = queryClient.getQueryData<Order>(orderKeys.detail(order.id));
    expect(updated?.files).toHaveLength(1);
    expect(updated?.files[0]?.status).toBe('clean');
  });
});
