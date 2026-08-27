import { z } from 'zod';

export const projectIdSchema = z.uuid();

export function parseProjectId(value: unknown): string | null {
  const result = projectIdSchema.safeParse(value);
  return result.success ? result.data : null;
}

const projectNameSchema = z
  .string()
  .trim()
  .min(1, 'Enter a project name.')
  .max(120, 'Project names must be 120 characters or fewer.')
  .refine(
    (name) =>
      Array.from(name).every((character) => {
        const codePoint = character.codePointAt(0);
        return codePoint !== undefined && codePoint > 31 && (codePoint < 127 || codePoint > 159);
      }),
    { message: 'Project names cannot contain control characters.' },
  );

export const createProjectSchema = z.object({
  name: projectNameSchema,
});

export type CreateProjectFormValues = z.input<typeof createProjectSchema>;
export type CreateProjectInput = z.output<typeof createProjectSchema>;

export interface Project {
  id: string;
  name: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface ProjectPage {
  items: Project[];
  nextCursor: string | null;
}
