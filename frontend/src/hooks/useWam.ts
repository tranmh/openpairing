'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { wamApi } from '@/lib/api';
import type {
  WamParticipant,
  WamTournamentStatus,
  GroupProposal,
  ConfirmGroupsResponse,
  U12Override,
  JokerValue,
} from '@/types/wam';

// Query keys for WAM data
export const wamKeys = {
  all: ['wam'] as const,
  status: (tournamentId: string) => [...wamKeys.all, 'status', tournamentId] as const,
  participants: (tournamentId: string) => [...wamKeys.all, 'participants', tournamentId] as const,
  proposal: (tournamentId: string) => [...wamKeys.all, 'proposal', tournamentId] as const,
};

/**
 * Hook for fetching WAM tournament status
 */
export function useWamStatus(tournamentId: string) {
  return useQuery({
    queryKey: wamKeys.status(tournamentId),
    queryFn: async () => {
      const response = await wamApi.getStatus(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data as WamTournamentStatus;
    },
    enabled: !!tournamentId,
  });
}

/**
 * Hook for fetching WAM participants with joker and U12 info
 */
export function useWamParticipants(tournamentId: string) {
  return useQuery({
    queryKey: wamKeys.participants(tournamentId),
    queryFn: async () => {
      const response = await wamApi.getParticipants(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data?.participants as WamParticipant[];
    },
    enabled: !!tournamentId,
  });
}

/**
 * Hook for fetching group proposal
 */
export function useGroupProposal(tournamentId: string, enabled = false) {
  return useQuery({
    queryKey: wamKeys.proposal(tournamentId),
    queryFn: async () => {
      const response = await wamApi.proposeGroups(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data as GroupProposal;
    },
    enabled: enabled && !!tournamentId,
    staleTime: 0, // Always refetch when requested
  });
}

/**
 * Hook for updating joker points
 */
export function useUpdateJoker(tournamentId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      participantId,
      jokerPoints,
    }: {
      participantId: string;
      jokerPoints: JokerValue;
    }) => {
      const response = await wamApi.setJokerPoints(tournamentId, participantId, jokerPoints);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
    onSuccess: () => {
      // Invalidate participants to refetch with updated data
      queryClient.invalidateQueries({ queryKey: wamKeys.participants(tournamentId) });
      // Also invalidate status as joker stats might have changed
      queryClient.invalidateQueries({ queryKey: wamKeys.status(tournamentId) });
    },
  });
}

/**
 * Hook for confirming group proposal
 */
export function useConfirmGroups(tournamentId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (u12Overrides: U12Override[]) => {
      const response = await wamApi.confirmGroups(tournamentId, {
        acceptProposal: true,
        u12Overrides: u12Overrides.length > 0 ? u12Overrides : undefined,
      });
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data as ConfirmGroupsResponse;
    },
    onSuccess: () => {
      // Invalidate all WAM data
      queryClient.invalidateQueries({ queryKey: wamKeys.status(tournamentId) });
      queryClient.invalidateQueries({ queryKey: wamKeys.participants(tournamentId) });
      // Remove stale proposal
      queryClient.removeQueries({ queryKey: wamKeys.proposal(tournamentId) });
    },
  });
}

/**
 * Hook for deleting all groups (reset)
 */
export function useDeleteGroups(tournamentId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const response = await wamApi.deleteGroups(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
    onSuccess: () => {
      // Invalidate all WAM data
      queryClient.invalidateQueries({ queryKey: wamKeys.status(tournamentId) });
    },
  });
}
