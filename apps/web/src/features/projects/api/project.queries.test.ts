import { describe, expect, it } from 'vitest';
import { projectDetailOptions, projectKeys, projectListOptions } from './project.queries';

describe('project query options', () => {
  it('uses hierarchical, parameter-aware keys', () => {
    expect(projectKeys.lists()).toEqual(['projects', 'list']);
    expect(projectListOptions({ limit: 10 }).queryKey).toEqual(['projects', 'list', { limit: 10 }]);
    expect(projectDetailOptions('project-1').queryKey).toEqual(['projects', 'detail', 'project-1']);
  });

  it('disables a detail query without its route prerequisite', () => {
    expect(projectDetailOptions('').enabled).toBe(false);
  });
});
