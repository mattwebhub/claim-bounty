import { Outlet } from 'react-router-dom';
import { LogoutButton } from '@/features/identity';
import { AppShell } from '@/shared/ui';

export function ApplicationLayout() {
  return (
    <AppShell
      brand={<span className="brand-mark">Peer2Paper</span>}
      brandIcon={
        <img
          className="brand-icon-image"
          src="/claimbounty-fox-loupe-icon.png"
          alt=""
          width={512}
          height={512}
        />
      }
      actions={
        <div className="header-actions">
          <LogoutButton />
        </div>
      }
    >
      <Outlet />
    </AppShell>
  );
}
