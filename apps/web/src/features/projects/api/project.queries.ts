import {
  infiniteQueryOptions,
  queryOptions,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import {
  createProject,
  getProject,
  listProjects,
  type ListProjectsParameters,
} from './project.service';

export interface ProjectListFilters {
  limit: number;
}

const defaultListFilters: ProjectListFilters = { limit: 20 };

export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (filters: ProjectListFilters) => [...projectKeys.lists(), filters] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (projectId: string) => [...projectKeys.details(), projectId] as const,
};

export function projectListOptions(filters: ProjectListFilters = defaultListFilters) {
  return infiniteQueryOptions({
    queryKey: projectKeys.list(filters),
    initialPageParam: null as string | null,
    queryFn: ({ pageParam, signal }) => {
      const parameters: ListProjectsParameters = { limit: filters.limit };
      if (pageParam !== null) parameters.cursor = pageParam;
      return listProjects(parameters, signal);
    },
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
  });
}

export function projectDetailOptions(projectId: string) {
  return queryOptions({
    queryKey: projectKeys.detail(projectId),
    queryFn: ({ signal }) => getProject(projectId, signal),
    enabled: projectId.length > 0,
  });
}

export function useProjects(filters: ProjectListFilters = defaultListFilters) {
  return useInfiniteQuery(projectListOptions(filters));
}

export function useProject(projectId: string) {
  return useQuery(projectDetailOptions(projectId));
}

export function useCreateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createProject,
    onSuccess: async (project) => {
      queryClient.setQueryData(projectKeys.detail(project.id), project);
      await queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}
