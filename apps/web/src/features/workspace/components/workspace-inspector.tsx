import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { Button, Input } from '@/shared/ui';
import type { WorkspaceObject } from '../model/workspace';

const inspectorSchema = z.object({ label: z.string().trim().min(1, 'Enter a label.').max(500) });
type InspectorFields = z.infer<typeof inspectorSchema>;

interface WorkspaceInspectorProps {
  hidden?: boolean;
  object: WorkspaceObject | null;
  objects: WorkspaceObject[];
  onMove: (deltaX: number, deltaY: number) => void;
  onRemove: () => void;
  onRename: (label: string) => void;
  onSelect: (id: string) => void;
}

function InspectorForm({ object, onRename }: Pick<WorkspaceInspectorProps, 'object' | 'onRename'>) {
  const {
    formState: { errors },
    handleSubmit,
    register,
  } = useForm<InspectorFields>({
    resolver: zodResolver(inspectorSchema),
    values: { label: object?.label ?? '' },
  });

  return (
    <form
      className="inspector-form"
      onSubmit={(event) => {
        void handleSubmit(({ label }) => {
          onRename(label);
        })(event);
      }}
    >
      <label htmlFor="object-label">Label</label>
      <Input
        aria-describedby={errors.label ? 'object-label-error' : undefined}
        aria-invalid={Boolean(errors.label)}
        id="object-label"
        {...register('label')}
      />
      {errors.label ? (
        <span className="field-error" id="object-label-error">
          {errors.label.message}
        </span>
      ) : null}
      <Button size="sm" type="submit" variant="secondary">
        Update label
      </Button>
    </form>
  );
}

export function WorkspaceInspector({
  hidden = false,
  object,
  objects,
  onMove,
  onRemove,
  onRename,
  onSelect,
}: WorkspaceInspectorProps) {
  return (
    <aside
      className="workspace-panel workspace-inspector"
      aria-labelledby="inspector-title"
      hidden={hidden}
      id="workspace-inspector"
    >
      <div className="workspace-panel-heading">
        <p className="workspace-kicker">Inspect</p>
        <h2 id="inspector-title">Objects</h2>
        <p>Select from this list or directly on the work surface.</p>
      </div>
      <ul className="object-list" aria-label="Workspace objects">
        {objects.map((item) => (
          <li key={item.id}>
            <button
              aria-current={item.id === object?.id ? 'true' : undefined}
              onClick={() => {
                onSelect(item.id);
              }}
              type="button"
            >
              <span>{item.label}</span>
              <small>{item.kind}</small>
            </button>
          </li>
        ))}
      </ul>
      {objects.length === 0 ? <p className="object-list-empty">No objects yet.</p> : null}
      {object ? (
        <div className="selection-inspector">
          <h3>Selected object</h3>
          <InspectorForm key={object.id} object={object} onRename={onRename} />
          <fieldset className="move-controls">
            <legend>Move object</legend>
            <Button
              aria-label="Move up"
              onClick={() => {
                onMove(0, -12);
              }}
              size="icon"
              variant="secondary"
            >
              <span aria-hidden="true">↑</span>
            </Button>
            <Button
              aria-label="Move left"
              onClick={() => {
                onMove(-12, 0);
              }}
              size="icon"
              variant="secondary"
            >
              <span aria-hidden="true">←</span>
            </Button>
            <Button
              aria-label="Move down"
              onClick={() => {
                onMove(0, 12);
              }}
              size="icon"
              variant="secondary"
            >
              <span aria-hidden="true">↓</span>
            </Button>
            <Button
              aria-label="Move right"
              onClick={() => {
                onMove(12, 0);
              }}
              size="icon"
              variant="secondary"
            >
              <span aria-hidden="true">→</span>
            </Button>
          </fieldset>
          <Button onClick={onRemove} size="sm" variant="destructive">
            Remove object
          </Button>
        </div>
      ) : (
        <p className="inspector-hint">Select an object to edit its label and position.</p>
      )}
    </aside>
  );
}
