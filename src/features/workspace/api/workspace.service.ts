import { z } from 'zod';
import { apiRequest } from '@/shared/api';
import type { components } from '@/shared/api/generated/schema';
import {
  saveWorkspaceInputSchema,
  workspaceDocumentSchema,
  type SaveWorkspaceInput,
  type Workspace,
} from '../model/workspace';

type WorkspaceDto = components['schemas']['Workspace'];
type SaveWorkspaceRequestDto = components['schemas']['SaveWorkspaceRequest'];

const workspaceDtoSchema: z.ZodType<WorkspaceDto> = z.object({
  projectId: z.uuid(),
  document: workspaceDocumentSchema,
  version: z.number().int().positive(),
  createdAt: z.iso.datetime({ offset: true }),
  updatedAt: z.iso.datetime({ offset: true }),
});

function mapWorkspace(dto: WorkspaceDto): Workspace {
  return {
    ...dto,
    createdAt: new Date(dto.createdAt),
    updatedAt: new Date(dto.updatedAt),
  };
}

function workspacePath(projectId: string) {
  return `/projects/${encodeURIComponent(projectId)}/workspace`;
}

export async function getWorkspace(projectId: string, signal?: AbortSignal): Promise<Workspace> {
  const dto = await apiRequest({
    path: workspacePath(projectId),
    schema: workspaceDtoSchema,
    ...(signal ? { signal } : {}),
  });
  return mapWorkspace(dto);
}

export async function saveWorkspace(
  projectId: string,
  input: SaveWorkspaceInput,
  signal?: AbortSignal,
): Promise<Workspace> {
  const body = saveWorkspaceInputSchema.parse(input);
  const request: SaveWorkspaceRequestDto = { document: body.document };
  const dto = await apiRequest({
    path: workspacePath(projectId),
    method: 'PUT',
    schema: workspaceDtoSchema,
    body: request,
    headers: { 'If-Match': `"${body.expectedVersion}"` },
    ...(signal ? { signal } : {}),
  });
  return mapWorkspace(dto);
}
