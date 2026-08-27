import { useCallback, useEffect, useState } from 'react';
import { isApiError } from '@/shared/api';
import { Alert, AlertDescription, AlertTitle, Button, ErrorState, PagePending } from '@/shared/ui';
import { useSaveWorkspace, useWorkspaceQuery } from '../api/workspace.queries';
import { useOnlineStatus } from '../lib/use-online-status';
import {
  selectCanRedo,
  selectCanUndo,
  selectDirty,
  selectDocument,
  selectObjects,
  selectSelectedObject,
  selectSelectedObjectId,
  useWorkspaceDraftStore,
} from '../model/workspace-store';
import { WorkspaceInspector } from './workspace-inspector';
import { WorkspacePalette } from './workspace-palette';
import { WorkspaceSurface } from './workspace-surface';
import { WorkspaceToolbar } from './workspace-toolbar';
import './workspace.css';

const AUTO_SAVE_DELAY_MS = 1_200;
const narrowWorkspaceQuery = '(max-width: 62rem)';

interface WorkspaceEditorProps {
  projectId: string;
}

function isTypingTarget(target: EventTarget | null) {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
  );
}

function panelsOpenByDefault() {
  return (
    typeof window.matchMedia !== 'function' || !window.matchMedia(narrowWorkspaceQuery).matches
  );
}

