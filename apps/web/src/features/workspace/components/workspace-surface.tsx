import type { WorkspaceObject } from '../model/workspace';

interface WorkspaceSurfaceProps {
  objects: WorkspaceObject[];
  selectedObjectId: string | null;
  onSelect: (id: string | null) => void;
}

export function WorkspaceSurface({ objects, onSelect, selectedObjectId }: WorkspaceSurfaceProps) {
  return (
    <main id="main-content" className="workspace-surface" aria-label="Work surface" tabIndex={-1}>
      <div className="workspace-grid" aria-hidden="true" />
      {objects.length === 0 ? (
        <div className="workspace-empty">
          <h2>Your workspace is ready</h2>
          <p>Add an object from the palette. Every canvas action is also available by keyboard.</p>
        </div>
      ) : null}
      {objects.map((object) => {
        const isSelected = object.id === selectedObjectId;
        return (
          <button
            aria-label={`${object.label}, ${object.kind}`}
            aria-pressed={isSelected}
            className="workspace-object"
            data-kind={object.kind}
            data-selected={isSelected}
            key={object.id}
            onClick={() => {
              onSelect(object.id);
            }}
            style={{
              width: object.width,
              height: object.height,
              transform: `translate(${object.x}px, ${object.y}px)`,
            }}
            type="button"
          >
            <span className="workspace-object-kind">{object.kind}</span>
            <strong>{object.label}</strong>
          </button>
        );
      })}
    </main>
  );
}
