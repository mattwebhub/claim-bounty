import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { mockServer } from '@/shared/test';
import { createProject, getProject, listProjects } from './project.service';

const projectDto = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Example project',
  createdAt: '2026-08-27T09:00:00Z',
  updatedAt: '2026-08-27T10:00:00Z',
};

describe('project service', () => {
  it('selects the list endpoint and maps DTO timestamps into the feature model', async () => {
    mockServer.use(
      http.get('http://127.0.0.1:8080/api/v1/projects', ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get('cursor')).toBe('next page');
        expect(url.searchParams.get('limit')).toBe('10');
        return HttpResponse.json({ data: { items: [projectDto], nextCursor: 'after' } });
      }),
    );

    const page = await listProjects({ cursor: 'next page', limit: 10 });

    expect(page.nextCursor).toBe('after');
    expect(page.items[0]).toMatchObject({
      id: projectDto.id,
      name: projectDto.name,
      createdAt: new Date(projectDto.createdAt),
      updatedAt: new Date(projectDto.updatedAt),
    });
  });

  it('posts normalized create input and maps the created project', async () => {
    mockServer.use(
      http.post('http://127.0.0.1:8080/api/v1/projects', async ({ request }) => {
        expect(await request.json()).toEqual({ name: 'New project' });
        return HttpResponse.json({ data: { ...projectDto, name: 'New project' } }, { status: 201 });
      }),
    );

    await expect(createProject({ name: ' New project ' })).resolves.toMatchObject({
      id: projectDto.id,
      name: 'New project',
    });
  });

  it('runtime-rejects a project response with an invalid identifier', async () => {
    mockServer.use(
      http.get('http://127.0.0.1:8080/api/v1/projects/not-a-project', () =>
        HttpResponse.json({ data: { ...projectDto, id: 'not-a-uuid' } }),
      ),
    );

    await expect(getProject('not-a-project')).rejects.toMatchObject({ kind: 'decode' });
  });
});
