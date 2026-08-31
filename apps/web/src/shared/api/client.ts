import { z } from 'zod';
import { environment } from '@/shared/config';
import { ApiError, normalizeHttpError } from '@/shared/api/errors';

const successEnvelopeSchema = z.object({ data: z.unknown() });

export interface ApiRequest<TSchema extends z.ZodType> {
  path: string;
  schema: TSchema;
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  headers?: HeadersInit;
  signal?: AbortSignal;
}

async function readJson(response: Response): Promise<unknown> {
  const contentType = response.headers.get('content-type');
  if (!contentType?.includes('application/json')) return undefined;

  try {
    return await response.json();
  } catch (cause) {
    throw new ApiError({
      kind: 'decode',
      message: 'The server returned malformed JSON.',
      retryable: false,
      status: response.status,
      requestId: response.headers.get('x-request-id') ?? undefined,
      cause,
    });
  }
}

export function resolveApiUrl(pathname: string) {
  const base = environment.VITE_API_BASE_URL.endsWith('/')
    ? environment.VITE_API_BASE_URL
    : `${environment.VITE_API_BASE_URL}/`;
  const resolvedBase = new URL(base, window.location.origin);
  if (pathname.startsWith('/api/')) return new URL(pathname, resolvedBase.origin).toString();
  return new URL(pathname.replace(/^\//, ''), resolvedBase).toString();
}

export async function apiRequest<TSchema extends z.ZodType>(
  request: ApiRequest<TSchema>,
): Promise<z.output<TSchema>> {
  const result = await apiRequestWithMetadata(request);
  return result.data;
}

export interface ApiResponse<T> {
  data: T;
  headers: Headers;
}

export interface ApiDownloadResponse {
  delivery: 'streamed';
  headers: Headers;
}

export interface ApiDownloadRequest {
  filename: string;
  path: string;
  signal?: AbortSignal;
  validateHeaders?: (headers: Headers) => void;
}

type SaveFilePicker = (options: {
  suggestedName: string;
  types: { accept: Record<string, string[]>; description: string }[];
}) => Promise<FileSystemFileHandle>;

function getSaveFilePicker(): SaveFilePicker | null {
  const candidate: unknown = Reflect.get(window, 'showSaveFilePicker');
  return typeof candidate === 'function' ? (candidate.bind(window) as SaveFilePicker) : null;
}

export function supportsStreamingApiDownloads() {
  return getSaveFilePicker() !== null && typeof ReadableStream === 'function';
}

export async function downloadApiFile(request: ApiDownloadRequest): Promise<ApiDownloadResponse> {
  try {
    const saveFilePicker = getSaveFilePicker();
    if (!saveFilePicker || typeof ReadableStream !== 'function') {
      throw new ApiError({
        kind: 'unknown',
        message: 'Direct-to-disk downloads are unavailable in this browser.',
        retryable: false,
      });
    }
    const fileHandle = await saveFilePicker({
      suggestedName: request.filename,
      types: [
        {
          description: 'ZIP archive',
          accept: { 'application/zip': ['.zip'] },
        },
      ],
    });
    const response = await fetch(resolveApiUrl(request.path), {
      credentials: 'same-origin',
      headers: { Accept: 'application/zip' },
      ...(request.signal ? { signal: request.signal } : {}),
    });
    if (!response.ok) {
      const payload = await readJson(response);
      throw normalizeHttpError(
        response.status,
        payload,
        response.headers.get('x-request-id') ?? undefined,
      );
    }
    request.validateHeaders?.(response.headers);

    if (!response.body) {
      throw new ApiError({
        kind: 'decode',
        message: 'The server returned an empty download stream.',
        retryable: false,
      });
    }

    const writable = await fileHandle.createWritable();
    await response.body.pipeTo(writable, request.signal ? { signal: request.signal } : undefined);
    return { delivery: 'streamed', headers: response.headers };
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (request.signal?.aborted) throw error;
    if (error instanceof DOMException && error.name === 'AbortError') throw error;
    if (error instanceof TypeError) {
      throw new ApiError({
        kind: 'network',
        message: 'The server could not be reached. Check your connection and try again.',
        retryable: true,
        cause: error,
      });
    }
    throw new ApiError({
      kind: 'unknown',
      message: 'An unexpected download error occurred.',
      retryable: false,
      cause: error,
    });
  }
}

export async function apiRequestWithMetadata<TSchema extends z.ZodType>(
  request: ApiRequest<TSchema>,
): Promise<ApiResponse<z.output<TSchema>>> {
  const timeoutController = new AbortController();
  const timeout = window.setTimeout(() => {
    timeoutController.abort(new DOMException('Request timed out.', 'TimeoutError'));
  }, environment.VITE_API_TIMEOUT_MS);
  const signal = request.signal
    ? AbortSignal.any([request.signal, timeoutController.signal])
    : timeoutController.signal;
  const headers = new Headers(request.headers);
  headers.set('Accept', 'application/json');
  if (request.body !== undefined) headers.set('Content-Type', 'application/json');

  try {
    const response = await fetch(resolveApiUrl(request.path), {
      method: request.method ?? 'GET',
      credentials: 'same-origin',
      headers,
      ...(request.body === undefined ? {} : { body: JSON.stringify(request.body) }),
      signal,
    });
    const payload = await readJson(response);
    const requestId = response.headers.get('x-request-id') ?? undefined;

    if (!response.ok) {
      throw normalizeHttpError(response.status, payload, requestId);
    }

    const envelope = successEnvelopeSchema.safeParse(payload);
    if (response.status !== 204 && !envelope.success) {
      throw new ApiError({
        kind: 'decode',
        message: 'The server response did not match the success envelope.',
        retryable: false,
        status: response.status,
        requestId,
        details: z.treeifyError(envelope.error),
      });
    }

    const result = request.schema.safeParse(
      response.status === 204 ? undefined : envelope.success ? envelope.data.data : undefined,
    );
    if (!result.success) {
      throw new ApiError({
        kind: 'decode',
        message: 'The server response did not match the expected contract.',
        retryable: false,
        status: response.status,
        requestId,
        details: z.treeifyError(result.error),
      });
    }

    return { data: result.data, headers: response.headers };
  } catch (error) {
    if (error instanceof ApiError) throw error;

    if (timeoutController.signal.aborted) {
      throw new ApiError({
        kind: 'timeout',
        message: 'The request timed out. Try again.',
        retryable: true,
        cause: error,
      });
    }

    if (request.signal?.aborted) throw error;

    if (error instanceof TypeError) {
      throw new ApiError({
        kind: 'network',
        message: 'The server could not be reached. Check your connection and try again.',
        retryable: true,
        cause: error,
      });
    }

    throw new ApiError({
      kind: 'unknown',
      message: 'An unexpected request error occurred.',
      retryable: false,
      cause: error,
    });
  } finally {
    window.clearTimeout(timeout);
  }
}

export interface ApiMultipartUpload<TSchema extends z.ZodType> {
  file: File;
  fields: Record<string, string>;
  headers?: HeadersInit;
  onProgress?: (percentage: number) => void;
  path: string;
  schema: TSchema;
  signal?: AbortSignal;
}

export interface ApiMultipartResponse<T> {
  data: T;
  etag?: string;
}

function parseXhrJson(request: XMLHttpRequest): unknown {
  if (!request.responseText) return undefined;
  try {
    return JSON.parse(request.responseText) as unknown;
  } catch (cause) {
    throw new ApiError({
      kind: 'decode',
      message: 'The server returned malformed JSON.',
      retryable: false,
      status: request.status,
      requestId: request.getResponseHeader('x-request-id') ?? undefined,
      cause,
    });
  }
}

/**
 * Sends one bounded multipart upload through the configured API gateway.
 * The caller supplies only an API path; storage locations and credentials
 * never cross the browser boundary.
 */
export function uploadApiMultipart<TSchema extends z.ZodType>({
  file,
  fields,
  headers,
  onProgress,
  path,
  schema,
  signal,
}: ApiMultipartUpload<TSchema>): Promise<ApiMultipartResponse<z.output<TSchema>>> {
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    const form = new FormData();
    for (const [name, value] of Object.entries(fields)) form.append(name, value);
    form.append('file', file);

    request.open('POST', resolveApiUrl(path));
    request.withCredentials = true;
    request.setRequestHeader('Accept', 'application/json');
    const requestHeaders = new Headers(headers);
    requestHeaders.forEach((value, name) => {
      request.setRequestHeader(name, value);
    });

    const abort = () => {
      request.abort();
    };
    const finish = () => signal?.removeEventListener('abort', abort);

    request.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) onProgress?.(Math.round((event.loaded / event.total) * 100));
    });
    request.addEventListener('load', () => {
      finish();
      let payload: unknown;
      try {
        payload = parseXhrJson(request);
      } catch (error) {
        reject(
          error instanceof Error ? error : new Error('The upload response could not be read.'),
        );
        return;
      }

      const requestId = request.getResponseHeader('x-request-id') ?? undefined;
      if (request.status < 200 || request.status >= 300) {
        reject(normalizeHttpError(request.status, payload, requestId));
        return;
      }

      const envelope = successEnvelopeSchema.safeParse(payload);
      const result = envelope.success ? schema.safeParse(envelope.data.data) : null;
      if (!envelope.success || !result?.success) {
        const details = result && !result.success ? z.treeifyError(result.error) : undefined;
        reject(
          new ApiError({
            kind: 'decode',
            message: 'The server response did not match the expected upload contract.',
            retryable: false,
            status: request.status,
            requestId,
            details,
          }),
        );
        return;
      }

      const etag = request.getResponseHeader('ETag') ?? undefined;
      resolve({ data: result.data, ...(etag ? { etag } : {}) });
    });
    request.addEventListener('error', () => {
      finish();
      reject(
        new ApiError({
          kind: 'network',
          message: 'The upload connection failed. Try again.',
          retryable: true,
        }),
      );
    });
    request.addEventListener('abort', () => {
      finish();
      reject(new DOMException('The upload was cancelled.', 'AbortError'));
    });
    if (signal?.aborted) {
      reject(new DOMException('The upload was cancelled.', 'AbortError'));
      return;
    }
    signal?.addEventListener('abort', abort, { once: true });
    request.send(form);
  });
}
