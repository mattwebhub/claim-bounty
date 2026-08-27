export { apiRequest, type ApiRequest } from '@/shared/api/client';
export {
  ApiError,
  isApiError,
  isRetryableStatus,
  normalizeHttpError,
  type ApiErrorKind,
  type ApiErrorOptions,
} from '@/shared/api/errors';
