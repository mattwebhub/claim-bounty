import { Link } from 'react-router-dom';
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  ErrorState,
} from '@/shared/ui';
import { useProjects } from '../api/project.queries';
import type { Project } from '../model/project.schema';
import '../projects.css';
import { CreateProjectForm } from './create-project-form';
import { getProjectErrorDescription } from './project-error';

function ProjectCard({ project }: { project: Project }) {
  return (
    <article>
      <Card>
        <CardHeader>
          <p className="card-label">Project</p>
          <CardTitle>{project.name}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="project-updated">
            Updated{' '}
            <time dateTime={project.updatedAt.toISOString()}>
              {project.updatedAt.toLocaleString()}
            </time>
          </p>
          <Button asChild variant="secondary">
            <Link to={`/projects/${project.id}`}>View project</Link>
          </Button>
        </CardContent>
      </Card>
    </article>
  );
}

export function ProjectsPage() {
  const projectsQuery = useProjects();
  const projects = projectsQuery.data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <main id="main-content" className="projects-page" tabIndex={-1}>
      <header className="projects-heading">
        <div>
          <p className="eyebrow">Workspace</p>
          <h1>Projects</h1>
          <p>Create a project, then open its workspace to begin.</p>
        </div>
      </header>

      <section className="project-create" aria-labelledby="create-project-title">
        <div className="project-section-heading">
          <h2 id="create-project-title">New project</h2>
        </div>
        <CreateProjectForm />
      </section>

      <section aria-labelledby="project-list-title" aria-busy={projectsQuery.isPending}>
        <div className="project-section-heading">
          <h2 id="project-list-title">Your projects</h2>
        </div>

        {projectsQuery.isPending ? (
          <div className="project-list-skeleton" role="status" aria-live="polite">
            <span className="sr-only">Loading projects…</span>
            {Array.from({ length: 3 }, (_, index) => (
              <div className="project-skeleton" aria-hidden="true" key={index} />
            ))}
          </div>
        ) : projectsQuery.isError ? (
          <ErrorState
            title="Projects could not be loaded"
            description={getProjectErrorDescription(projectsQuery.error)}
            actionLabel="Try again"
            onAction={() => {
              void projectsQuery.refetch();
            }}
          />
        ) : projects.length === 0 ? (
          <EmptyState
            title="No projects yet"
            description="Create your first project with the form above."
          />
        ) : (
          <>
            <div className="project-grid">
              {projects.map((project) => (
                <ProjectCard project={project} key={project.id} />
              ))}
            </div>
            {projectsQuery.hasNextPage ? (
              <div className="project-load-more">
                <Button
                  variant="secondary"
                  disabled={projectsQuery.isFetchingNextPage}
                  onClick={() => {
                    void projectsQuery.fetchNextPage();
                  }}
                >
                  {projectsQuery.isFetchingNextPage ? 'Loading…' : 'Load more'}
                </Button>
              </div>
            ) : null}
          </>
        )}
      </section>
    </main>
  );
}
