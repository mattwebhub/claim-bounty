import { isRouteErrorResponse, useRouteError } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { isApiError } from '@/shared/api';
import { ErrorState } from '@/shared/ui';

function getErrorDetails(error: unknown) {
  if (isRouteErrorResponse(error)) {
    return {
      title: `${error.status} ${error.statusText}`,
      description: typeof error.data === 'string' ? error.data : undefined,
    };
  }

  if (isApiError(error)) {
    return {
      title: error.message,
      description: error.requestId ? `Request ID: ${error.requestId}` : undefined,
    };
  }

  if (error instanceof Error) {
    return { title: error.message, description: undefined };
  }

  return { title: undefined, description: undefined };
}

export function RouteErrorBoundary() {
  const error = useRouteError();
  const { t } = useTranslation('common');
  const details = getErrorDetails(error);

  return (
    <main id="main-content" className="state-page" tabIndex={-1}>
      <ErrorState
        title={details.title ?? t('errors.unexpectedTitle')}
        description={details.description ?? t('errors.unexpectedDescription')}
        actionLabel={t('actions.reload')}
        onAction={() => {
          window.location.reload();
        }}
      />
    </main>
  );
}
