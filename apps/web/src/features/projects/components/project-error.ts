import { isApiError } from '@/shared/api';

export interface ProjectFieldIssue {
  path: string;
  message: string;
}

export function getProjectFieldIssues(error: unknown): ProjectFieldIssue[] {
  if (!isApiError(error)) return [];
  return error.fieldIssues?.map(({ message, path }) => ({ message, path })) ?? [];
}

export function getProjectErrorDescription(error: unknown): string {
  if (!isApiError(error)) return 'An unexpected error occurred. Try again.';
  const requestReference = error.requestId ? ` Reference: ${error.requestId}.` : '';
  return `${error.message}${requestReference}`;
}
