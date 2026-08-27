import { z } from 'zod';

export const fieldIssueSchema = z.object({
  path: z.string(),
  code: z.string(),
  message: z.string(),
});

const fieldIssuesSchema = z.array(fieldIssueSchema);

const errorEnvelopeSchema = z.object({
  error: z.object({
    code: z.string().optional(),
    message: z.string().optional(),
    requestId: z.string().optional(),
    details: z.unknown().optional(),
  }),
});

export type ApiErrorKind = 'http' | 'network' | 'timeout' | 'decode' | 'unknown';

export interface ApiErrorOptions {
  kind: ApiErrorKind;
  message: string;
  retryable: boolean;
  status?: number | undefined;
  code?: string | undefined;
  requestId?: string | undefined;
  details?: unknown;
  fieldIssues?: z.infer<typeof fieldIssueSchema>[] | undefined;
  cause?: unknown;
}

export class ApiError extends Error {
  readonly kind: ApiErrorKind;
  readonly retryable: boolean;
  readonly status: number | undefined;
  readonly code: string | undefined;
  readonly requestId: string | undefined;
  readonly details: unknown;
  readonly fieldIssues: z.infer<typeof fieldIssueSchema>[] | undefined;

  constructor(options: ApiErrorOptions) {
    super(options.message, options.cause === undefined ? undefined : { cause: options.cause });
    this.name = 'ApiError';
    this.kind = options.kind;
    this.retryable = options.retryable;
    this.status = options.status;
    this.code = options.code;
    this.requestId = options.requestId;
    this.details = options.details;
    this.fieldIssues = options.fieldIssues;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export function isRetryableStatus(status: number) {
  return status === 408 || status === 425 || status === 429 || status >= 500;
}

export function normalizeHttpError(status: number, payload: unknown, requestId?: string) {
  const parsed = errorEnvelopeSchema.safeParse(payload);
  const body = parsed.success ? parsed.data.error : undefined;
  const resolvedRequestId = body?.requestId ?? requestId;
  const parsedFieldIssues = fieldIssuesSchema.safeParse(body?.details);

  return new ApiError({
    kind: 'http',
    status,
    code: body?.code,
    message: body?.message ?? `Request failed with status ${status}.`,
    requestId: resolvedRequestId,
    details: body?.details,
    fieldIssues: parsedFieldIssues.success ? parsedFieldIssues.data : undefined,
    retryable: isRetryableStatus(status),
  });
}
