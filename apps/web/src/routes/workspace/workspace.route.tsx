import { Navigate, useParams } from 'react-router-dom';
import { parseProjectId } from '@/features/projects';
import { WorkspaceEditor } from '@/features/workspace';

export function Component() {
  const { projectId: routeProjectId } = useParams<{ projectId: string }>();
  const projectId = parseProjectId(routeProjectId);
  if (!projectId) return <Navigate replace to="/projects" />;
  return <WorkspaceEditor projectId={projectId} />;
}
