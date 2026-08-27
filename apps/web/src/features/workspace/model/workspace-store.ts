import { create } from 'zustand';
import type {
  Workspace,
  WorkspaceDocument,
  WorkspaceObject,
  WorkspaceObjectKind,
} from './workspace';
import { cloneDocument, documentsMatch } from './workspace';

const HISTORY_LIMIT = 50;
interface DraftSnapshot {
  document: WorkspaceDocument;
  selectedObjectId: string | null;
  dirty: boolean;
}

export interface WorkspaceDraftState {
  projectId: string | null;
  baseVersion: number | null;
  document: WorkspaceDocument;
  selectedObjectId: string | null;
  past: DraftSnapshot[];
  future: DraftSnapshot[];
  dirty: boolean;
}

export interface WorkspaceDraftActions {
  initialize: (workspace: Workspace) => void;
  replaceFromServer: (workspace: Workspace) => void;
  addObject: (kind: WorkspaceObjectKind) => void;
  selectObject: (id: string | null) => void;
  moveSelection: (deltaX: number, deltaY: number) => void;
  renameSelection: (label: string) => void;
  removeSelection: () => void;
  undo: () => void;
  redo: () => void;
  acknowledgeSave: (projectId: string, document: WorkspaceDocument, version: number) => void;
  reset: () => void;
}

export type WorkspaceDraftStore = WorkspaceDraftState & WorkspaceDraftActions;

const emptyDocument: WorkspaceDocument = { schemaVersion: 1, objects: [] };

function initialState(): WorkspaceDraftState {
  return {
    projectId: null,
    baseVersion: null,
    document: emptyDocument,
    selectedObjectId: null,
    past: [],
    future: [],
    dirty: false,
  };
}

function snapshot(state: WorkspaceDraftState): DraftSnapshot {
  return {
    document: cloneDocument(state.document),
    selectedObjectId: state.selectedObjectId,
    dirty: state.dirty,
  };
}

function recordMutation(
  state: WorkspaceDraftState,
  document: WorkspaceDocument,
  selectedObjectId = state.selectedObjectId,
): Partial<WorkspaceDraftState> {
  return {
    document,
    selectedObjectId,
    past: [...state.past, snapshot(state)].slice(-HISTORY_LIMIT),
    future: [],
    dirty: true,
  };
}

function makeObject(kind: WorkspaceObjectKind, index: number): WorkspaceObject {
  const labels: Record<WorkspaceObjectKind, string> = {
    note: 'Note',
    card: 'Card',
    marker: 'Marker',
  };
  const size = kind === 'marker' ? { width: 88, height: 88 } : { width: 176, height: 104 };

  return {
    id: globalThis.crypto.randomUUID(),
    kind,
    label: `${labels[kind]} ${index + 1}`,
    x: 48 + (index % 6) * 28,
    y: 48 + (index % 5) * 28,
    ...size,
  };
}

export const useWorkspaceDraftStore = create<WorkspaceDraftStore>()((set) => ({
  ...initialState(),
  initialize: (workspace) => {
    set((state) => {
      if (state.projectId === workspace.projectId) {
        if (state.baseVersion === workspace.version || state.dirty) return state;
      }
      return {
        ...state,
        projectId: workspace.projectId,
        baseVersion: workspace.version,
        document: cloneDocument(workspace.document),
        selectedObjectId: null,
        past: [],
        future: [],
        dirty: false,
      };
    });
  },
  replaceFromServer: (workspace) => {
    set((state) => ({
      ...state,
      projectId: workspace.projectId,
      baseVersion: workspace.version,
      document: cloneDocument(workspace.document),
      selectedObjectId: null,
      past: [],
      future: [],
      dirty: false,
    }));
  },
  addObject: (kind) => {
    set((state) => {
      const object = makeObject(kind, state.document.objects.length);
      return {
        ...state,
        ...recordMutation(
          state,
          { ...state.document, objects: [...state.document.objects, object] },
          object.id,
        ),
      };
    });
  },
  selectObject: (selectedObjectId) => {
    set({ selectedObjectId });
  },
  moveSelection: (deltaX, deltaY) => {
    set((state) => {
      if (!state.selectedObjectId) return state;
      const document = {
        ...state.document,
        objects: state.document.objects.map((object) =>
          object.id === state.selectedObjectId
            ? { ...object, x: object.x + deltaX, y: object.y + deltaY }
            : object,
        ),
      };
      return { ...state, ...recordMutation(state, document) };
    });
  },
  renameSelection: (label) => {
    set((state) => {
      if (!state.selectedObjectId) return state;
      const document = {
        ...state.document,
        objects: state.document.objects.map((object) =>
          object.id === state.selectedObjectId ? { ...object, label } : object,
        ),
      };
      return { ...state, ...recordMutation(state, document) };
    });
  },
  removeSelection: () => {
    set((state) => {
      if (!state.selectedObjectId) return state;
      const document = {
        ...state.document,
        objects: state.document.objects.filter((object) => object.id !== state.selectedObjectId),
      };
      return { ...state, ...recordMutation(state, document, null) };
    });
  },
  undo: () => {
    set((state) => {
      const previous = state.past.at(-1);
      if (!previous) return state;
      return {
        ...state,
        document: cloneDocument(previous.document),
        selectedObjectId: previous.selectedObjectId,
        past: state.past.slice(0, -1),
        future: [snapshot(state), ...state.future].slice(0, HISTORY_LIMIT),
        dirty: previous.dirty,
      };
    });
  },
  redo: () => {
    set((state) => {
      const next = state.future[0];
      if (!next) return state;
      return {
        ...state,
        document: cloneDocument(next.document),
        selectedObjectId: next.selectedObjectId,
        past: [...state.past, snapshot(state)].slice(-HISTORY_LIMIT),
        future: state.future.slice(1),
        dirty: next.dirty,
      };
    });
  },
  acknowledgeSave: (projectId, document, version) => {
    set((state) => {
      if (state.projectId !== projectId) return state;
      return {
        ...state,
        baseVersion: version,
        dirty: !documentsMatch(state.document, document),
        past: state.past.map((entry) => ({ ...entry, dirty: true })),
        future: state.future.map((entry) => ({ ...entry, dirty: true })),
      };
    });
  },
  reset: () => {
    set(initialState());
  },
}));

export const selectDocument = (state: WorkspaceDraftStore) => state.document;
export const selectObjects = (state: WorkspaceDraftStore) => state.document.objects;
export const selectSelectedObjectId = (state: WorkspaceDraftStore) => state.selectedObjectId;
export const selectSelectedObject = (state: WorkspaceDraftStore) =>
  state.document.objects.find((object) => object.id === state.selectedObjectId) ?? null;
export const selectCanUndo = (state: WorkspaceDraftStore) => state.past.length > 0;
export const selectCanRedo = (state: WorkspaceDraftStore) => state.future.length > 0;
export const selectDirty = (state: WorkspaceDraftStore) => state.dirty;
