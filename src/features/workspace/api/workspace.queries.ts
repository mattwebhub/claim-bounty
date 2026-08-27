import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { SaveWorkspaceInput, Workspace } from '../model/workspace';
import { getWorkspace, saveWorkspace } from './workspace.service';

export const workspaceKeys = {
  all: ['workspaces'] as const,
  byProject: (projectId: string) => [...workspaceKeys.all, 'project', projectId] as const,
};

export function workspaceQueryOptions(projectId: string) {
  return queryOptions({
    queryKey: workspaceKeys.byProject(projectId),
    queryFn: ({ signal }) => getWorkspace(projectId, signal),
    staleTime: 30_000,
  });
}

export function useWorkspaceQuery(projectId: string) {
  return useQuery(workspaceQueryOptions(projectId));
}

interface SaveWorkspaceVariables {
  projectId: string;
  input: SaveWorkspaceInput;
}

export function useSaveWorkspace() {
  const queryClient = useQueryClient();

  return useMutation<Workspace, Error, SaveWorkspaceVariables>({
    mutationFn: ({ input, projectId }) => saveWorkspace(projectId, input),
    onSuccess: (workspace) => {
      queryClient.setQueryData(workspaceKeys.byProject(workspace.projectId), workspace);
    },
  });
}
