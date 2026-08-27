import { ArrowLeft, ExternalLink } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button, Card, CardContent, CardHeader, CardTitle, ErrorState } from '@/shared/ui';
import { useProject } from '../api/project.queries';
import '../projects.css';
import { getProjectErrorDescription } from './project-error';

export interface ProjectDetailsProps {
  projectId: string;
}

export function ProjectDetails({ projectId }: ProjectDetailsProps) {
  const projectQuery = useProject(projectId);

  if (projectQuery.isPending) {
    return (
      <main id="main-content" className="projects-page" aria-busy="true" aria-live="polite">
        <div className="project-detail-skeleton" aria-hidden="true" />
        <span className="sr-only">Loading project…</span>
      </main>
    );
  }

  if (projectQuery.isError) {
    return (
      <main id="main-content" className="projects-page">
        <ErrorState
          title="Project could not be loaded"
          description={getProjectErrorDescription(projectQuery.error)}
          actionLabel="Try again"
          onAction={() => {
            void projectQuery.refetch();
          }}
        />
      </main>
    );
  }

  const project = projectQuery.data;
  return (
    <main id="main-content" className="projects-page" tabIndex={-1}>
      <Button asChild variant="ghost">
        <Link to="/projects">
          <ArrowLeft aria-hidden="true" />
          Back to projects
        </Link>
      </Button>
      <Card className="project-detail-card">
        <CardHeader>
          <p className="eyebrow">Project</p>
          <CardTitle>{project.name}</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="project-metadata">
            <div>
              <dt>Created</dt>
              <dd>
                <time dateTime={project.createdAt.toISOString()}>
                  {project.createdAt.toLocaleString()}
                </time>
              </dd>
            </div>
            <div>
              <dt>Last updated</dt>
              <dd>
                <time dateTime={project.updatedAt.toISOString()}>
                  {project.updatedAt.toLocaleString()}
                </time>
              </dd>
            </div>
          </dl>
          <Button asChild>
            <Link to={`/workspace/${project.id}`}>
              Open workspace
              <ExternalLink aria-hidden="true" />
            </Link>
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
