import { bench, describe, vi } from 'vitest';
import React from 'react';
import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('@/lib/api', () => ({
  api: {
    post: vi.fn(async () => ({ data: { ok: true, warnings: [] } })),
    delete: vi.fn(async () => ({ data: { ok: true } })),
  },
}));
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

import { useEnterResult, useWithdrawPlayer } from './useArbiterMutations';
import { arbiterStatusKey, pairingsKey } from './useArbiterStatus';

const TOURNAMENT = 't-1';

// Fill the cache with pairings of many rounds plus unrelated queries, so
// invalidation has to scan and refetch-mark a cache of growing size.
function makeClient(cachedQueries: number) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  client.setQueryData(arbiterStatusKey(TOURNAMENT), { status: 'ongoing' });
  for (let i = 0; i < cachedQueries; i++) {
    client.setQueryData(pairingsKey(TOURNAMENT, i + 1), { round: i + 1 });
    client.setQueryData(['players', 'search', `q${i}`], { players: [] });
  }
  return client;
}

function setup(cachedQueries: number) {
  const client = makeClient(cachedQueries);
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(
    () => ({
      enter: useEnterResult(TOURNAMENT),
      withdraw: useWithdrawPlayer(TOURNAMENT),
    }),
    { wrapper }
  );
  return result;
}

let sink: unknown;

for (const size of [10, 100, 1000]) {
  describe(`useArbiterMutations enterResult (invalidateAll), ${size} cached queries`, () => {
    const result = setup(size);
    bench(`enterResult x1, cache ${size}`, async () => {
      await act(async () => {
        sink = await result.current.enter.mutateAsync({ gameId: 'g1', result: '1-0' });
      });
    });
  });

  describe(`useArbiterMutations withdrawPlayer (invalidateStatus), ${size} cached queries`, () => {
    const result = setup(size);
    bench(`withdrawPlayer x1, cache ${size}`, async () => {
      await act(async () => {
        sink = await result.current.withdraw.mutateAsync({ participantId: 'p1' });
      });
    });
  });
}

describe('useArbiterMutations sequential results (100 cached queries)', () => {
  for (const n of [1, 10, 50]) {
    const result = setup(100);
    bench(`enterResult x${n}`, async () => {
      await act(async () => {
        for (let i = 0; i < n; i++) {
          sink = await result.current.enter.mutateAsync({ gameId: `g${i}`, result: '1-0' });
        }
      });
    });
  }
});

export { sink };
