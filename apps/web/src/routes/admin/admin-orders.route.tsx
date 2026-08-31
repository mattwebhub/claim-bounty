import { useSearchParams } from 'react-router-dom';
import { AdminOrdersList, orderStatuses, type OrderStatus } from '@/features/admin-orders';
import { IdentityGate, useClearSession, useSession } from '@/features/identity';
import { isApiError } from '@/shared/api';
import { PagePending } from '@/shared/ui';

function parseStatus(value: string | null): OrderStatus | undefined {
  return orderStatuses.find((status) => status === value);
}

export function Component() {
  const session = useSession();
  const clearSession = useClearSession();
  const [search, setSearch] = useSearchParams();
  const sessionDenied =
    isApiError(session.error) && (session.error.status === 401 || session.error.status === 403);
  const adminSession =
    !sessionDenied && session.data?.audience === 'administrator' ? session.data : null;
  const status = parseStatus(search.get('status'));
  const cursor = search.get('cursor') ?? undefined;

  if (session.isPending) return <PagePending />;

  return (
    <main id="main-content" className="admin-page" tabIndex={-1}>
      {adminSession ? (
        <AdminOrdersList
          {...(cursor ? { cursor } : {})}
          {...(status ? { status } : {})}
          onAccessDenied={clearSession}
          onFiltersChange={(filters) => {
            const next = new URLSearchParams();
            if (filters.status) next.set('status', filters.status);
            if (filters.cursor) next.set('cursor', filters.cursor);
            setSearch(next);
          }}
        />
      ) : (
        <div className="admin-auth-wrap">
          <IdentityGate audience="administrator" />
        </div>
      )}
    </main>
  );
}
