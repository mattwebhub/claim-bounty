import { z } from 'zod';
import { apiRequest } from '@/shared/api';
import type { components } from '@/shared/api/generated/schema';
import {
  createProjectSchema,
  projectIdSchema,
  type CreateProjectInput,
  type Project,
  type ProjectPage,
} from '../model/project.schema';

type ProjectDto = components['schemas']['Project'];
type ProjectPageDto = components['schemas']['ProjectList'];
type CreateProjectRequestDto = components['schemas']['CreateProjectRequest'];

const projectDtoSchema: z.ZodType<ProjectDto> = z.object({
  id: projectIdSchema,
  name: z.string().min(1),
  createdAt: z.iso.datetime({ offset: true }),
  updatedAt: z.iso.datetime({ offset: true }),
});

const projectPageDtoSchema = z.object({
  items: z.array(projectDtoSchema),
  nextCursor: z.string().min(1).optional(),
});

function mapProject(dto: ProjectDto): Project {
  return {
    id: dto.id,
    name: dto.name,
    createdAt: new Date(dto.createdAt),
    updatedAt: new Date(dto.updatedAt),
  };
}

export interface ListProjectsParameters {
  cursor?: string;
  limit?: number;
}

export async function listProjects(
  parameters: ListProjectsParameters = {},
  signal?: AbortSignal,
): Promise<ProjectPage> {
  const search = new URLSearchParams();
  if (parameters.cursor) search.set('cursor', parameters.cursor);
  if (parameters.limit !== undefined) search.set('limit', String(parameters.limit));
  const query = search.size > 0 ? `?${search.toString()}` : '';
  const decoded = await apiRequest({
    path: `/projects${query}`,
    schema: projectPageDtoSchema,
    ...(signal ? { signal } : {}),
  });
  const page: ProjectPageDto = {
    items: decoded.items,
    ...(decoded.nextCursor === undefined ? {} : { nextCursor: decoded.nextCursor }),
  };

  return {
    items: page.items.map(mapProject),
    nextCursor: page.nextCursor ?? null,
  };
}

export async function getProject(projectId: string, signal?: AbortSignal): Promise<Project> {
  const dto = await apiRequest({
    path: `/projects/${encodeURIComponent(projectId)}`,
    schema: projectDtoSchema,
    ...(signal ? { signal } : {}),
  });
  return mapProject(dto);
}

export async function createProject(input: CreateProjectInput): Promise<Project> {
  const command: CreateProjectRequestDto = createProjectSchema.parse(input);
  const dto = await apiRequest({
    path: '/projects',
    method: 'POST',
    body: command,
    schema: projectDtoSchema,
  });
  return mapProject(dto);
}
