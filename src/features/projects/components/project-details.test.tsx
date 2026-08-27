import { screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { mockServer, renderApplication } from '@/shared/test';
import { ProjectDetails } from './project-details';

const projectDto = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Project detail',
  createdAt: '2026-08-27T09:00:00Z',
  updatedAt: '2026-08-27T10:00:00Z',
};

describe('ProjectDetails', () => {
  it('renders project metadata and the workspace destination', async () => {
    mockServer.use(
      http.get(`http://localhost:8080/api/v1/projects/${projectDto.id}`, () =>
        HttpResponse.json({ data: projectDto }),
      ),
    );

    renderApplication(<ProjectDetails projectId={projectDto.id} />);

    expect(await screen.findByRole('heading', { name: projectDto.name })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /open workspace/i })).toHaveAttribute(
      'href',
      `/workspace/${projectDto.id}`,
    );
  });
});
