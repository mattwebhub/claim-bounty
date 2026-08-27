import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderOptions } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { i18n } from '@/shared/i18n/i18n';
import { ThemeProvider } from '@/shared/ui';

interface RenderApplicationOptions extends Omit<RenderOptions, 'wrapper'> {
  route?: string;
}

export function renderApplication(
  ui: React.ReactNode,
  { route = '/', ...options }: RenderApplicationOptions = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return {
    queryClient,
    ...render(ui, {
      wrapper: ({ children }) => (
        <I18nextProvider i18n={i18n}>
          <QueryClientProvider client={queryClient}>
            <ThemeProvider>
              <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>
            </ThemeProvider>
          </QueryClientProvider>
        </I18nextProvider>
      ),
      ...options,
    }),
  };
}
