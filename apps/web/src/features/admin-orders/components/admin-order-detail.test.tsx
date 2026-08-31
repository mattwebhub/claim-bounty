import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { renderApplication } from '@/shared/test';
import { AdminOrderDetail } from './admin-order-detail';

const mocks = vi.hoisted(() => ({
  createExport: vi.fn(),
  downloadExport: vi.fn(),
  useAdminOrder: vi.fn<() => unknown>(),
  readyExport: {
    id: '323e4567-e89b-12d3-a456-426614174000',
    orderId: '123e4567-e89b-12d3-a456-426614174000',
    status: 'ready',
    sha256: 'a'.repeat(64),
    contentPath: '/api/v1/admin/exports/323e4567-e89b-12d3-a456-426614174000/download',
    createdAt: '2026-08-30T11:00:00Z',
  },
}));

vi.mock('../api/admin-order.queries', () => ({
  useAdminAccessRevocation: () => false,
  useAdminOrder: mocks.useAdminOrder,
  useCreateExport: () => ({
    isPending: false,
    isError: false,
    error: null,
    mutateAsync: mocks.createExport,
  }),
  useDownloadExport: () => ({
    isPending: false,
    isError: false,
    error: null,
    mutateAsync: mocks.downloadExport,
  }),
  useExportStatus: (_orderId: string, exportId: string | null) => ({
    data: exportId ? mocks.readyExport : undefined,
  }),
}));

vi.mock('./admin-intake-editor', () => ({
  AdminIntakeEditor: () => <article aria-label="Local handoff intake editor" />,
}));

const order = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  publicReference: 'CB-ABCDEF123456',
  status: 'ready_for_export',
  version: 7,
  title: 'Retention intervention',
  purpose: 'Check the primary estimate.',
  targetClaim: { text: 'Retention increased.', sourceLocation: 'Table 2' },
  submitterEmail: 'researcher@example.org',
  permissions: { executeSuppliedCode: false, externalSearch: true },
  privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
  files: [
    {
      id: '223e4567-e89b-12d3-a456-426614174000',
      role: 'primary_paper',
      originalDisplayName: 'paper.pdf',
      status: 'clean',
    },
  ],
  frozenIntake: {},
  readinessIssues: [],
  events: [],
  exports: [],
  createdAt: '2026-08-30T10:00:00Z',
} as const;

describe('AdminOrderDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.useAdminOrder.mockReturnValue({
      isPending: false,
      isError: false,
      error: null,
      data: order,
      refetch: vi.fn(),
    });
    mocks.createExport.mockResolvedValue(mocks.readyExport);
    mocks.downloadExport.mockResolvedValue({
      contentDigest: 'sha-256=:qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqo=:',
      delivery: 'streamed',
      filename: `claimbounty-${mocks.readyExport.id}.zip`,
      sha256: 'a'.repeat(64),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders a loading state while private order data is requested', () => {
    mocks.useAdminOrder.mockReturnValue({
      isPending: true,
      isError: false,
      error: null,
      data: undefined,
      refetch: vi.fn(),
    });
    renderApplication(
      <AdminOrderDetail csrfToken={'c'.repeat(32)} onAccessDenied={vi.fn()} orderId={order.id} />,
    );

    expect(screen.getByText('Loading')).toBeInTheDocument();
  });

  it('renders clean-file download and creates an API-only export handoff', async () => {
    const user = userEvent.setup();
    renderApplication(
      <AdminOrderDetail csrfToken={'c'.repeat(32)} onAccessDenied={vi.fn()} orderId={order.id} />,
    );

    expect(screen.getByRole('heading', { name: 'Retention intervention' })).toBeVisible();
    expect(screen.getByRole('link', { name: /Download/ })).toHaveAttribute(
      'href',
      `http://127.0.0.1:8080/api/v1/admin/orders/${order.id}/files/223e4567-e89b-12d3-a456-426614174000/content`,
    );
    await user.click(screen.getByRole('checkbox', { name: /I reviewed the claim/ }));
    await user.click(screen.getByRole('checkbox', { name: /I confirmed every included file/ }));
    await user.click(screen.getByRole('button', { name: 'Create export' }));

    expect(mocks.createExport).toHaveBeenCalledWith({
      metadataReviewed: true,
      filesReviewed: true,
      retentionPolicyVersion: 'claimbounty-p0.1',
      preserveRunOutputs: true,
    });
    const download = await screen.findByRole('link', { name: /Download export/ });
    expect(download).toHaveAttribute(
      'href',
      'http://127.0.0.1:8080/api/v1/admin/exports/323e4567-e89b-12d3-a456-426614174000/download',
    );
    expect(screen.getByText('a'.repeat(64))).toBeVisible();
    expect(
      screen.getByText(
        'This browser will handle the export as a native download without buffering it in this page. Verify the saved archive with the displayed SHA-256 before opening it.',
      ),
    ).toBeVisible();
    expect(
      screen.getByText('sha-256=:qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqo=:'),
    ).toBeVisible();
  });

  it('uses the native same-origin link without starting a JavaScript download when streaming is unsupported', async () => {
    const user = userEvent.setup();
    renderApplication(
      <AdminOrderDetail csrfToken={'c'.repeat(32)} onAccessDenied={vi.fn()} orderId={order.id} />,
    );

    await user.click(screen.getByRole('checkbox', { name: /I reviewed the claim/ }));
    await user.click(screen.getByRole('checkbox', { name: /I confirmed every included file/ }));
    await user.click(screen.getByRole('button', { name: 'Create export' }));
    const download = await screen.findByRole('link', { name: 'Download export' });
    let defaultPreventedByHandler = true;
    window.addEventListener(
      'click',
      (event) => {
        defaultPreventedByHandler = event.defaultPrevented;
        event.preventDefault();
      },
      { once: true },
    );
    await user.click(download);

    expect(defaultPreventedByHandler).toBe(false);
    expect(download).toHaveAttribute('download');
    expect(download).toHaveAttribute(
      'href',
      'http://127.0.0.1:8080/api/v1/admin/exports/323e4567-e89b-12d3-a456-426614174000/download',
    );
    expect(mocks.downloadExport).not.toHaveBeenCalled();
  });

  it('intercepts the link and records the matched digest when direct-to-disk streaming is supported', async () => {
    const user = userEvent.setup();
    vi.stubGlobal('showSaveFilePicker', vi.fn());
    renderApplication(
      <AdminOrderDetail csrfToken={'c'.repeat(32)} onAccessDenied={vi.fn()} orderId={order.id} />,
    );

    await user.click(screen.getByRole('checkbox', { name: /I reviewed the claim/ }));
    await user.click(screen.getByRole('checkbox', { name: /I confirmed every included file/ }));
    await user.click(screen.getByRole('button', { name: 'Create export' }));
    await user.click(await screen.findByRole('link', { name: 'Download export' }));

    expect(mocks.downloadExport).toHaveBeenCalledWith(mocks.readyExport);
    expect(await screen.findByText('Streamed directly to the selected file.')).toBeVisible();
    expect(screen.getByText('Matched response Content-Digest header')).toBeVisible();
    expect(screen.getAllByText(/sha-256=:qqqq/)).toHaveLength(2);
  });
});
