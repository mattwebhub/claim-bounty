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

function resolveUrl(pathname: string) {
  const base = environment.VITE_API_BASE_URL.endsWith('/')
    ? environment.VITE_API_BASE_URL
    : `${environment.VITE_API_BASE_URL}/`;
  return new URL(pathname.replace(/^\//, ''), base).toString();
}

export async function apiRequest<TSchema extends z.ZodType>(
  request: ApiRequest<TSchema>,
): Promise<z.output<TSchema>> {
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
    const response = await fetch(resolveUrl(request.path), {
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
    if (!envelope.success) {
      throw new ApiError({
        kind: 'decode',
        message: 'The server response did not match the success envelope.',
        retryable: false,
        status: response.status,
        requestId,
        details: z.treeifyError(envelope.error),
      });
    }

    const result = request.schema.safeParse(envelope.data.data);
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

    return result.data;
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
