import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { mockServer } from '@/shared/test';
import { declaredMediaType, getOrder, submitOrder } from './order.service';

const order = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  publicReference: 'CB-ABCDEF123456',
  status: 'uploading',
  version: 2,
  title: 'Retention intervention',
  purpose: 'Check the reported effect.',
  targetClaim: { text: 'Retention increased.', sourceLocation: 'Table 2' },
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
} as const;

describe('order service recovery and authorization', () => {
  it('restores the authoritative order version from its ETag', async () => {
    mockServer.use(
      http.get('*/api/v1/orders/:orderId', () =>
        HttpResponse.json({ data: order }, { headers: { ETag: '"8"' } }),
      ),
    );

    await expect(getOrder(order.id)).resolves.toMatchObject({ id: order.id, version: 8 });
  });

  it('persists each explicit authorization and hard-disables external redistribution', async () => {
    let submittedBody: unknown;
    mockServer.use(
      http.post('*/api/v1/orders/:orderId/submit', async ({ request }) => {
        submittedBody = await request.json();
        return HttpResponse.json({ data: { ...order, status: 'submitted', version: 3 } });
      }),
    );

    await submitOrder(order.id, 2, 'c'.repeat(32), 'stable-submit-key', {
      termsAccepted: true,
      uploadsAuthorized: true,
      analysisUseAuthorized: true,
    });

    expect(submittedBody).toEqual({
      termsAccepted: true,
      termsVersion: 'claimbounty-p0.1',
      uploadsAuthorized: true,
      analysisUseAuthorized: true,
      externalRedistributionAuthorized: false,
    });
  });

  it('does not send a submission when a required customer authorization is missing', async () => {
    await expect(
      submitOrder(order.id, 2, 'c'.repeat(32), 'stable-submit-key', {
        termsAccepted: true,
        uploadsAuthorized: true,
        analysisUseAuthorized: false,
      }),
    ).rejects.toThrow('Every required customer authorization must be confirmed.');
  });
});

describe('empty browser media types', () => {
  it.each([
    ['analysis.R', 'code', 'text/x-r-source'],
    ['analysis.py', 'code', 'text/x-python'],
    ['analysis.ipynb', 'code', 'application/json'],
    ['analysis.do', 'code', 'text/plain'],
    ['analysis.sql', 'code', 'application/sql'],
    ['analysis.sh', 'code', 'text/x-shellscript'],
    ['package.lock', 'environment', 'text/plain'],
    ['environment.yaml', 'environment', 'application/yaml'],
    ['environment.yml', 'environment', 'application/yaml'],
    ['pyproject.toml', 'environment', 'application/toml'],
    ['Dockerfile', 'environment', 'text/plain'],
    ['requirements.txt', 'environment', 'text/plain'],
    ['requirements-test.txt', 'environment', 'text/plain'],
    ['renv.lock', 'environment', 'application/json'],
  ] as const)('declares %s as %s', (name, role, expected) => {
    expect(declaredMediaType(new File(['content'], name), role)).toBe(expected);
  });

  it.each([
    ['Makefile', 'code'],
    ['analysis.jl', 'code'],
    ['deps.lock', 'code'],
    ['Dockerfile.json', 'environment'],
    ['requirements.json', 'environment'],
    ['notes.txt', 'environment'],
    ['config.json', 'environment'],
  ] as const)('does not broaden the frozen filename policy to %s', (name, role) => {
    expect(declaredMediaType(new File(['content'], name), role)).toBe('application/octet-stream');
  });

  it('keeps the browser declaration when File.type is present', () => {
    expect(
      declaredMediaType(new File(['content'], 'analysis.py', { type: 'text/plain' }), 'code'),
    ).toBe('text/plain');
  });
});
