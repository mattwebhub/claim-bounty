import { CircleDot, FileText, LayoutPanelTop } from 'lucide-react';
import { Button } from '@/shared/ui';
import type { WorkspaceObjectKind } from '../model/workspace';

const paletteItems = [
  { kind: 'note', label: 'Note', description: 'Capture a short idea.', Icon: FileText },
  {
    kind: 'card',
    label: 'Card',
    description: 'Group a piece of information.',
    Icon: LayoutPanelTop,
  },
  { kind: 'marker', label: 'Marker', description: 'Mark an important position.', Icon: CircleDot },
] satisfies readonly {
  kind: WorkspaceObjectKind;
  label: string;
  description: string;
  Icon: typeof FileText;
}[];

interface WorkspacePaletteProps {
  hidden?: boolean;
  onAdd: (kind: WorkspaceObjectKind) => void;
}

export function WorkspacePalette({ hidden = false, onAdd }: WorkspacePaletteProps) {
  return (
    <aside
      className="workspace-panel workspace-palette"
      aria-labelledby="palette-title"
      hidden={hidden}
      id="workspace-palette"
    >
      <div className="workspace-panel-heading">
        <p className="workspace-kicker">Create</p>
        <h2 id="palette-title">Object palette</h2>
        <p>Add an object, then position it on the work surface.</p>
      </div>
      <div className="palette-list">
        {paletteItems.map(({ description, Icon, kind, label }) => (
          <Button
            className="palette-item"
            key={kind}
            onClick={() => {
              onAdd(kind);
            }}
            variant="secondary"
          >
            <Icon aria-hidden="true" />
            <span>
              <strong>Add {label}</strong>
              <small>{description}</small>
            </span>
          </Button>
        ))}
      </div>
    </aside>
  );
}
