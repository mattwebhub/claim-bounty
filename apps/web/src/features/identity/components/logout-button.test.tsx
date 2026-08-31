import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { useLocation } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { mockServer, renderApplication } from '@/shared/test';
import { LogoutButton } from './logout-button';

const session = {
  audience: 'submitter',
  csrfToken: 'c'.repeat(32),
  authorizationPolicyVersion: 'policy.1',
  expiresAt: '2026-08-30T18:00:00Z',
} as const;

function LocationProbe() {
  const location = useLocation();
  return <output aria-label="Current location">{`${location.pathname}${location.search}`}</output>;
}

describe('LogoutButton', () => {
  it('revokes the session, clears private queries, and removes the draft URL', async () => {
    const user = userEvent.setup();
    let loggedOut = false;
    let csrfToken: string | null = null;
    mockServer.use(
      http.get('*/api/v1/session', () =>
        loggedOut
          ? HttpResponse.json({ error: { message: 'Sign in required.' } }, { status: 401 })
          : HttpResponse.json({ data: session }),
      ),
      http.delete('*/api/v1/session', ({ request }) => {
        csrfToken = request.headers.get('x-csrf-token');
        loggedOut = true;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    const { queryClient } = renderApplication(
      <>
        <LogoutButton />
        <LocationProbe />
      </>,
      { route: '/?draft=123e4567-e89b-12d3-a456-426614174000' },
    );
    queryClient.setQueryData(['claim-orders', 'private-order'], { title: 'Private draft' });

    await user.click(await screen.findByRole('button', { name: 'Sign out' }));

    expect(await screen.findByLabelText('Current location')).toHaveTextContent(/^\/$/);
    expect(csrfToken).toBe(session.csrfToken);
    expect(queryClient.getQueryData(['claim-orders', 'private-order'])).toBeUndefined();
    expect(screen.queryByRole('button', { name: 'Sign out' })).not.toBeInTheDocument();
  });

  it('keeps the session available and announces a failed revocation', async () => {
    const user = userEvent.setup();
    mockServer.use(
      http.get('*/api/v1/session', () => HttpResponse.json({ data: session })),
      http.delete('*/api/v1/session', () =>
        HttpResponse.json(
          { error: { message: 'Session could not be revoked.', requestId: 'req-logout-1' } },
          { status: 503 },
        ),
      ),
    );
    renderApplication(<LogoutButton />);

    await user.click(await screen.findByRole('button', { name: 'Sign out' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Session could not be revoked. Request ID: req-logout-1.',
    );
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeEnabled();
  });
});
