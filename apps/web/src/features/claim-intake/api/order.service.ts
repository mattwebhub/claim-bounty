import { z } from 'zod';
import { apiRequest, apiRequestWithMetadata, uploadApiMultipart } from '@/shared/api';
import type { components } from '@/shared/api/generated/schema';
import {
  orderFileSchema,
  orderSchema,
  type FileRole,
  type Order,
  type OrderIntakeInput,
} from '../model/order.schema';

export async function sha256(file: File) {
  const digest = await crypto.subtle.digest('SHA-256', await file.arrayBuffer());
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('');
}

export async function createOrder(
  input: OrderIntakeInput,
  csrfToken: string,
  idempotencyKey: string,
): Promise<Order> {
  const command: components['schemas']['CreateOrderRequest'] = input;
  return apiRequest({
    path: '/orders',
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': idempotencyKey },
    body: command,
    schema: orderSchema,
  });
}

export async function getOrder(orderId: string, signal?: AbortSignal): Promise<Order> {
  const response = await apiRequestWithMetadata({
    path: `/orders/${encodeURIComponent(orderId)}`,
    schema: orderSchema,
    ...(signal ? { signal } : {}),
  });
  const versionMatch = /^"(\d+)"$/.exec(response.headers.get('etag') ?? '');
  return versionMatch ? { ...response.data, version: Number(versionMatch[1]) } : response.data;
}

export interface UploadOrderFileInput {
  csrfToken: string;
  digest: string;
  file: File;
  idempotencyKey: string;
  onProgress: (percentage: number) => void;
  orderId: string;
  role: FileRole;
  signal: AbortSignal;
  version: number;
}

const emptyTypeCodeMediaTypes: Readonly<Record<string, string>> = {
  '.r': 'text/x-r-source',
  '.py': 'text/x-python',
  '.ipynb': 'application/json',
  '.do': 'text/plain',
  '.sql': 'application/sql',
  '.sh': 'text/x-shellscript',
};

function fileExtension(name: string) {
  const lastDot = name.lastIndexOf('.');
  return lastDot === -1 ? '' : name.slice(lastDot).toLowerCase();
}

export function declaredMediaType(file: File, role: FileRole) {
  if (role === 'primary_paper') return 'application/pdf';
  if (file.type) return file.type;

  const name = file.name.toLowerCase();
  const extension = fileExtension(name);
  if (role === 'code') return emptyTypeCodeMediaTypes[extension] ?? 'application/octet-stream';
  if (role !== 'environment') return 'application/octet-stream';

  if (name === 'renv.lock') return 'application/json';
  if (name === 'dockerfile') return 'text/plain';
  if (name === 'requirements.txt' || (name.startsWith('requirements-') && extension === '.txt')) {
    return 'text/plain';
  }
  if (extension === '.lock') return 'text/plain';
  if (extension === '.yaml' || extension === '.yml') return 'application/yaml';
  if (extension === '.toml') return 'application/toml';
  return 'application/octet-stream';
}

export async function uploadOrderFile(input: UploadOrderFileInput) {
  const response = await uploadApiMultipart({
    path: `/orders/${encodeURIComponent(input.orderId)}/files`,
    headers: {
      'X-CSRF-Token': input.csrfToken,
      'If-Match': `"${input.version}"`,
      'Idempotency-Key': input.idempotencyKey,
    },
    fields: {
      role: input.role,
      originalDisplayName: input.file.name,
      sizeBytes: String(input.file.size),
      expectedSha256: input.digest,
      declaredMediaType: declaredMediaType(input.file, input.role),
    },
    file: input.file,
    signal: input.signal,
    onProgress: input.onProgress,
    schema: orderFileSchema,
  });
  const versionMatch = /^"(\d+)"$/.exec(response.etag ?? '');
  return {
    file: response.data,
    version: versionMatch ? Number(versionMatch[1]) : input.version + 1,
  };
}

export async function removeOrderFile(
  orderId: string,
  fileId: string,
  version: number,
  csrfToken: string,
) {
  const response = await apiRequestWithMetadata({
    path: `/orders/${encodeURIComponent(orderId)}/files/${encodeURIComponent(fileId)}`,
    method: 'DELETE',
    headers: {
      'X-CSRF-Token': csrfToken,
      'If-Match': `"${version}"`,
    },
    schema: z.undefined(),
  });
  const versionMatch = /^"(\d+)"$/.exec(response.headers.get('etag') ?? '');
  return versionMatch ? Number(versionMatch[1]) : version + 1;
}

export async function submitOrder(
  orderId: string,
  version: number,
  csrfToken: string,
  idempotencyKey: string,
  authorization: {
    termsAccepted: boolean;
    uploadsAuthorized: boolean;
    analysisUseAuthorized: boolean;
  },
): Promise<Order> {
  if (
    !authorization.termsAccepted ||
    !authorization.uploadsAuthorized ||
    !authorization.analysisUseAuthorized
  ) {
    throw new Error('Every required customer authorization must be confirmed.');
  }
  const command: components['schemas']['SubmitOrderRequest'] = {
    termsAccepted: authorization.termsAccepted,
    termsVersion: 'claimbounty-p0.1',
    uploadsAuthorized: authorization.uploadsAuthorized,
    analysisUseAuthorized: authorization.analysisUseAuthorized,
    externalRedistributionAuthorized: false,
  };
  return apiRequest({
    path: `/orders/${encodeURIComponent(orderId)}/submit`,
    method: 'POST',
    headers: {
      'X-CSRF-Token': csrfToken,
      'If-Match': `"${version}"`,
      'Idempotency-Key': idempotencyKey,
    },
    body: command,
    schema: orderSchema,
  });
}
