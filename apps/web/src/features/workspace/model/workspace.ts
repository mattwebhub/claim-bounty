import { z } from 'zod';

export const workspaceObjectKinds = ['note', 'card', 'marker'] as const;

export const workspaceObjectSchema = z.object({
  id: z.string().min(1).max(100),
  kind: z
    .string()
    .max(50)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/),
  label: z.string().max(500),
  x: z.number().min(-10_000_000).max(10_000_000),
  y: z.number().min(-10_000_000).max(10_000_000),
  width: z.number().positive().max(1_000_000),
  height: z.number().positive().max(1_000_000),
});

export const workspaceDocumentSchema = z.object({
  schemaVersion: z.literal(1),
  objects: z.array(workspaceObjectSchema).max(1_000),
});

export const saveWorkspaceInputSchema = z.object({
  expectedVersion: z.number().int().positive(),
  document: workspaceDocumentSchema,
});

export type WorkspaceObjectKind = (typeof workspaceObjectKinds)[number];
export type WorkspaceObject = z.infer<typeof workspaceObjectSchema>;
export type WorkspaceDocument = z.infer<typeof workspaceDocumentSchema>;
export type SaveWorkspaceInput = z.infer<typeof saveWorkspaceInputSchema>;

export interface Workspace {
  projectId: string;
  document: WorkspaceDocument;
  version: number;
  createdAt: Date;
  updatedAt: Date;
}

export function cloneDocument(document: WorkspaceDocument): WorkspaceDocument {
  return { ...document, objects: document.objects.map((object) => ({ ...object })) };
}

export function documentsMatch(left: WorkspaceDocument, right: WorkspaceDocument) {
  return JSON.stringify(left) === JSON.stringify(right);
}
