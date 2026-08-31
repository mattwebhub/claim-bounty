import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  confirmEmailVerification,
  getSession,
  logout,
  requestEmailVerification,
} from './identity.service';
import type { Audience, Session } from '../model/identity.schema';

export const sessionKey = ['identity', 'session'] as const;

export function useSession() {
  return useQuery<Session>({
    queryKey: sessionKey,
    queryFn: ({ signal }) => getSession(signal),
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: 'always',
  });
}

export function useRequestEmailVerification(audience: Audience) {
  return useMutation({ mutationFn: (email: string) => requestEmailVerification(email, audience) });
}

export function useConfirmEmailVerification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ email, audience, code }: { email: string; audience: Audience; code: string }) =>
      confirmEmailVerification(email, audience, code),
    onSuccess: (session) => queryClient.setQueryData(sessionKey, session),
  });
}

export function useClearSession() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.removeQueries({ queryKey: sessionKey });
  };
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.clear();
    },
  });
}
