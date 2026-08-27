import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { SaveStatus } from '@/shared/ui/states';
import { renderApplication } from '@/shared/test';

describe('SaveStatus', () => {
  it('announces a save error and exposes a keyboard-operable retry', async () => {
    const onRetry = vi.fn();
    const user = userEvent.setup();
    renderApplication(<SaveStatus state="error" onRetry={onRetry} />);

    const alert = screen.getByRole('alert');
    expect(alert).toHaveTextContent('Changes could not be saved');

    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('uses a polite status for non-failure states', () => {
    renderApplication(<SaveStatus state="saving" />);
    expect(screen.getByRole('status')).toHaveAttribute('aria-live', 'polite');
  });
});
