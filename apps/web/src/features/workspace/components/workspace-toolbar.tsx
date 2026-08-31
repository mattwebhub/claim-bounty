import { Button, SaveStatus, type SaveState } from '@/shared/ui';

interface WorkspaceToolbarProps {
  canRedo: boolean;
  canSave: boolean;
  canUndo: boolean;
  inspectorOpen: boolean;
  paletteOpen: boolean;
  saveState: SaveState;
  onRedo: () => void;
  onSave: () => void;
  onToggleInspector: () => void;
  onTogglePalette: () => void;
  onUndo: () => void;
}

export function WorkspaceToolbar({
  canRedo,
  canSave,
  canUndo,
  inspectorOpen,
  onRedo,
  onSave,
  onToggleInspector,
  onTogglePalette,
  onUndo,
  paletteOpen,
  saveState,
}: WorkspaceToolbarProps) {
  return (
    <header className="workspace-toolbar">
      <div className="workspace-toolbar-group">
        <Button
          aria-controls="workspace-palette"
          aria-expanded={paletteOpen}
          aria-label={paletteOpen ? 'Collapse object palette' : 'Expand object palette'}
          onClick={onTogglePalette}
          size="sm"
          variant="ghost"
        >
          {paletteOpen ? 'Hide palette' : 'Show palette'}
        </Button>
        <span className="workspace-title">Workspace</span>
      </div>
      <div className="workspace-toolbar-group workspace-history-controls">
        <Button aria-label="Undo" disabled={!canUndo} onClick={onUndo} size="sm" variant="ghost">
          Undo
        </Button>
        <Button aria-label="Redo" disabled={!canRedo} onClick={onRedo} size="sm" variant="ghost">
          Redo
        </Button>
      </div>
      <div className="workspace-toolbar-group workspace-save-controls">
        <SaveStatus {...(saveState === 'error' ? { onRetry: onSave } : {})} state={saveState} />
        <Button disabled={!canSave} onClick={onSave} size="sm">
          Save
        </Button>
        <Button
          aria-controls="workspace-inspector"
          aria-expanded={inspectorOpen}
          aria-label={inspectorOpen ? 'Collapse inspector' : 'Expand inspector'}
          onClick={onToggleInspector}
          size="sm"
          variant="ghost"
        >
          {inspectorOpen ? 'Hide inspector' : 'Show inspector'}
        </Button>
      </div>
    </header>
  );
}
