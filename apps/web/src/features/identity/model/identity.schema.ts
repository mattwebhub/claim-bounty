import { z } from 'zod';
import type { components } from '@/shared/api/generated/schema';

export type Audience = components['schemas']['Session']['audience'];
export type Session = components['schemas']['Session'];

export const emailVerificationSchema = z.object({
  email: z.email('Enter a valid email address.').max(254),
});

export const codeConfirmationSchema = z.object({
  code: z
    .string()
    .trim()
    .regex(/^[0-9]{6}$/, 'Enter the six-digit code.'),
});

export const sessionSchema: z.ZodType<Session> = z.object({
  audience: z.enum(['submitter', 'administrator']),
  csrfToken: z.string().min(32).max(256),
  authorizationPolicyVersion: z.string().regex(/^[a-z0-9][a-z0-9._-]{0,99}$/),
  expiresAt: z.iso.datetime({ offset: true }),
});

export type EmailVerificationValues = z.infer<typeof emailVerificationSchema>;
export type CodeConfirmationValues = z.infer<typeof codeConfirmationSchema>;
