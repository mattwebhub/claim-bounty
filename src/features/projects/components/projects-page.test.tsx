import { screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { mockServer, renderApplication } from '@/shared/test';
import { ProjectsPage } from './projects-page';

const projectDto = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Accessible workspace',
  createdAt: '2026-08-27T09:00:00Z',
  updatedAt: '2026-08-27T10:00:00Z',
};

describe('ProjectsPage', () => {
  it('renders an accessible loading state, then project links', async () => {
    mockServer.use(
      http.get('http://localhost:8080/api/v1/projects', () =>
        HttpResponse.json({ data: { items: [projectDto] } }),
      ),
    );

    const { unmount } = renderApplication(<ProjectsPage />);

    expect(screen.getByText('Loading projects…')).toBeInTheDocument();
    expect(await screen.findByRole('heading', { name: projectDto.name })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View project' })).toHaveAttribute(
      'href',
      `/projects/${projectDto.id}`,
    );
    unmount();
  });

  it('renders the empty state when the collection is empty', async () => {
    mockServer.use(
      http.get('http://localhost:8080/api/v1/projects', () =>
        HttpResponse.json({ data: { items: [] } }),
      ),
    );

    const { unmount } = renderApplication(<ProjectsPage />);

    expect(await screen.findByRole('heading', { name: 'No projects yet' })).toBeInTheDocument();
    unmount();
  });
});
