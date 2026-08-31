import { z } from 'zod';
import type { components } from '@/shared/api/generated/schema';

export type FileRole = components['schemas']['FileRole'];

export const fileRoles = [
  'primary_paper',
  'supplement',
  'preregistration',
  'data',
  'code',
  'environment',
  'data_dictionary',
  'other_evidence',
] as const satisfies readonly FileRole[];

export const fileRoleLabels: Record<FileRole, string> = {
  primary_paper: 'Primary paper',
  supplement: 'Supplement',
  preregistration: 'Preregistration',
  data: 'Data',
  code: 'Code',
  environment: 'Environment',
  data_dictionary: 'Data dictionary',
  other_evidence: 'Other evidence',
};

export const orderIntakeSchema = z.object({
  title: z.string().trim().min(1, 'Enter a short study title.').max(300),
  purpose: z.string().trim().min(1, 'Explain what you need checked.').max(1000),
  targetClaim: z.object({
    text: z.string().trim().min(1, 'Enter the exact claim to assess.').max(5000),
    sourceLocation: z.string().trim().min(1, 'Tell us where the claim appears.').max(1000),
  }),
  permissions: z.object({
    executeSuppliedCode: z.boolean(),
    externalSearch: z.boolean(),
  }),
  privacy: z.object({
    containsParticipantLevelData: z.boolean(),
    containsDirectIdentifiers: z.boolean(),
  }),
});

export const orderFileSchema = z.object({
  id: z.uuid(),
  role: z.enum(fileRoles),
  originalDisplayName: z.string().min(1).max(255),
  sizeBytes: z.number().int().positive().max(262_144_000),
  sha256: z.string().regex(/^[a-f0-9]{64}$/),
  storage: z.object({
    objectVersion: z.string().min(1).max(255),
    sha256: z.string().regex(/^[a-f0-9]{64}$/),
    immutability: z.literal('write_once'),
  }),
  declaredMediaType: z.string().min(1).max(255),
  detectedMediaType: z.string().max(255).nullable().optional(),
  status: z.enum(['upload_pending', 'uploaded', 'scanning', 'clean', 'rejected', 'expired']),
  rejectionCode: z.string().max(100).nullable().optional(),
  createdAt: z.iso.datetime({ offset: true }),
  updatedAt: z.iso.datetime({ offset: true }),
});

export const orderSchema = z.object({
  id: z.uuid(),
  publicReference: z.string().regex(/^CB-[A-Z0-9]{12}$/),
  status: z.enum([
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
  ]),
  version: z.number().int().positive(),
  title: z.string(),
  purpose: z.string(),
  targetClaim: z.object({ text: z.string(), sourceLocation: z.string().nullable().optional() }),
  permissions: z.object({
    executeSuppliedCode: z.boolean(),
    externalSearch: z.boolean(),
  }),
  privacy: z.object({
    containsParticipantLevelData: z.boolean(),
    containsDirectIdentifiers: z.boolean(),
  }),
  files: z.array(orderFileSchema).max(20),
  piiRetention: z.object({
    policyVersion: z.string(),
    disposition: z.literal('hard_delete'),
    sourceDeleteAfter: z.iso.datetime({ offset: true }),
    piiDeleteAfter: z.iso.datetime({ offset: true }),
  }),
  createdAt: z.iso.datetime({ offset: true }),
  updatedAt: z.iso.datetime({ offset: true }),
  submittedAt: z.iso.datetime({ offset: true }).nullable().optional(),
});

export type OrderIntakeValues = z.input<typeof orderIntakeSchema>;
export type OrderIntakeInput = z.output<typeof orderIntakeSchema>;
export type Order = z.output<typeof orderSchema>;
export type OrderFile = z.output<typeof orderFileSchema>;

export interface SelectedFile {
  file: File;
  id: string;
  idempotencyKey: string;
  progress: number;
  role: FileRole;
  serverFileId?: string;
  status: 'selected' | 'hashing' | 'uploading' | 'complete' | 'cancelled' | 'error';
}
