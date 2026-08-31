import { z } from 'zod';
import type { components } from '@/shared/api/generated/schema';
import { orderSchema } from '@/features/claim-intake';

export type OrderStatus = components['schemas']['OrderStatus'];

export const orderStatuses = [
  'draft',
  'awaiting_email_verification',
  'uploading',
  'submitted',
  'scanning',
  'needs_information',
  'ready_for_export',
  'exported',
  'rejected',
  'cancelled',
  'expired',
] as const satisfies readonly OrderStatus[];

const hashSchema = z.string().regex(/^[a-f0-9]{64}$/);
const versionSchema = z.string().regex(/^[a-z0-9][a-z0-9._-]{0,99}$/);

export const routineContractSchema = z.object({
  routineId: z.literal('claim-bounty-operations/run-claimbounty-scientific-audit'),
  revision: z.string().regex(/^sha256:[a-f0-9]{64}$/),
  validation: z.object({
    status: z.literal('validated'),
    validatedAt: z.iso.datetime({ offset: true }),
    evidenceSha256: hashSchema,
  }),
});

export const auditRequestSchema = z.object({
  schemaVersion: z.literal('1.0.0'),
  caseId: z.uuid(),
  purpose: z.string().min(1),
  targetClaim: z.object({
    claimId: z.string().min(1),
    text: z.string().min(1),
    source: z.object({ artifact: z.string().min(1), location: z.string().min(1) }),
    status: z.literal('frozen'),
  }),
  permissions: z.object({
    readUploadedFiles: z.literal(true),
    executeSuppliedCode: z.boolean(),
    createDerivedFiles: z.boolean(),
    externalSearch: z.boolean(),
    openAccessSourcesOnly: z.boolean(),
    externalRedistributionAuthorized: z.literal(false),
  }),
  privacy: z.object({
    classification: z.enum(['public', 'restricted_research', 'confidential']),
    containsParticipantLevelData: z.boolean(),
    containsDirectIdentifiers: z.boolean(),
    redactRowLevelDataFromReports: z.boolean(),
  }),
  retention: z.object({
    policyVersion: versionSchema,
    sourceDeleteAfter: z.iso.datetime({ offset: true }),
    piiDeleteAfter: z.iso.datetime({ offset: true }),
    piiDisposition: z.literal('hard_delete'),
    preserveRunOutputs: z.boolean(),
  }),
  authority: z.object({
    uploadsAuthorized: z.literal(true),
    analysisUseAuthorized: z.literal(true),
    externalRedistributionAuthorized: z.literal(false),
    termsVersion: versionSchema,
    customerConfirmedAt: z.iso.datetime({ offset: true }),
    frozenBy: z.uuid(),
    frozenAt: z.iso.datetime({ offset: true }),
    authorizationPolicyVersion: versionSchema,
    adminAllowlistVersion: versionSchema,
  }),
  releaseScope: z.literal('internal'),
});

export const scientificPolicySchema = z.object({
  schemaVersion: z.literal('1.0.0'),
  policyVersion: versionSchema,
  defaultsVersion: versionSchema,
  targetFreeze: z.object({
    inferMissingScientificChoices: z.literal(false),
    ambiguity: z.enum(['block', 'preserve_conflict_and_continue_with_limits']),
  }),
  reproduction: z.object({
    comparisonProfile: z.string().min(1),
    scientificChangesCountAsExact: z.literal(false),
  }),
  sensitivity: z.object({
    maximumCandidates: z.number().int().positive(),
    resultsBlindReview: z.literal(true),
    reviewerCount: z.number().int().positive(),
  }),
  evidence: z.object({
    maximumQuestions: z.number().int().positive(),
    maximumDeepSources: z.number().int().positive(),
  }),
  verification: z.object({
    independentRerun: z.literal(true),
    maximumCorrectionRounds: z.number().int().nonnegative(),
  }),
});

