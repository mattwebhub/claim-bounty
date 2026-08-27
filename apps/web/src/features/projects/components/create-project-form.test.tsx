import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';
import { mockServer, renderApplication } from '@/shared/test';
import { CreateProjectForm } from './create-project-form';

const projectDto = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Team workspace',
  createdAt: '2026-08-27T09:00:00Z',
  updatedAt: '2026-08-27T09:00:00Z',
};

describe('CreateProjectForm', () => {
  it('validates locally and focuses the invalid field', async () => {
    const user = userEvent.setup();
    const { unmount } = renderApplication(<CreateProjectForm />);

    await user.click(screen.getByRole('button', { name: 'Create project' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Enter a project name.');
    expect(screen.getByLabelText('Project name')).toHaveFocus();
    unmount();
  });

  it('maps API validation details to the owned form field', async () => {
    mockServer.use(
      http.post('http://localhost:8080/api/v1/projects', () =>
        HttpResponse.json(
          {
            error: {
              code: 'validation_failed',
              message: 'The request is invalid.',
              details: [{ path: 'name', code: 'already_exists', message: 'Use another name.' }],
            },
          },
          { status: 422 },
        ),
      ),
    );
    const user = userEvent.setup();
    const { unmount } = renderApplication(<CreateProjectForm />);

    await user.type(screen.getByLabelText('Project name'), 'Existing');
    await user.click(screen.getByRole('button', { name: 'Create project' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Use another name.');
    expect(screen.getByLabelText('Project name')).toHaveFocus();
    unmount();
  });

  it('announces success and returns the mapped project to its owner', async () => {
    mockServer.use(
      http.post('http://localhost:8080/api/v1/projects', () =>
        HttpResponse.json({ data: projectDto }, { status: 201 }),
      ),
    );
    const onCreated = vi.fn();
    const user = userEvent.setup();
    const { unmount } = renderApplication(<CreateProjectForm onCreated={onCreated} />);

    await user.type(screen.getByLabelText('Project name'), projectDto.name);
    await user.click(screen.getByRole('button', { name: 'Create project' }));

    expect(await screen.findByRole('status')).toHaveTextContent(`${projectDto.name} was created.`);
    expect(onCreated).toHaveBeenCalledWith(
      expect.objectContaining({ id: projectDto.id, name: projectDto.name }),
    );
    expect(screen.getByLabelText('Project name')).toHaveValue('');
    unmount();
  });
});
