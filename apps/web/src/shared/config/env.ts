import { z } from 'zod';

const apiBaseUrlSchema = z
  .string()
  .trim()
  .refine(
    (value) => {
      if (value.startsWith('/') && !value.startsWith('//')) {
        try {
          const parsed = new URL(value, 'https://same-origin.invalid');
          return parsed.origin === 'https://same-origin.invalid' && !parsed.search && !parsed.hash;
        } catch {
          return false;
        }
      }

      try {
        const parsed = new URL(value);
        return (
          (parsed.protocol === 'http:' || parsed.protocol === 'https:') &&
          !parsed.search &&
          !parsed.hash
        );
      } catch {
        return false;
      }
    },
    { message: 'Must be a root-relative path or an HTTP(S) URL without a query or fragment' },
  );

const environmentSchema = z.object({
  VITE_APP_NAME: z.string().trim().min(1).default('Micro1 Template'),
  VITE_API_BASE_URL: apiBaseUrlSchema.default('/api/v1'),
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