export const executionPolicySchema = z.object({
  schemaVersion: z.literal('1.0.0'),
  policyVersion: versionSchema,
  runClass: z.literal('manual_local_operator'),
  releaseScope: z.literal('internal'),
  resources: z.object({
    maximumCpuCores: z.number().positive(),
    maximumMemoryMiB: z.number().int().positive(),
    maximumWorkingStorageMiB: z.number().int().positive(),
  }),
  sandbox: z.object({
    isolationRequired: z.literal(true),
    networkDuringAnalysis: z.literal('disabled'),
    dependencyAcquisition: z.enum(['disabled', 'operator_approved']),
    expandArchivesAutomatically: z.literal(false),
  }),
  sourceAccess: z.object({
    externalSearch: z.boolean(),
    openAccessOnly: z.boolean(),
    paywallBypass: z.literal(false),
  }),
  privacy: z.object({
    publishParticipantRows: z.literal(false),
    publishDirectIdentifiers: z.literal(false),
    reportsMayIncludeAggregateResults: z.boolean(),
  }),
  replay: z.object({
    requireCleanEnvironment: z.literal(true),
    requireInputAndOutputHashes: z.literal(true),
    requireDependencyVersions: z.literal(true),
  }),
});

export const adminIntakeSchema = z.object({
  auditRequest: auditRequestSchema,
  scientificPolicy: scientificPolicySchema,
  executionPolicy: executionPolicySchema,
  routineContract: routineContractSchema,
});

export const exportSchema = z.object({
  id: z.uuid(),
  orderId: z.uuid(),
  status: z.enum(['queued', 'building', 'ready', 'failed', 'expired']),
  routineContract: routineContractSchema,
  inputs: z
    .array(
      z.object({ fileId: z.uuid(), objectVersion: z.string().min(1).max(255), sha256: hashSchema }),
    )
    .min(1)
    .max(20),
  sha256: hashSchema.nullable().optional(),
  sizeBytes: z.number().int().positive().nullable().optional(),
  contentPath: z
    .string()
    .regex(/^\/api\/v1\/admin\/exports\/[0-9a-f-]+\/download$/)
    .nullable()
    .optional(),
  createdAt: z.iso.datetime({ offset: true }),
  completedAt: z.iso.datetime({ offset: true }).nullable().optional(),
  failureCode: z.string().max(100).nullable().optional(),
});

export const adminOrderSchema = orderSchema.extend({
  submitterEmail: z.email().nullable(),
  permissions: z.object({ executeSuppliedCode: z.boolean(), externalSearch: z.boolean() }),
  privacy: z.object({
    containsParticipantLevelData: z.boolean(),
    containsDirectIdentifiers: z.boolean(),
  }),
  frozenIntake: adminIntakeSchema.nullable().optional(),
  readinessIssues: z.array(z.object({ code: z.string(), path: z.string(), message: z.string() })),
  events: z.array(
    z.object({
      id: z.uuid(),
      actorKind: z.enum(['submitter', 'administrator', 'system']),
      actorId: z.string(),
      type: z.string(),
      metadata: z.record(z.string(), z.unknown()).optional(),
      createdAt: z.iso.datetime({ offset: true }),
    }),
  ),
  exports: z.array(exportSchema),
});

export const orderListSchema = z.object({
  items: z.array(orderSchema).max(100),
  nextCursor: z.string().max(1024).optional(),
});

export const exportReadinessSchema = z.object({
  metadataReviewed: z.boolean().refine(Boolean, 'Confirm the intake metadata review.'),
  filesReviewed: z.boolean().refine(Boolean, 'Confirm that all included files are clean.'),
  retentionPolicyVersion: versionSchema,
  preserveRunOutputs: z.boolean(),
});

export type AdminIntake = z.output<typeof adminIntakeSchema>;
export type AdminOrder = z.output<typeof adminOrderSchema>;
export type ExportRecord = z.output<typeof exportSchema>;
export type ExportReadinessValues = z.input<typeof exportReadinessSchema>;
export type ExportReadinessInput = z.output<typeof exportReadinessSchema>;
