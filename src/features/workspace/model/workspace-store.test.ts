import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Workspace } from './workspace';
import { useWorkspaceDraftStore } from './workspace-store';

const firstWorkspace: Workspace = {
  projectId: '00000000-0000-4000-8000-000000000001',
  version: 1,
  createdAt: new Date('2026-08-27T10:00:00.000Z'),
  updatedAt: new Date('2026-08-27T10:00:00.000Z'),
  document: {
    schemaVersion: 1,
    objects: [
      {
        id: 'object-1',
        kind: 'note',
        label: 'First note',
        x: 10,
        y: 20,
        width: 176,
        height: 104,
      },
    ],
  },
};

describe('workspace draft store', () => {
  beforeEach(() => {
    useWorkspaceDraftStore.getState().reset();
  });

  it('initializes idempotently and does not overwrite dirty work on refetch', () => {
    const store = useWorkspaceDraftStore.getState();
    store.initialize(firstWorkspace);
    store.selectObject('object-1');
    store.moveSelection(12, 0);

    store.initialize({
      ...firstWorkspace,
      version: 2,
      document: { ...firstWorkspace.document, objects: [] },
    });

    const state = useWorkspaceDraftStore.getState();
    expect(state.document.objects[0]?.x).toBe(22);
    expect(state.baseVersion).toBe(1);
    expect(state.dirty).toBe(true);
  });

  it('supports bounded undo and redo without recording selection changes', () => {
    const randomUUID = vi.spyOn(globalThis.crypto, 'randomUUID');
    randomUUID.mockReturnValue('00000000-0000-4000-8000-000000000010');
    const store = useWorkspaceDraftStore.getState();
    store.initialize(firstWorkspace);
    store.addObject('card');
    for (let movement = 0; movement < 60; movement += 1) store.moveSelection(12, 0);

    expect(useWorkspaceDraftStore.getState().document.objects).toHaveLength(2);
    expect(useWorkspaceDraftStore.getState().past).toHaveLength(50);
    useWorkspaceDraftStore.getState().undo();
    expect(useWorkspaceDraftStore.getState().document.objects[1]?.x).toBe(784);
    useWorkspaceDraftStore.getState().redo();
    expect(useWorkspaceDraftStore.getState().document.objects).toHaveLength(2);
  });

  it('returns to a clean state when undo restores the loaded document', () => {
    const store = useWorkspaceDraftStore.getState();
    store.initialize(firstWorkspace);
    store.selectObject('object-1');
    store.moveSelection(12, 0);
    store.undo();

    expect(useWorkspaceDraftStore.getState().dirty).toBe(false);
  });

  it('acknowledges the saved version without clearing newer local edits', () => {
    const store = useWorkspaceDraftStore.getState();
    store.initialize(firstWorkspace);
    store.selectObject('object-1');
    store.moveSelection(12, 0);
    const submitted = useWorkspaceDraftStore.getState().document;
    store.moveSelection(12, 0);
    store.acknowledgeSave(firstWorkspace.projectId, submitted, 2);

    const state = useWorkspaceDraftStore.getState();
    expect(state.baseVersion).toBe(2);
    expect(state.dirty).toBe(true);
    expect(state.document.objects[0]?.x).toBe(34);
  });
});
