import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { isApiError } from '@/shared/api';
import { Alert, AlertDescription, AlertTitle, Button } from '@/shared/ui';
import { useUpdateAdminIntake } from '../api/admin-order.queries';
import { adminIntakeSchema, type AdminOrder } from '../model/admin-order.schema';

interface IntakeDocumentValues {
  auditRequest: string;
  executionPolicy: string;
  routineContract: string;
  scientificPolicy: string;
}

interface AdminIntakeEditorProps {
  csrfToken: string;
  order: AdminOrder;
}

const documentNames = [
  ['auditRequest', 'Audit request'],
  ['scientificPolicy', 'Scientific policy'],
  ['executionPolicy', 'Execution policy'],
  ['routineContract', 'Validated routine contract'],
] as const;

function documentsFrom(order: AdminOrder): IntakeDocumentValues {
  return {
    auditRequest: JSON.stringify(order.frozenIntake?.auditRequest ?? {}, null, 2),
    scientificPolicy: JSON.stringify(order.frozenIntake?.scientificPolicy ?? {}, null, 2),
    executionPolicy: JSON.stringify(order.frozenIntake?.executionPolicy ?? {}, null, 2),
    routineContract: JSON.stringify(order.frozenIntake?.routineContract ?? {}, null, 2),
  };
}

export function AdminIntakeEditor({ csrfToken, order }: AdminIntakeEditorProps) {
  const mutation = useUpdateAdminIntake(order.id, csrfToken);
  const form = useForm<IntakeDocumentValues>({ defaultValues: documentsFrom(order) });

  useEffect(() => {
    form.reset(documentsFrom(order));
  }, [form, order]);

  useEffect(() => {
    if (mutation.isSuccess) {
      document.querySelector<HTMLElement>('#intake-save-status')?.focus();
    }
  }, [mutation.isSuccess]);

  const save = form.handleSubmit(async (values) => {
    const candidate: Record<string, unknown> = {};
    let invalidJson = false;
    for (const [name] of documentNames) {
      try {
        candidate[name] = JSON.parse(values[name]) as unknown;
      } catch {
        form.setError(name, { type: 'validate', message: 'Enter valid JSON.' });
        invalidJson = true;
      }
    }
    if (invalidJson) return;

    const parsed = adminIntakeSchema.safeParse(candidate);
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const name = issue.path[0];
        if (typeof name === 'string' && documentNames.some(([field]) => field === name)) {
          form.setError(name as keyof IntakeDocumentValues, {
            type: 'validate',
            message: `${issue.path.slice(1).join('.') || 'document'}: ${issue.message}`,
          });
        }
      }
      return;
    }
    try {
      await mutation.mutateAsync(parsed.data);
    } catch {
      // The mutation state renders the safe API error and leaves the form editable.
    }
  });

  return (
    <article className="detail-panel">
      <h2>Local handoff intake editor</h2>
      <p className="section-copy">
        Freeze the complete audit request and version-pinned policies. Missing scientific choices
        stay explicit; this editor does not infer models, variables, commands, or host paths.
      </p>
      <form className="form-stack" onSubmit={save} noValidate>
        {documentNames.map(([name, label]) => (
          <details key={name} open={name === 'auditRequest'}>
            <summary>{label}</summary>
            <label className="sr-only" htmlFor={`intake-${name}`}>
              {label} JSON
            </label>
            <textarea
              id={`intake-${name}`}
              className="json-editor"
              rows={14}
              spellCheck={false}
              aria-invalid={Boolean(form.formState.errors[name]) || undefined}
              aria-describedby={form.formState.errors[name] ? `intake-${name}-error` : undefined}
              {...form.register(name)}
            />
            {form.formState.errors[name] ? (
              <span id={`intake-${name}-error`} className="field-error" role="alert">
                {form.formState.errors[name].message}
              </span>
            ) : null}
          </details>
        ))}
        <Button type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? 'Validating and freezing…' : 'Validate and freeze intake'}
        </Button>
      </form>
      {mutation.isError ? (
        <Alert variant="destructive">
          <div>
            <AlertTitle>Intake could not be saved</AlertTitle>
            <AlertDescription>
              {isApiError(mutation.error)
                ? mutation.error.message
                : 'Check the documents and retry.'}
            </AlertDescription>
          </div>
        </Alert>
      ) : null}
      {mutation.isSuccess ? (
        <p id="intake-save-status" className="success-message" role="status" tabIndex={-1}>
          Intake frozen and readiness recalculated.
        </p>
      ) : null}
      {order.readinessIssues.length > 0 ? (
        <div className="readiness-issues">
          <h3>Readiness issues</h3>
          <ul>
            {order.readinessIssues.map((issue) => (
              <li key={`${issue.code}-${issue.path}`}>
                <strong>{issue.path}</strong>: {issue.message}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </article>
  );
}
