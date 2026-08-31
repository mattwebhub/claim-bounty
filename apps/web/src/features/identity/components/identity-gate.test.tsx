import { http, HttpResponse } from 'msw';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { renderApplication } from '@/shared/test';
import { mockServer } from '@/shared/test/mocks/server';
import { sessionKey } from '../api/identity.queries';
import type { Session } from '../model/identity.schema';
import { IdentityGate } from './identity-gate';

describe('IdentityGate', () => {
  it('progressively requests a challenge and establishes an in-memory session', async () => {
    const user = userEvent.setup();
    const session: Session = {
      audience: 'submitter',
      csrfToken: 'c'.repeat(32),
      authorizationPolicyVersion: 'policy.1',
      expiresAt: '2026-08-30T18:00:00Z',
    };
    mockServer.use(
      http.post('*/api/v1/email-challenges', () =>
        HttpResponse.json({ data: { accepted: true } }, { status: 202 }),
      ),
      http.post('*/api/v1/email-challenges/verify', () => HttpResponse.json({ data: session })),
    );
    const { queryClient } = renderApplication(<IdentityGate audience="submitter" />);

    expect(screen.queryByLabelText('Verification code')).not.toBeInTheDocument();
    await user.type(screen.getByLabelText('Email address'), 'researcher@example.org');
    await user.click(screen.getByRole('button', { name: 'Send verification code' }));
    await screen.findByLabelText('Verification code');
    expect(screen.getByRole('heading', { name: 'Check your inbox' })).toHaveFocus();

    await user.type(screen.getByLabelText('Verification code'), '123456');
    await user.click(screen.getByRole('button', { name: 'Verify and continue' }));
    await expect.poll(() => queryClient.getQueryData(sessionKey)).toEqual(session);
  });

  it('reports an invalid email without sending a request', async () => {
    const user = userEvent.setup();
    renderApplication(<IdentityGate audience="administrator" />);
    await user.type(screen.getByLabelText('Email address'), 'not-an-email');
    await user.click(screen.getByRole('button', { name: 'Send verification code' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Enter a valid email address.');
  });

  it('shows the request reference and recovers when the challenge is requested again', async () => {
    const user = userEvent.setup();
    let attempts = 0;
    mockServer.use(
      http.post('*/api/v1/email-challenges', () => {
        attempts += 1;
        return attempts === 1
          ? HttpResponse.json(
              { error: { message: 'Email delivery failed.', requestId: 'req-email-1' } },
              { status: 503 },
            )
          : HttpResponse.json({ data: { accepted: true } }, { status: 202 });
      }),
    );
    renderApplication(<IdentityGate audience="submitter" />);

    await user.type(screen.getByLabelText('Email address'), 'researcher@example.org');
    await user.click(screen.getByRole('button', { name: 'Send verification code' }));
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Email delivery failed. Request ID: req-email-1.',
    );

    await user.click(screen.getByRole('button', { name: 'Send verification code' }));
    expect(await screen.findByRole('heading', { name: 'Check your inbox' })).toBeVisible();
    expect(attempts).toBe(2);
  });
});
