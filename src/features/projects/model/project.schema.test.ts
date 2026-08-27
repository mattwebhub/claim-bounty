import { describe, expect, it } from 'vitest';
import { createProjectSchema, parseProjectId } from './project.schema';

describe('createProjectSchema', () => {
  it('normalizes a valid project name', () => {
    expect(createProjectSchema.parse({ name: '  Design system  ' })).toEqual({
      name: 'Design system',
    });
  });

  it.each([{ name: '   ' }, { name: 'bad\nname' }, { name: 'x'.repeat(121) }])(
    'rejects invalid project input %#',
    (input) => {
      expect(createProjectSchema.safeParse(input).success).toBe(false);
    },
  );
});

describe('parseProjectId', () => {
  it('keeps valid opaque identifiers and rejects malformed route input', () => {
    const id = '123e4567-e89b-12d3-a456-426614174000';
    expect(parseProjectId(id)).toBe(id);
    expect(parseProjectId('../projects')).toBeNull();
  });
});
