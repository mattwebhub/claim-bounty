import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderApplication } from '@/shared/test';
import { ClaimIntake } from './claim-intake';

const mutations = vi.hoisted(() => ({
  create: vi.fn(),
  remove: vi.fn(),
  submit: vi.fn(),
  upload: vi.fn(),
}));

vi.mock('../api/order.queries', () => ({
  useCreateOrder: () => ({ isPending: false, mutateAsync: mutations.create }),
  useOrder: () => ({ data: undefined, isPending: false, isError: false }),
  useRemoveOrderFile: () => ({ isPending: false, mutateAsync: mutations.remove }),
  useSubmitOrder: () => ({ isPending: false, mutateAsync: mutations.submit }),
  useUploadOrderFile: () => ({ isPending: false, mutateAsync: mutations.upload }),
}));

const draft = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  publicReference: 'CB-ABCDEF123456',
  status: 'draft',
  version: 1,
  title: 'Replication study',
  purpose: 'Check the result.',
  targetClaim: { text: 'The treatment improved scores.', sourceLocation: 'Page 4' },
  files: [],
  piiRetention: {
    policyVersion: 'policy.1',
    disposition: 'hard_delete',
    sourceDeleteAfter: '2026-09-15T10:00:00Z',
    piiDeleteAfter: '2026-09-30T10:00:00Z',
  },
  createdAt: '2026-08-30T10:00:00Z',
  updatedAt: '2026-08-30T10:00:00Z',
} as const;

const uploadedFile = {
  id: '223e4567-e89b-12d3-a456-426614174000',
  role: 'primary_paper',
  originalDisplayName: 'paper.pdf',
  sizeBytes: 5,
  sha256: 'a'.repeat(64),
  storage: { objectVersion: 'v1', sha256: 'a'.repeat(64), immutability: 'write_once' },
  declaredMediaType: 'application/pdf',
  detectedMediaType: 'application/pdf',
  status: 'clean',
  createdAt: '2026-08-30T10:00:00Z',
  updatedAt: '2026-08-30T10:01:00Z',
} as const;

describe('ClaimIntake receipt', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mutations.create.mockResolvedValue(draft);
    mutations.upload.mockResolvedValue(uploadedFile);
    mutations.submit.mockResolvedValue({
      ...draft,
      status: 'submitted',
      version: 3,
      files: [uploadedFile],
      submittedAt: '2026-08-30T10:02:00Z',
    });
  });

  it('announces a private receipt after every upload completes and terms are accepted', async () => {
    const user = userEvent.setup();
    renderApplication(<ClaimIntake csrfToken={'c'.repeat(32)} />);

    await user.upload(
      screen.getByLabelText('Evidence files'),
      new File(['paper'], 'paper.pdf', { type: 'application/pdf' }),
    );
    await user.type(screen.getByLabelText('Study title'), 'Replication study');
    await user.type(
      screen.getByLabelText('What should the review establish?'),
      'Check the result.',
    );
    await user.type(screen.getByLabelText('Exact target claim'), 'The treatment improved scores.');
    await user.type(screen.getByLabelText('Claim location'), 'Page 4');
    await user.click(screen.getByRole('button', { name: 'Create intake and upload' }));

    const terms = await screen.findByRole('checkbox', { name: /I accept the Peer2Paper/ });
    await waitFor(() => {
      expect(screen.getByRole('listitem')).toHaveTextContent('complete');
    });
    await user.click(terms);
    await user.click(screen.getByRole('checkbox', { name: /retain and privately inspect/ }));
    await user.click(screen.getByRole('checkbox', { name: /private derived analysis files/ }));
    expect(screen.getByText(/External redistribution is not authorized/)).toBeVisible();
    await user.click(screen.getByRole('button', { name: 'Submit for review' }));

    expect(mutations.submit).toHaveBeenCalledWith({
      orderId: draft.id,
      csrfToken: 'c'.repeat(32),
      termsAccepted: true,
      uploadsAuthorized: true,
      analysisUseAuthorized: true,
    });

    const heading = await screen.findByRole('heading', { name: 'Your evidence is in review' });
    expect(heading).toHaveFocus();
    expect(screen.getByText('CB-ABCDEF123456')).toBeVisible();
    expect(screen.getByText('1', { selector: 'dd' })).toBeVisible();
  });
});
