import { AlertCircle, Cloud, CloudOff, LoaderCircle, TriangleAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/shared/ui/button';

export function PagePending() {
  const { t } = useTranslation('common');
  return (
    <main
      id="main-content"
      className="state-page"
      aria-busy="true"
      aria-live="polite"
      tabIndex={-1}
    >
      <LoaderCircle className="spinner" aria-hidden="true" />
      <span>{t('states.loading')}</span>
    </main>
  );
}

export interface EmptyStateProps {
  title: string;
  description: string;
  action?: React.ReactNode;
}

export function EmptyState({ action, description, title }: EmptyStateProps) {
  return (
    <section className="state-card" aria-labelledby="empty-state-title">
      <Cloud className="state-icon" aria-hidden="true" />
      <h1 id="empty-state-title">{title}</h1>
      <p>{description}</p>
      {action}
    </section>
  );
}

export interface ErrorStateProps {
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function ErrorState({ actionLabel, description, onAction, title }: ErrorStateProps) {
  return (
    <section className="state-card" role="alert" aria-labelledby="error-state-title">
      <AlertCircle className="state-icon state-icon-error" aria-hidden="true" />
      <h1 id="error-state-title">{title}</h1>
      <p>{description}</p>
      {actionLabel && onAction ? <Button onClick={onAction}>{actionLabel}</Button> : null}
    </section>
  );
}

export function OfflineBanner() {
  const { t } = useTranslation('common');
  return (
    <div className="offline-banner" role="status">
      <CloudOff aria-hidden="true" />
      <span>{t('states.offline')}</span>
    </div>
  );
}

export type SaveState = 'idle' | 'dirty' | 'saving' | 'saved' | 'error' | 'conflict' | 'offline';

export interface SaveStatusProps {
  state: SaveState;
  onRetry?: () => void;
}

export function SaveStatus({ onRetry, state }: SaveStatusProps) {
  const { t } = useTranslation('common');
  const isFailure = state === 'error' || state === 'conflict';

  return (
    <div
      className={`save-status save-status-${state}`}
      role={isFailure ? 'alert' : 'status'}
      aria-live={isFailure ? 'assertive' : 'polite'}
    >
      {isFailure ? (
        <TriangleAlert aria-hidden="true" />
      ) : state === 'saving' ? (
        <LoaderCircle className="spinner" aria-hidden="true" />
      ) : (
        <Cloud aria-hidden="true" />
      )}
      <span>{t(`states.save.${state}`)}</span>
      {state === 'error' && onRetry ? (
        <Button onClick={onRetry} size="sm" variant="ghost">
          {t('actions.retry')}
        </Button>
      ) : null}
    </div>
  );
}
