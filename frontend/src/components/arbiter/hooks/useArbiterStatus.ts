'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
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
export function useArbiterStatus(tournamentId: string) {
  const query = useQuery<TournamentStatus>({
    queryKey: arbiterStatusKey(tournamentId),
    queryFn: async (): Promise<TournamentStatus> => {
      const response = await api.get<TournamentStatus>(
        `/arbiter/tournaments/${tournamentId}/status`
      );
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data as TournamentStatus;
    },
    refetchInterval: 30000, // Refresh every 30 seconds
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
    enabled: !!roundNumber,
  });
}
