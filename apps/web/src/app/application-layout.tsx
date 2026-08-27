import { Outlet } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { AppShell, ThemeToggle } from '@/shared/ui';

export function ApplicationLayout() {
  const { t } = useTranslation('common');

  return (
    <AppShell brand={<span className="brand-mark">{t('appName')}</span>} actions={<ThemeToggle />}>
      <Outlet />
    </AppShell>
  );
}
