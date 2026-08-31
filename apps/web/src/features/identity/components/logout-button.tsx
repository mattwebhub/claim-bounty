import { useNavigate } from 'react-router-dom';
import { isApiError } from '@/shared/api';
import { Button } from '@/shared/ui';
import { useLogout, useSession } from '../api/identity.queries';

function describeLogoutError(error: unknown) {
  if (!isApiError(error)) return 'Sign out could not be completed. Try again.';
  return `${error.message}${error.requestId ? ` Request ID: ${error.requestId}.` : ''}`;
}

export function LogoutButton() {
  const session = useSession();
  const logout = useLogout();
  const navigate = useNavigate();

  if (!session.data) return null;

  return (
    <div className="logout-action">
      <Button
        type="button"
        variant="ghost"
        size="sm"
        disabled={logout.isPending}
        onClick={() => {
          logout.mutate(session.data.csrfToken, {
            onSuccess: () => {
              void navigate('/', { replace: true });
            },
          });
        }}
      >
        {logout.isPending ? 'Signing out…' : 'Sign out'}
      </Button>
      {logout.isError ? (
        <span className="logout-error field-error" role="alert">
          {describeLogoutError(logout.error)}
        </span>
      ) : null}
    </div>
  );
}
