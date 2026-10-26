import { bench, describe, vi } from 'vitest';
import React from 'react';
import { act, fireEvent, render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: () => {}, replace: () => {}, refresh: () => {} }),
}));
vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(async () => ({ data: { players: [] } })),
    post: vi.fn(async () => ({ data: { ok: true } })),
  },
  federationApi: { search: vi.fn(async () => ({ data: { results: [] } })) },
}));
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

import { RegistrationForm } from './RegistrationForm';

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

function renderForms(count: number) {
  const client = makeClient();
  return render(
    <QueryClientProvider client={client}>
      {Array.from({ length: count }, (_, i) => (
        <RegistrationForm key={i} tournamentId={`t-${i}`} />
      ))}
    </QueryClientProvider>
  );
}

let sink: unknown;

describe('RegistrationForm render + unmount', () => {
  for (const n of [1, 10, 50]) {
    bench(`${n} forms`, () => {
      const view = renderForms(n);
      sink = view.container.childElementCount;
      view.unmount();
    });
  }
});

describe('RegistrationForm submit (zod validation path)', () => {
  for (const n of [1, 10, 50]) {
    // Empty form: refine() fails, so handleSubmit runs validation and
    // re-renders the error message for every submit.
    const view = renderForms(1);
    const form = view.container.querySelector('form')!;
    const ratingInput = view.container.querySelector('input[type="number"]')!;
    bench(`${n} submits`, async () => {
      for (let i = 0; i < n; i++) {
        await act(async () => {
          fireEvent.change(ratingInput, { target: { value: String(1500 + i) } });
          fireEvent.submit(form);
        });
      }
      sink = view.container.textContent?.length;
    });
  }
});

export { sink };