export function WorkspaceEditor({ projectId }: WorkspaceEditorProps) {
  const workspaceQuery = useWorkspaceQuery(projectId);
  const saveMutation = useSaveWorkspace();
  const isOnline = useOnlineStatus();
  const [paletteOpen, setPaletteOpen] = useState(panelsOpenByDefault);
  const [inspectorOpen, setInspectorOpen] = useState(panelsOpenByDefault);
  const document = useWorkspaceDraftStore(selectDocument);
  const objects = useWorkspaceDraftStore(selectObjects);
  const selectedObject = useWorkspaceDraftStore(selectSelectedObject);
  const selectedObjectId = useWorkspaceDraftStore(selectSelectedObjectId);
  const dirty = useWorkspaceDraftStore(selectDirty);
  const canUndo = useWorkspaceDraftStore(selectCanUndo);
  const canRedo = useWorkspaceDraftStore(selectCanRedo);
  const initialize = useWorkspaceDraftStore((state) => state.initialize);
  const replaceFromServer = useWorkspaceDraftStore((state) => state.replaceFromServer);
  const addObject = useWorkspaceDraftStore((state) => state.addObject);
  const selectObject = useWorkspaceDraftStore((state) => state.selectObject);
  const moveSelection = useWorkspaceDraftStore((state) => state.moveSelection);
  const renameSelection = useWorkspaceDraftStore((state) => state.renameSelection);
  const removeSelection = useWorkspaceDraftStore((state) => state.removeSelection);
  const undo = useWorkspaceDraftStore((state) => state.undo);
  const redo = useWorkspaceDraftStore((state) => state.redo);
  const acknowledgeSave = useWorkspaceDraftStore((state) => state.acknowledgeSave);

  useEffect(() => {
    if (workspaceQuery.data) initialize(workspaceQuery.data);
  }, [initialize, workspaceQuery.data]);

  useEffect(
    () => () => {
      useWorkspaceDraftStore.getState().reset();
    },
    [],
  );

  const save = useCallback(() => {
    const draft = useWorkspaceDraftStore.getState();
    if (!draft.dirty || !draft.baseVersion || !isOnline || saveMutation.isPending) return;
    const submittedDocument = draft.document;
    saveMutation.mutate(
      {
        projectId,
        input: { expectedVersion: draft.baseVersion, document: submittedDocument },
      },
      {
        onSuccess: (workspace) => {
          acknowledgeSave(projectId, submittedDocument, workspace.version);
        },
      },
    );
  }, [acknowledgeSave, isOnline, projectId, saveMutation]);

  const hasConflict =
    saveMutation.isError &&
    isApiError(saveMutation.error) &&
    (saveMutation.error.status === 409 || saveMutation.error.code === 'version_conflict');

  useEffect(() => {
    if (!dirty || !isOnline || saveMutation.isPending || saveMutation.isError) return;
    const timeout = window.setTimeout(save, AUTO_SAVE_DELAY_MS);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [dirty, document, isOnline, save, saveMutation.isError, saveMutation.isPending]);

  useEffect(() => {
    const handleKeyboard = (event: KeyboardEvent) => {
      const modifier = event.metaKey || event.ctrlKey;
      if (modifier && event.key.toLowerCase() === 's') {
        event.preventDefault();
        save();
        return;
      }
      if (isTypingTarget(event.target)) return;
      if (modifier && event.key.toLowerCase() === 'z') {
        event.preventDefault();
        if (event.shiftKey) redo();
        else undo();
        return;
      }
      if (modifier && event.key.toLowerCase() === 'y') {
        event.preventDefault();
        redo();
        return;
      }
      const moves: Partial<Record<string, [number, number]>> = {
        ArrowUp: [0, -12],
        ArrowDown: [0, 12],
        ArrowLeft: [-12, 0],
        ArrowRight: [12, 0],
      };
      const movement = moves[event.key];
      if (movement && selectedObjectId) {
        event.preventDefault();
        moveSelection(...movement);
      } else if ((event.key === 'Delete' || event.key === 'Backspace') && selectedObjectId) {
        event.preventDefault();
        removeSelection();
      } else if (event.key === 'Escape') {
        selectObject(null);
      }
    };
    window.addEventListener('keydown', handleKeyboard);
    return () => {
      window.removeEventListener('keydown', handleKeyboard);
    };
  }, [moveSelection, redo, removeSelection, save, selectObject, selectedObjectId, undo]);

  if (workspaceQuery.isPending) return <PagePending />;
  if (workspaceQuery.isError) {
    const requestId = isApiError(workspaceQuery.error) ? workspaceQuery.error.requestId : undefined;
    return (
      <ErrorState
        actionLabel="Try again"
        description={`${workspaceQuery.error.message}${requestId ? ` Request ID: ${requestId}` : ''}`}
        onAction={() => {
          void workspaceQuery.refetch();
        }}
        title="Workspace unavailable"
      />
    );
  }

  const saveState =
    !isOnline && dirty
      ? 'offline'
      : hasConflict
        ? 'conflict'
        : saveMutation.isError
          ? 'error'
          : saveMutation.isPending
            ? 'saving'
            : dirty
              ? 'dirty'
              : 'saved';

  const discardLocalChanges = async () => {
    const result = await workspaceQuery.refetch();
    if (result.data) {
      replaceFromServer(result.data);
      saveMutation.reset();
    }
  };

  return (
    <div
      className="workspace-page"
      data-inspector-open={inspectorOpen}
      data-palette-open={paletteOpen}
    >
      <WorkspaceToolbar
        canRedo={canRedo}
        canSave={dirty && isOnline && !saveMutation.isPending && !hasConflict}
        canUndo={canUndo}
        inspectorOpen={inspectorOpen}
        onRedo={redo}
        onSave={save}
        onToggleInspector={() => {
          setInspectorOpen((open) => !open);
        }}
        onTogglePalette={() => {
          setPaletteOpen((open) => !open);
        }}
        onUndo={undo}
        paletteOpen={paletteOpen}
        saveState={saveState}
      />
      {hasConflict ? (
        <Alert className="workspace-conflict" variant="warning">
          <div>
            <AlertTitle>Newer server version available</AlertTitle>
            <AlertDescription>
              Your draft is still safe. Reload the server copy to discard it, or keep it open while
              you compare changes.
            </AlertDescription>
          </div>
          <Button
            onClick={() => {
              void discardLocalChanges();
            }}
            size="sm"
            variant="secondary"
          >
            Discard and reload
          </Button>
        </Alert>
      ) : null}
      <div className="workspace-layout">
        <WorkspacePalette hidden={!paletteOpen} onAdd={addObject} />
        <WorkspaceSurface
          objects={objects}
          onSelect={selectObject}
          selectedObjectId={selectedObjectId}
        />
        <WorkspaceInspector
          hidden={!inspectorOpen}
          object={selectedObject}
          objects={objects}
          onMove={moveSelection}
          onRemove={removeSelection}
          onRename={renameSelection}
          onSelect={selectObject}
        />
      </div>
    </div>
  );
}
