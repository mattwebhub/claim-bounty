import { QueryClient } from '@tanstack/react-query';
import { isApiError } from '@/shared/api';

const retryDelay = (attempt: number) => Math.min(1_000 * 2 ** attempt, 30_000);

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 5 * 60_000,
        refetchOnWindowFocus: false,
        retry: (attempt, error) => {
          if (isApiError(error) && !error.retryable) return false;
          return attempt < 2;
        },
        retryDelay,
      },
      mutations: {
        retry: false,
      },
    },
  });
}
