import { z } from 'zod';
import { useParams } from 'react-router-dom';
import { AdminOrderDetail } from '@/features/admin-orders';
import { IdentityGate, useClearSession, useSession } from '@/features/identity';
import { isApiError } from '@/shared/api';
import { ErrorState, PagePending } from '@/shared/ui';

export function Component() {
  const { orderId } = useParams();
  const session = useSession();
  const clearSession = useClearSession();
  const sessionDenied =
    isApiError(session.error) && (session.error.status === 401 || session.error.status === 403);
  const parsedOrderId = z.uuid().safeParse(orderId);

  if (!parsedOrderId.success) {
    return (
      <main id="main-content" className="admin-page" tabIndex={-1}>
        <ErrorState
          title="Invalid order link"
          description="The order identifier in this link is not valid."
        />
      </main>
    );
  }

  if (session.isPending) return <PagePending />;

  return (
    <main id="main-content" className="admin-page" tabIndex={-1}>
      {!sessionDenied && session.data?.audience === 'administrator' ? (
        <AdminOrderDetail
          orderId={parsedOrderId.data}
          csrfToken={session.data.csrfToken}
          onAccessDenied={clearSession}
        />
      ) : (
        <div className="admin-auth-wrap">
          <IdentityGate audience="administrator" />
        </div>
      )}
    </main>
  );
}
