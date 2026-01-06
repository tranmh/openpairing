import { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

/**
 * Creates a new QueryClient configured for testing.
 * Disables retries and caching to make tests more predictable.
 */
export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

/**
 * All providers wrapper for testing.
 */
function AllProviders({ children }: { children: React.ReactNode }) {
  const queryClient = createTestQueryClient();
  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

/**
 * Custom render function that wraps component with all necessary providers.
 * Use this instead of @testing-library/react's render in component tests.
 */
export function renderWithProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, { wrapper: AllProviders, ...options });
}

/**
 * Creates a wrapper with a specific QueryClient instance.
 * Useful when you need to control the QueryClient in tests.
 */
export function createWrapper(queryClient?: QueryClient) {
  const client = queryClient ?? createTestQueryClient();
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
  };
}

// Re-export everything from testing-library for convenience
export * from '@testing-library/react';
export { renderWithProviders as render };
