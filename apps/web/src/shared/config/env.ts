import { z } from 'zod';

const environmentSchema = z.object({
  VITE_APP_NAME: z.string().trim().min(1).default('React Frontend Template'),
  VITE_API_BASE_URL: z.url().default('http://localhost:8080/api/v1'),
  VITE_API_TIMEOUT_MS: z.coerce.number().int().positive().max(120_000).default(10_000),
});

export type AppEnvironment = z.infer<typeof environmentSchema>;

export function parseEnvironment(source: Record<string, unknown>): AppEnvironment {
  const result = environmentSchema.safeParse(source);

  if (!result.success) {
    const issues = result.error.issues
      .map((issue) => `${issue.path.join('.') || 'environment'}: ${issue.message}`)
      .join('; ');
    throw new Error(`Invalid public environment configuration: ${issues}`);
  }

  return result.data;
}

export const environment = parseEnvironment(import.meta.env);
