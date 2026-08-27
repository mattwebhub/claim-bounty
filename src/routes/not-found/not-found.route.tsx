import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, EmptyState } from '@/shared/ui';

export function Component() {
  const { t } = useTranslation('common');

  return (
    <main id="main-content" className="state-page" tabIndex={-1}>
      <EmptyState
        title={t('notFound.title')}
        description={t('notFound.description')}
        action={
          <Button asChild>
            <Link to="/">{t('actions.goHome')}</Link>
          </Button>
        }
      />
    </main>
  );
}
