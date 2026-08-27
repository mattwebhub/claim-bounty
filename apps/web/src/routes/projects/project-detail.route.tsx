import { Navigate, useParams } from 'react-router-dom';
import { parseProjectId, ProjectDetails } from '@/features/projects';

export function Component() {
  const { projectId: routeProjectId } = useParams<{ projectId: string }>();
  const projectId = parseProjectId(routeProjectId);
  if (!projectId) return <Navigate to="/projects" replace />;
  return <ProjectDetails projectId={projectId} />;
}
