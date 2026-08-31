import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Alert, AlertDescription, AlertTitle, Button, Input } from '@/shared/ui';
import { isApiError } from '@/shared/api';
import { useConfirmEmailVerification, useRequestEmailVerification } from '../api/identity.queries';
import {
  emailVerificationSchema,
  codeConfirmationSchema,
  type Audience,
  type EmailVerificationValues,
  type CodeConfirmationValues,
} from '../model/identity.schema';

interface IdentityGateProps {
  audience: Audience;
}

function errorMessage(error: unknown) {
  if (!isApiError(error)) return 'The request could not be completed. Try again.';
  return `${error.message}${error.requestId ? ` Request ID: ${error.requestId}.` : ''}`;
}

export function IdentityGate({ audience }: IdentityGateProps) {
  const [verificationRequested, setVerificationRequested] = useState(false);
  const [requestedEmail, setRequestedEmail] = useState('');
  const codeHeading = useRef<HTMLHeadingElement>(null);
  const requestMutation = useRequestEmailVerification(audience);
  const confirmationMutation = useConfirmEmailVerification();
  const emailForm = useForm<EmailVerificationValues>({
    resolver: zodResolver(emailVerificationSchema),
    defaultValues: { email: '' },
  });
  const codeForm = useForm<CodeConfirmationValues>({
    resolver: zodResolver(codeConfirmationSchema),
    defaultValues: { code: '' },
  });

  useEffect(() => {
    if (verificationRequested) codeHeading.current?.focus();
  }, [verificationRequested]);

  const requestCode = emailForm.handleSubmit(async ({ email }) => {
    try {
      await requestMutation.mutateAsync(email);
    } catch {
      return;
    }
    setRequestedEmail(email);
    setVerificationRequested(true);
  });
  const confirmCode = codeForm.handleSubmit(async ({ code }) => {
    const email = requestedEmail || emailForm.getValues('email');
    if (!email) {
      codeForm.setError('code', {
        type: 'server',
        message: 'Enter your email and request a new code.',
      });
      return;
    }
    let session;
    try {
      session = await confirmationMutation.mutateAsync({ email, audience, code });
    } catch {
      return;
    }
    if (session.audience !== audience) {
      codeForm.setError('code', {
        type: 'server',
        message: `This link does not grant ${audience} access.`,
      });
    }
  });

  return (
    <section className="auth-card" aria-labelledby="verify-title">
      <p className="step-label">{audience === 'submitter' ? 'Step 2 of 4' : 'Step 1 of 2'}</p>
      <h2 id="verify-title">
        {audience === 'administrator' ? 'Admin sign in' : 'Verify your email'}
      </h2>
      <p className="section-copy">
        We send a six-digit code that expires after ten minutes. Your session stays in a secure
        browser cookie and is never saved to browser storage.
      </p>
      <form className="form-stack" onSubmit={requestCode} noValidate>
        <label htmlFor={`${audience}-email`}>Email address</label>
        <Input
          id={`${audience}-email`}
          type="email"
          autoComplete="email"
          invalid={Boolean(emailForm.formState.errors.email)}
          aria-describedby={
            emailForm.formState.errors.email ? `${audience}-email-error` : undefined
          }
          {...emailForm.register('email')}
        />
        {emailForm.formState.errors.email ? (
          <span id={`${audience}-email-error`} className="field-error" role="alert">
            {emailForm.formState.errors.email.message}
          </span>
        ) : null}
        <Button type="submit" disabled={requestMutation.isPending}>
          {requestMutation.isPending ? 'Sending…' : 'Send verification code'}
        </Button>
      </form>
      {requestMutation.isError ? (
        <Alert variant="destructive">
          <div>
            <AlertTitle>Code could not be sent</AlertTitle>
            <AlertDescription>{errorMessage(requestMutation.error)}</AlertDescription>
          </div>
        </Alert>
      ) : null}
      {verificationRequested ? (
        <div className="token-panel">
          <h3 ref={codeHeading} tabIndex={-1}>
            Check your inbox
          </h3>
          <p>
            Enter the code sent to <strong>{requestedEmail}</strong>.
          </p>
          <form className="form-stack" onSubmit={confirmCode} noValidate>
            <label htmlFor={`${audience}-code`}>Verification code</label>
            <Input
              id={`${audience}-code`}
              inputMode="numeric"
              pattern="[0-9]{6}"
              maxLength={6}
              autoComplete="one-time-code"
              invalid={Boolean(codeForm.formState.errors.code)}
              aria-describedby={
                codeForm.formState.errors.code ? `${audience}-code-error` : undefined
              }
              {...codeForm.register('code')}
            />
            {codeForm.formState.errors.code ? (
              <span id={`${audience}-code-error`} className="field-error" role="alert">
                {codeForm.formState.errors.code.message}
              </span>
            ) : null}
            <Button type="submit" disabled={confirmationMutation.isPending}>
              {confirmationMutation.isPending ? 'Verifying…' : 'Verify and continue'}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                setVerificationRequested(false);
                setRequestedEmail('');
                codeForm.reset();
              }}
            >
              Use a different email
            </Button>
          </form>
          {confirmationMutation.isError ? (
            <Alert variant="destructive">
              <div>
                <AlertTitle>Verification failed</AlertTitle>
                <AlertDescription>{errorMessage(confirmationMutation.error)}</AlertDescription>
              </div>
            </Alert>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
