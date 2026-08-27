import { createBrowserRouter } from 'react-router-dom';
import { ApplicationLayout } from '@/app/application-layout';
import { RouteErrorBoundary } from '@/app/route-error-boundary';

export const router = createBrowserRouter([
  {
    path: '/',
    Component: ApplicationLayout,
    ErrorBoundary: RouteErrorBoundary,
    children: [
      {
        index: true,
        lazy: () => import('@/routes/home/home.route'),
      },
      {
        path: 'projects',
        lazy: () => import('@/routes/projects/projects.route'),
      },
      {
        path: 'projects/:projectId',
        lazy: () => import('@/routes/projects/project-detail.route'),
      },
      {
        path: 'workspace/:projectId',
        lazy: () => import('@/routes/workspace/workspace.route'),
      },
      {
        path: '*',
        lazy: () => import('@/routes/not-found/not-found.route'),
      },
    ],
  },
]);
