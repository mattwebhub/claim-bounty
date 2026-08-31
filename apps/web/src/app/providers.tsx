import { Suspense, useState, type PropsWithChildren } from 'react';
import { I18nextProvider } from 'react-i18next';
import { QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from 'sonner';
import { createQueryClient } from '@/app/query-client';
import { i18n } from '@/shared/i18n/i18n';
import { PagePending, ThemeProvider } from '@/shared/ui';

export function AppProviders({ children }: PropsWithChildren) {
  const [queryClient] = useState(createQueryClient);

  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <Suspense fallback={<PagePending />}>{children}</Suspense>
          <Toaster closeButton position="bottom-right" />
        </ThemeProvider>
      </QueryClientProvider>
    </I18nextProvider>
  );
}
