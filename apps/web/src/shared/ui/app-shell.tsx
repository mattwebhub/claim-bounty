import { type ReactNode } from 'react';

export interface AppShellProps {
  actions?: ReactNode;
  brand: ReactNode;
  brandIcon?: ReactNode;
  children: ReactNode;
}

export function AppShell({ actions, brand, brandIcon = 'C', children }: AppShellProps) {
  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="app-header-content">
          <a className="brand-link focus-ring" href="/" aria-label="Home">
            <span className="brand-icon" aria-hidden="true">
              {brandIcon}
            </span>
            {brand}
          </a>
          {actions ? <nav aria-label="Application actions">{actions}</nav> : null}
        </div>
      </header>
      <div className="app-content">{children}</div>
    </div>
  );
}
