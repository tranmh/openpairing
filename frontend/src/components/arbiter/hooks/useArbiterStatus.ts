'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import type { TournamentStatus } from '@/types/arbiter';

/**
 * Query key for arbiter status
 */
export const arbiterStatusKey = (tournamentId: string) =>
  ['arbiter-status', tournamentId] as const;

/**
 * Hook for fetching and managing arbiter console status
 *
 * Features:
 * - Auto-refresh every 30 seconds
 * - Proper typing for TournamentStatus
 * - Easy refetch capability
 */
// API response wrapper type
interface ArbiterStatusResponse {
  status: TournamentStatus;
}

export function useArbiterStatus(tournamentId: string) {
  const { isHydrated, isAuthenticated } = useAuthStore();

  // Wait for auth hydration before making authenticated API calls
  const canFetch = isHydrated && isAuthenticated;

  const query = useQuery<TournamentStatus>({
    queryKey: arbiterStatusKey(tournamentId),
    queryFn: async (): Promise<TournamentStatus> => {
      const response = await api.get<ArbiterStatusResponse>(
        `/arbiter/tournaments/${tournamentId}/status`
      );
      if (response.error) {
        throw new Error(response.error.message);
      }
      // API returns { status: TournamentStatus }, extract the status object
      return response.data?.status as TournamentStatus;
    },
    enabled: canFetch,
    refetchInterval: canFetch ? 30000 : false, // Refresh every 30 seconds only when authenticated
    staleTime: 10000, // Consider data stale after 10 seconds
  });

  return {
    status: query.data,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    isRefetching: query.isRefetching,
  };
}

/**
 * Hook for fetching pairings for a specific round
 */
export const pairingsKey = (tournamentId: string, round: string | number) =>
  ['pairings', tournamentId, String(round)] as const;

export function usePairings(tournamentId: string, roundNumber: number | string) {
  const { isHydrated, isAuthenticated } = useAuthStore();
  const canFetch = isHydrated && isAuthenticated;

  return useQuery({
    queryKey: pairingsKey(tournamentId, roundNumber),
    queryFn: async () => {
      const response = await api.get<{ games: import('@/types/arbiter').PairingGame[] }>(
        `/tournaments/${tournamentId}/rounds/${roundNumber}/pairings`
      );
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
    enabled: canFetch && !!roundNumber,
  });
}
