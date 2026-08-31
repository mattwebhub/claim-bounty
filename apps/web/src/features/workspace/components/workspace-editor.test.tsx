import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mockServer } from '@/shared/test/mocks/server';
import { renderApplication } from '@/shared/test';
import { useWorkspaceDraftStore } from '../model/workspace-store';
import { WorkspaceEditor } from './workspace-editor';

const projectId = '00000000-0000-4000-8000-000000000001';
const workspace = {
  projectId,
  version: 1,
  createdAt: '2026-08-27T10:00:00.000Z',
  updatedAt: '2026-08-27T10:00:00.000Z',
  document: { schemaVersion: 1 as const, objects: [] },
};

describe('WorkspaceEditor', () => {
  beforeEach(() => {
    useWorkspaceDraftStore.getState().reset();
    mockServer.use(
      http.get(`http://127.0.0.1:8080/api/v1/projects/${projectId}/workspace`, () =>
        HttpResponse.json({ data: workspace }),
      ),
    );
  });

  it('keeps add, select, move, remove, and undo available without pointer gestures', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue(
      '00000000-0000-4000-8000-000000000010',
    );
    const user = userEvent.setup();
    renderApplication(<WorkspaceEditor projectId={projectId} />);

    await user.click(await screen.findByRole('button', { name: /Add Note/ }));
    expect(screen.getAllByRole('button', { name: 'Note 1, note' })).toHaveLength(1);
    await user.keyboard('{ArrowRight}');
    expect(useWorkspaceDraftStore.getState().document.objects[0]?.x).toBe(60);
    await user.click(screen.getByRole('button', { name: 'Remove object' }));
    expect(screen.queryByRole('button', { name: 'Note 1, note' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Undo' }));
    expect(screen.getByRole('button', { name: 'Note 1, note' })).toBeInTheDocument();
  });

  it('saves through React Query with the loaded version', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue(
      '00000000-0000-4000-8000-000000000010',
    );
    let ifMatch: string | null = null;
    mockServer.use(
      http.put(
        `http://127.0.0.1:8080/api/v1/projects/${projectId}/workspace`,
        async ({ request }) => {
          ifMatch = request.headers.get('if-match');
          const body = (await request.json()) as { document: typeof workspace.document };
          return HttpResponse.json({
            data: { ...workspace, version: 2, updatedAt: '2026-08-27T10:01:00.000Z', ...body },
          });
        },
      ),
    );
    const user = userEvent.setup();
    renderApplication(<WorkspaceEditor projectId={projectId} />);

    await user.click(await screen.findByRole('button', { name: /Add Card/ }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(ifMatch).toBe('"1"');
      expect(useWorkspaceDraftStore.getState().dirty).toBe(false);
      expect(useWorkspaceDraftStore.getState().baseVersion).toBe(2);
    });
  });

  it('keeps the draft available when a version conflict occurs', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue(
      '00000000-0000-4000-8000-000000000010',
    );
    mockServer.use(
      http.put(`http://127.0.0.1:8080/api/v1/projects/${projectId}/workspace`, () =>
        HttpResponse.json(
          { error: { code: 'version_conflict', message: 'Workspace changed elsewhere.' } },
          { status: 409 },
        ),
      ),
    );
    const user = userEvent.setup();
    renderApplication(<WorkspaceEditor projectId={projectId} />);

    await user.click(await screen.findByRole('button', { name: /Add Marker/ }));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Newer server version available')).toBeInTheDocument();
    expect(useWorkspaceDraftStore.getState().document.objects).toHaveLength(1);
    expect(useWorkspaceDraftStore.getState().dirty).toBe(true);
  });

  it('shows an offline draft state and prevents saving', async () => {
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false);
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue(
      '00000000-0000-4000-8000-000000000010',
    );
    const user = userEvent.setup();
    renderApplication(<WorkspaceEditor projectId={projectId} />);

    await user.click(await screen.findByRole('button', { name: /Add Note/ }));

    expect(screen.getByText('Waiting for connection')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();
  });
});
