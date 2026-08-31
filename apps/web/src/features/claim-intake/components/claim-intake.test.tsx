import { fireEvent, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { mockServer, renderApplication } from '@/shared/test';
import { ClaimIntake } from './claim-intake';

describe('ClaimIntake', () => {
  it('shows the branded audit action while files are dragged over the input', () => {
    renderApplication(<ClaimIntake csrfToken={'c'.repeat(32)} />);

    const dropSurface = screen.getByRole('button', { name: /choose or drop more evidence here/i });
    const logo = screen.getByRole('img', { name: 'Peer2Paper' });

    expect(logo).toHaveAttribute('src', '/claimbounty-fox-loupe-icon.png');
    expect(dropSurface).toHaveAttribute('data-drag-active', 'false');

    fireEvent.dragEnter(dropSurface);
    expect(dropSurface).toHaveAttribute('data-drag-active', 'true');

    fireEvent.drop(dropSurface, { dataTransfer: { files: [] } });
    expect(dropSurface).toHaveAttribute('data-drag-active', 'false');
  });

  it('classifies the first PDF and lets a keyboard user remove selected evidence', async () => {
    const user = userEvent.setup();
    renderApplication(<ClaimIntake csrfToken={'c'.repeat(32)} />);
    const paper = new File(['paper'], 'paper.pdf', { type: 'application/pdf' });
    const data = new File(['a,b'], 'data.csv', { type: 'text/csv' });

    await user.upload(screen.getByLabelText('Evidence files'), [paper, data]);

    const list = screen.getByRole('list', { name: 'Selected evidence files' });
    expect(within(list).getByText('paper.pdf')).toBeVisible();
    expect(within(list).getByText('data.csv')).toBeVisible();
    expect(screen.getAllByRole('combobox')[0]).toHaveValue('primary_paper');
    expect(screen.getAllByRole('combobox')[1]).toHaveValue('other_evidence');

    await user.click(screen.getByRole('button', { name: 'Remove data.csv' }));
    expect(screen.queryByText('data.csv')).not.toBeInTheDocument();
    expect(screen.getByText('paper.pdf')).toBeVisible();
  });

  it('requires a primary PDF before creating a secure intake', async () => {
    const user = userEvent.setup();
    renderApplication(<ClaimIntake csrfToken={'c'.repeat(32)} />);

    await user.upload(
      screen.getByLabelText('Evidence files'),
      new File(['data'], 'results.csv', { type: 'text/csv' }),
    );
    await user.type(screen.getByLabelText('Study title'), 'Replication study');
    await user.type(
      screen.getByLabelText('What should the review establish?'),
      'Check the result.',
    );
    await user.type(screen.getByLabelText('Exact target claim'), 'The treatment improved scores.');
    await user.type(screen.getByLabelText('Claim location'), 'Page 4, Table 2');
    await user.click(screen.getByRole('button', { name: 'Create intake and upload' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Add exactly one PDF and label it Primary paper.',
    );
    expect(screen.getByLabelText('Evidence files')).toHaveFocus();
  });

  it('restores every immutable permission and privacy declaration from a draft', async () => {
    const orderId = '123e4567-e89b-12d3-a456-426614174000';
    mockServer.use(
      http.get(`*/api/v1/orders/${orderId}`, () =>
        HttpResponse.json({
          data: {
            id: orderId,
            publicReference: 'CB-ABCDEF123456',
            status: 'draft',
            version: 1,
            title: 'Stored study',
            purpose: 'Review the stored claim.',
            targetClaim: { text: 'The stored claim.', sourceLocation: 'Table 2' },
            permissions: { executeSuppliedCode: true, externalSearch: true },
            privacy: { containsParticipantLevelData: true, containsDirectIdentifiers: true },
            files: [],
            piiRetention: {
              policyVersion: 'policy.1',
              disposition: 'hard_delete',
              sourceDeleteAfter: '2026-09-15T10:00:00Z',
              piiDeleteAfter: '2026-09-30T10:00:00Z',
            },
            createdAt: '2026-08-30T10:00:00Z',
            updatedAt: '2026-08-30T10:00:00Z',
          },
        }),
      ),
    );

    renderApplication(<ClaimIntake csrfToken={'c'.repeat(32)} draftOrderId={orderId} />);

    expect(
      await screen.findByRole('checkbox', {
        name: 'Execute code I supply in an isolated environment',
      }),
    ).toBeChecked();
    expect(
      screen.getByRole('checkbox', { name: 'Search external scholarly sources' }),
    ).toBeChecked();
    expect(
      screen.getByRole('checkbox', { name: 'Files contain participant-level data' }),
    ).toBeChecked();
    expect(
      screen.getByRole('checkbox', { name: 'Files contain direct personal identifiers' }),
    ).toBeChecked();
  });
});
