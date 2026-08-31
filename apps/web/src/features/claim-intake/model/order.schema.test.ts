import { describe, expect, it } from 'vitest';
import { orderIntakeSchema, orderSchema } from './order.schema';

describe('ClaimBounty order schemas', () => {
  it('trims and accepts a complete customer intake', () => {
    const result = orderIntakeSchema.parse({
      title: '  Study title  ',
      purpose: ' Check the reported effect. ',
      targetClaim: { text: ' The treatment improved retention. ', sourceLocation: ' p. 14 ' },
      permissions: { executeSuppliedCode: false, externalSearch: true },
      privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
    });
    expect(result.title).toBe('Study title');
    expect(result.targetClaim.sourceLocation).toBe('p. 14');
  });

  it('rejects a file response without immutable storage metadata', () => {
    const result = orderSchema.safeParse({
      id: '123e4567-e89b-12d3-a456-426614174000',
      publicReference: 'CB-ABCDEF123456',
      status: 'uploading',
      version: 2,
      title: 'Study',
      purpose: 'Review',
      targetClaim: { text: 'Claim', sourceLocation: 'Page 1' },
      permissions: { executeSuppliedCode: true, externalSearch: true },
      privacy: { containsParticipantLevelData: true, containsDirectIdentifiers: true },
      files: [
        {
          id: '223e4567-e89b-12d3-a456-426614174000',
          role: 'primary_paper',
          originalDisplayName: 'paper.pdf',
          sizeBytes: 200,
          sha256: 'a'.repeat(64),
          declaredMediaType: 'application/pdf',
          status: 'uploaded',
          createdAt: '2026-08-30T10:00:00Z',
          updatedAt: '2026-08-30T10:00:00Z',
        },
      ],
      piiRetention: {
        policyVersion: 'policy.1',
        disposition: 'hard_delete',
        sourceDeleteAfter: '2026-09-15T10:00:00Z',
        piiDeleteAfter: '2026-09-30T10:00:00Z',
      },
      createdAt: '2026-08-30T10:00:00Z',
      updatedAt: '2026-08-30T10:00:00Z',
    });
    expect(result.success).toBe(false);
  });
});
