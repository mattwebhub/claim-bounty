import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { Alert, AlertDescription, AlertTitle, Button, Input } from '@/shared/ui';
import { useCreateProject } from '../api/project.queries';
import {
  createProjectSchema,
  type CreateProjectFormValues,
  type CreateProjectInput,
  type Project,
} from '../model/project.schema';
import { getProjectErrorDescription, getProjectFieldIssues } from './project-error';

export interface CreateProjectFormProps {
  onCreated?: (project: Project) => void;
}

export function CreateProjectForm({ onCreated }: CreateProjectFormProps) {
  const mutation = useCreateProject();
  const [submissionError, setSubmissionError] = useState<string | null>(null);
  const [createdAnnouncement, setCreatedAnnouncement] = useState('');
  const {
    formState: { errors },
    handleSubmit,
    register,
    reset,
    setError,
  } = useForm<CreateProjectFormValues, unknown, CreateProjectInput>({
    resolver: zodResolver(createProjectSchema),
    defaultValues: { name: '' },
  });

  const submit = handleSubmit(async (input) => {
    setSubmissionError(null);
    setCreatedAnnouncement('');

    try {
      const project = await mutation.mutateAsync(input);
      reset();
      setCreatedAnnouncement(`${project.name} was created.`);
      onCreated?.(project);
    } catch (error) {
      const nameIssue = getProjectFieldIssues(error).find(({ path }) => path === 'name');
      if (nameIssue) {
        setError('name', { message: nameIssue.message, type: 'server' }, { shouldFocus: true });
        return;
      }
      setSubmissionError(getProjectErrorDescription(error));
    }
  });

  return (
    <form className="project-create-form" onSubmit={submit} noValidate>
      <div className="project-field">
        <label htmlFor="project-name">Project name</label>
        <Input
          id="project-name"
          autoComplete="off"
          invalid={Boolean(errors.name)}
          aria-describedby={errors.name ? 'project-name-error' : 'project-name-help'}
          {...register('name')}
        />
        {errors.name ? (
          <span id="project-name-error" className="project-field-error" role="alert">
            {errors.name.message}
          </span>
        ) : (
          <span id="project-name-help" className="project-field-help">
            Use a short name your team will recognize.
          </span>
        )}
      </div>
      <Button type="submit" disabled={mutation.isPending}>
        {mutation.isPending ? 'Creating…' : 'Create project'}
      </Button>
      {submissionError ? (
        <Alert variant="destructive">
          <div>
            <AlertTitle>Project could not be created</AlertTitle>
            <AlertDescription>{submissionError}</AlertDescription>
          </div>
        </Alert>
      ) : null}
      <p className="sr-only" role="status" aria-live="polite">
        {createdAnnouncement}
      </p>
    </form>
  );
}
