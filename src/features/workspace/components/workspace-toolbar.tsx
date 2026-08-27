import { ChevronLeft, ChevronRight, PanelLeft, PanelRight, Redo2, Save, Undo2 } from 'lucide-react';
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
          size="icon"
          variant="ghost"
        >
          {paletteOpen ? <ChevronLeft aria-hidden="true" /> : <PanelLeft aria-hidden="true" />}
        </Button>
        <span className="workspace-title">Workspace</span>
      </div>
      <div className="workspace-toolbar-group workspace-history-controls">
        <Button aria-label="Undo" disabled={!canUndo} onClick={onUndo} size="icon" variant="ghost">
          <Undo2 aria-hidden="true" />
        </Button>
        <Button aria-label="Redo" disabled={!canRedo} onClick={onRedo} size="icon" variant="ghost">
          <Redo2 aria-hidden="true" />
        </Button>
      </div>
      <div className="workspace-toolbar-group workspace-save-controls">
        <SaveStatus {...(saveState === 'error' ? { onRetry: onSave } : {})} state={saveState} />
        <Button disabled={!canSave} onClick={onSave} size="sm">
          <Save aria-hidden="true" /> Save
        </Button>
        <Button
          aria-controls="workspace-inspector"
          aria-expanded={inspectorOpen}
          aria-label={inspectorOpen ? 'Collapse inspector' : 'Expand inspector'}
          onClick={onToggleInspector}
          size="icon"
          variant="ghost"
        >
          {inspectorOpen ? <ChevronRight aria-hidden="true" /> : <PanelRight aria-hidden="true" />}
        </Button>
      </div>
    </header>
  );
}
