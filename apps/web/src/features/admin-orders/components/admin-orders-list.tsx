import { Link } from 'react-router-dom';
import { Alert, AlertDescription, AlertTitle, Button, PagePending } from '@/shared/ui';
import { isApiError } from '@/shared/api';
import { useAdminAccessRevocation, useAdminOrders } from '../api/admin-order.queries';
import { orderStatuses, type OrderStatus } from '../model/admin-order.schema';

interface AdminOrdersListProps {
  cursor?: string;
  onAccessDenied: () => void;
  onFiltersChange: (filters: { cursor?: string; status?: OrderStatus }) => void;
  status?: OrderStatus;
}

function readableStatus(status: string) {
  return status.replaceAll('_', ' ');
}

export function AdminOrdersList({
  cursor,
  onAccessDenied,
  onFiltersChange,
  status,
}: AdminOrdersListProps) {
  const query = useAdminOrders({
    limit: 20,
    ...(cursor ? { cursor } : {}),
    ...(status ? { status } : {}),
  });
  const accessDenied = useAdminAccessRevocation(query.error, onAccessDenied);

  if (query.isPending) return <PagePending />;
  if (accessDenied) {
    return (
      <Alert variant="destructive">
        <div>
          <AlertTitle>Admin access was removed</AlertTitle>
          <AlertDescription>
            Sign in again after an administrator restores your access.
          </AlertDescription>
        </div>
      </Alert>
    );
  }
  if (query.isError) {
    return (
      <Alert variant="destructive">
        <div>
          <AlertTitle>Orders could not be loaded</AlertTitle>
          <AlertDescription>
            {isApiError(query.error)
              ? `${query.error.message}${query.error.requestId ? ` Request ID: ${query.error.requestId}.` : ''}`
              : 'Try signing in again.'}
          </AlertDescription>
          <Button variant="secondary" size="sm" onClick={() => void query.refetch()}>
            Retry
          </Button>
        </div>
      </Alert>
    );
  }

  return (
    <section aria-labelledby="orders-title">
      <div className="page-heading-row">
        <div>
          <p className="eyebrow">Private administration</p>
          <h1 id="orders-title">Research orders</h1>
          <p>Review intake and inspection state before preparing a local handoff.</p>
        </div>
        <label className="filter-field">
          Status
          <select
            value={status ?? ''}
            onChange={(event) => {
              onFiltersChange({
                ...(event.target.value ? { status: event.target.value as OrderStatus } : {}),
              });
            }}
          >
            <option value="">All statuses</option>
            {orderStatuses.map((value) => (
              <option key={value} value={value}>
                {readableStatus(value)}
              </option>
            ))}
          </select>
        </label>
      </div>
      {query.data.items.length === 0 ? (
        <div className="empty-panel">
          <h2>No orders match this view</h2>
          <p>Change the status filter or return after a submitter completes intake.</p>
        </div>
      ) : (
        <div className="orders-table-wrap">
          <table className="orders-table">
            <caption className="sr-only">ClaimBounty orders</caption>
            <thead>
              <tr>
                <th scope="col">Reference</th>
                <th scope="col">Study</th>
                <th scope="col">Status</th>
                <th scope="col">Submitted</th>
                <th scope="col">
                  <span className="sr-only">Action</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {query.data.items.map((order) => (
                <tr key={order.id}>
                  <td>
                    <strong>{order.publicReference}</strong>
                  </td>
                  <td>{order.title}</td>
                  <td>
                    <span className={`status-pill status-${order.status}`}>
                      {readableStatus(order.status)}
                    </span>
                  </td>
                  <td>
                    {order.submittedAt
                      ? new Date(order.submittedAt).toLocaleString()
                      : 'Not submitted'}
                  </td>
                  <td>
                    <Link to={`/admin/orders/${order.id}`}>Review</Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className="pagination-row">
        {cursor ? (
          <Button
            variant="secondary"
            onClick={() => {
              onFiltersChange({ ...(status ? { status } : {}) });
            }}
          >
            First page
          </Button>
        ) : null}
        {query.data.nextCursor ? (
          <Button
            variant="secondary"
            onClick={() => {
              onFiltersChange({
                ...(query.data.nextCursor ? { cursor: query.data.nextCursor } : {}),
                ...(status ? { status } : {}),
              });
            }}
          >
            Next page
          </Button>
        ) : null}
      </div>
    </section>
  );
}
