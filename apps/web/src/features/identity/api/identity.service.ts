import { z } from 'zod';
import { apiRequest } from '@/shared/api';
import type { components } from '@/shared/api/generated/schema';
import { sessionSchema, type Audience, type Session } from '../model/identity.schema';

const acceptedSchema = z.object({ accepted: z.literal(true) });

export async function requestEmailVerification(email: string, audience: Audience) {
  const command: components['schemas']['RequestEmailChallengeRequest'] = { email, audience };
  await apiRequest({
    path: '/email-challenges',
    method: 'POST',
    body: command,
    schema: acceptedSchema,
  });
}

export async function confirmEmailVerification(
  email: string,
  audience: Audience,
  code: string,
): Promise<Session> {
  const command: components['schemas']['VerifyEmailChallengeRequest'] = { email, audience, code };
  return apiRequest({
    path: '/email-challenges/verify',
    method: 'POST',
    body: command,
    schema: sessionSchema,
  });
}

export function getSession(signal?: AbortSignal): Promise<Session> {
  return apiRequest({
    path: '/session',
    schema: sessionSchema,
    ...(signal ? { signal } : {}),
  });
}

export async function logout(csrfToken: string): Promise<void> {
  await apiRequest({
    path: '/session',
    method: 'DELETE',
    headers: { 'X-CSRF-Token': csrfToken },
    schema: z.undefined(),
  });
}
