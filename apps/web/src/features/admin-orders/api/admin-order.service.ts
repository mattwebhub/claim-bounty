import { ApiError, apiRequest, downloadApiFile, resolveApiUrl } from '@/shared/api';
import type { components } from '@/shared/api/generated/schema';
import { orderSchema } from '@/features/claim-intake';
import {
  adminOrderSchema,
  exportSchema,
  orderListSchema,
  type AdminOrder,
  type AdminIntake,
  type ExportRecord,
  type OrderStatus,
} from '../model/admin-order.schema';

export interface AdminOrderFilters {
  cursor?: string;
  limit: number;
  status?: OrderStatus;
}

export async function listAdminOrders(filters: AdminOrderFilters, signal?: AbortSignal) {
  const search = new URLSearchParams({ limit: String(filters.limit) });
  if (filters.status) search.set('status', filters.status);
  if (filters.cursor) search.set('cursor', filters.cursor);
  return apiRequest({
    path: `/admin/orders?${search.toString()}`,
    schema: orderListSchema,
    ...(signal ? { signal } : {}),
  });
}

export async function getAdminOrder(orderId: string, signal?: AbortSignal): Promise<AdminOrder> {
  return apiRequest({
    path: `/admin/orders/${encodeURIComponent(orderId)}`,
    schema: adminOrderSchema,
    ...(signal ? { signal } : {}),
  });
}

export async function getExport(
  orderId: string,
  exportId: string,
  signal?: AbortSignal,
): Promise<ExportRecord> {
  return apiRequest({
    path: `/admin/orders/${encodeURIComponent(orderId)}/exports/${encodeURIComponent(exportId)}`,
    schema: exportSchema,
    ...(signal ? { signal } : {}),
  });
}

export async function createExport(
  orderId: string,
  version: number,
  csrfToken: string,
  input: components['schemas']['CreateExportRequest'],
  idempotencyKey: string,
): Promise<ExportRecord> {
  return apiRequest({
    path: `/admin/orders/${encodeURIComponent(orderId)}/exports`,
    method: 'POST',
    headers: {
      'X-CSRF-Token': csrfToken,
      'If-Match': `"${version}"`,
      'Idempotency-Key': idempotencyKey,
    },
    body: input,
    schema: exportSchema,
  });
}

export async function updateAdminIntake(
  orderId: string,
  version: number,
  csrfToken: string,
  input: AdminIntake,
): Promise<AdminOrder> {
  const command: components['schemas']['UpdateAdminIntakeRequest'] = input;
  return apiRequest({
    path: `/admin/orders/${encodeURIComponent(orderId)}/intake`,
    method: 'PATCH',
    headers: { 'X-CSRF-Token': csrfToken, 'If-Match': `"${version}"` },
    body: command,
    schema: adminOrderSchema,
  });
}

export function adminFileDownloadUrl(orderId: string, fileId: string) {
  return resolveApiUrl(
    `/admin/orders/${encodeURIComponent(orderId)}/files/${encodeURIComponent(fileId)}/content`,
  );
}

export function exportDownloadUrl(orderId: string, exportRecord: ExportRecord) {
  void orderId;
  return resolveApiUrl(exportDownloadPath(exportRecord));
}

function exportDownloadPath(exportRecord: ExportRecord) {
  return (
    exportRecord.contentPath?.replace(/^\/api\/v1(?=\/)/, '') ??
    `/admin/exports/${encodeURIComponent(exportRecord.id)}/download`
  );
}

function contentDigestForSha256(sha256: string) {
  const bytes = sha256.match(/.{2}/g)?.map((value) => Number.parseInt(value, 16));
  if (bytes?.length !== 32) return null;
  const encoded = btoa(String.fromCharCode(...bytes));
  return `sha-256=:${encoded}:`;
}

export interface ExportDownload {
  contentDigest: string;
  delivery: 'streamed';
  filename: string;
  sha256: string;
}

export async function downloadExport(exportRecord: ExportRecord): Promise<ExportDownload> {
  const expectedSha256 = exportRecord.sha256;
  const expectedDigest = expectedSha256 ? contentDigestForSha256(expectedSha256) : null;
  if (!expectedSha256 || !expectedDigest || exportRecord.status !== 'ready') {
    throw new ApiError({
      kind: 'decode',
      message: 'The export integrity metadata is not ready.',
      retryable: false,
    });
  }
  const filename = `claimbounty-${exportRecord.id}.zip`;
  const assertExpectedDigest = (headers: Headers) => {
    const digest = headers.get('content-digest');
    if (digest !== expectedDigest) {
      throw new ApiError({
        kind: 'decode',
        message: 'The downloaded export digest did not match its integrity metadata.',
        retryable: false,
      });
    }
    return digest;
  };
  const response = await downloadApiFile({
    filename,
    path: exportDownloadPath(exportRecord),
    validateHeaders: assertExpectedDigest,
  });
  const contentDigest = assertExpectedDigest(response.headers);
  return {
    contentDigest,
    delivery: response.delivery,
    filename,
    sha256: expectedSha256,
  };
}

export { orderSchema };
