export {
  apiRequest,
  apiRequestWithMetadata,
  downloadApiFile,
  resolveApiUrl,
  supportsStreamingApiDownloads,
  uploadApiMultipart,
  type ApiMultipartResponse,
  type ApiMultipartUpload,
  type ApiDownloadResponse,
  type ApiDownloadRequest,
  type ApiRequest,
} from '@/shared/api/client';
export {
  ApiError,
  isApiError,
  isRetryableStatus,
  normalizeHttpError,
  type ApiErrorKind,
  type ApiErrorOptions,
} from '@/shared/api/errors';
