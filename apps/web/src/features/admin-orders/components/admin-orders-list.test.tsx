import { http, HttpResponse } from 'msw';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { renderApplication } from '@/shared/test';
import { mockServer } from '@/shared/test/mocks/server';
import { AdminOrdersList } from './admin-orders-list';

const order = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  publicReference: 'CB-ABCDEF123456',
  status: 'scanning',
  version: 3,
  title: 'Retention intervention',
  purpose: 'Check the primary estimate',
  targetClaim: { text: 'Retention increased', sourceLocation: 'Table 2' },
  permissions: { executeSuppliedCode: false, externalSearch: true },
  privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
  files: [],
  piiRetention: {
    policyVersion: 'policy.1',
    disposition: 'hard_delete',
    sourceDeleteAfter: '2026-09-15T10:00:00Z',
    piiDeleteAfter: '2026-09-30T10:00:00Z',
  },
  createdAt: '2026-08-30T10:00:00Z',
  updatedAt: '2026-08-30T11:00:00Z',
  submittedAt: '2026-08-30T10:30:00Z',
} as const;

describe('AdminOrdersList', () => {
  it('renders contract data and emits URL-owned status filters', async () => {
    const user = userEvent.setup();
    const onFiltersChange = vi.fn();
    mockServer.use(
      http.get('*/api/v1/admin/orders', () =>
        HttpResponse.json({ data: { items: [order], nextCursor: 'next-page' } }),
      ),
    );
    renderApplication(
      <AdminOrdersList onAccessDenied={vi.fn()} onFiltersChange={onFiltersChange} />,
    );

    expect(screen.getByText('Loading')).toBeInTheDocument();
    expect(await screen.findByText('CB-ABCDEF123456')).toBeVisible();
    expect(screen.getByText('Retention intervention')).toBeVisible();
    await user.selectOptions(screen.getByLabelText('Status'), 'ready_for_export');
    expect(onFiltersChange).toHaveBeenCalledWith({ status: 'ready_for_export' });
  });

  it('renders the empty state', async () => {
    mockServer.use(
      http.get('*/api/v1/admin/orders', () => HttpResponse.json({ data: { items: [] } })),
    );
    renderApplication(<AdminOrdersList onAccessDenied={vi.fn()} onFiltersChange={vi.fn()} />);

    expect(await screen.findByRole('heading', { name: 'No orders match this view' })).toBeVisible();
  });

  it('shows a retry and request reference for ordinary API errors', async () => {
    mockServer.use(
      http.get('*/api/v1/admin/orders', () =>
        HttpResponse.json(
          { error: { message: 'Orders are temporarily unavailable.', requestId: 'req-list-1' } },
          { status: 500 },
        ),
      ),
    );
    renderApplication(<AdminOrdersList onAccessDenied={vi.fn()} onFiltersChange={vi.fn()} />);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Orders are temporarily unavailable. Request ID: req-list-1.',
    );
    expect(screen.getByRole('button', { name: 'Retry' })).toBeEnabled();
  });

  it('clears cached order data immediately when allowlist access is revoked', async () => {
    const onAccessDenied = vi.fn();
    mockServer.use(
      http.get('*/api/v1/admin/orders', () =>
        HttpResponse.json({ error: { message: 'Admin access denied.' } }, { status: 403 }),
      ),
    );
    const { queryClient } = renderApplication(
      <AdminOrdersList onAccessDenied={onAccessDenied} onFiltersChange={vi.fn()} />,
    );

    expect(await screen.findByText('Admin access was removed')).toBeVisible();
    await expect.poll(() => onAccessDenied).toHaveBeenCalledOnce();
    expect(queryClient.getQueriesData({ queryKey: ['admin-orders'] })).toEqual([]);
  });
});
