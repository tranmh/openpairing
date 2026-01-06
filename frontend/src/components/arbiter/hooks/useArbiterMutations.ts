'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { toast } from 'sonner';
import { arbiterStatusKey, pairingsKey } from './useArbiterStatus';
import type {
  GameResult,
  ByeType,
  AnnouncementPriority,
  CreateManualPairingResponse,
} from '@/types/arbiter';

/**
 * Helper to invalidate arbiter-related queries
 */
function useInvalidateArbiterQueries(tournamentId: string) {
  const queryClient = useQueryClient();

  return {
    invalidateStatus: () =>
      queryClient.invalidateQueries({ queryKey: arbiterStatusKey(tournamentId) }),
    invalidatePairings: (round?: string | number) => {
      if (round) {
        queryClient.invalidateQueries({ queryKey: pairingsKey(tournamentId, round) });
      } else {
        queryClient.invalidateQueries({ queryKey: ['pairings', tournamentId] });
      }
    },
    invalidateAll: () => {
      queryClient.invalidateQueries({ queryKey: arbiterStatusKey(tournamentId) });
      queryClient.invalidateQueries({ queryKey: ['pairings', tournamentId] });
    },
  };
}

/**
 * Parse API error from response
 */
function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === 'object' && error !== null) {
    const errorObj = error as Record<string, unknown>;
    const response = errorObj.response as Record<string, unknown> | undefined;
    const data = response?.data as Record<string, unknown> | undefined;
    if (typeof data?.error === 'string') return data.error;
    if (typeof errorObj.message === 'string') return errorObj.message;
  }
  return 'Ein Fehler ist aufgetreten';
}

// =============================================================================
// Result Management
// =============================================================================

export function useEnterResult(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({ gameId, result }: { gameId: string; result: string }) => {
      const response = await api.post(`/results`, {
        gameId,
        result,
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Ergebnis eingetragen');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useCorrectResult(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({
      gameId,
      newResult,
      reason,
    }: {
      gameId: string;
      newResult: GameResult;
      reason: string;
    }) => {
      const response = await api.post(`/arbiter/tournaments/${tournamentId}/correct-result`, {
        gameId,
        newResult,
        reason,
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Ergebnis korrigiert');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

// =============================================================================
// Player Management
// =============================================================================

export function useWithdrawPlayer(tournamentId: string) {
  const { invalidateStatus } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({
      participantId,
      reason,
    }: {
      participantId: string;
      reason?: string;
    }) => {
      const response = await api.post(`/arbiter/tournaments/${tournamentId}/withdraw-player`, {
        participantId,
        reason: reason || 'Abmeldung durch Schiedsrichter',
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateStatus();
      toast.success('Spieler zurückgezogen');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useReactivatePlayer(tournamentId: string) {
  const { invalidateStatus } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({
      participantId,
      missedRoundPoints,
      reason,
    }: {
      participantId: string;
      missedRoundPoints?: number;
      reason?: string;
    }) => {
      const response = await api.post(`/arbiter/tournaments/${tournamentId}/reactivate-player`, {
        participantId,
        missedRoundPoints,
        reason: reason || 'Reaktivierung durch Schiedsrichter',
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateStatus();
      toast.success('Spieler reaktiviert');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useAssignBye(tournamentId: string) {
  const { invalidateStatus } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({
      participantId,
      roundNumber,
      byeType,
    }: {
      participantId: string;
      roundNumber: number;
      byeType: ByeType;
    }) => {
      const response = await api.post(`/arbiter/tournaments/${tournamentId}/assign-bye`, {
        participantId,
        roundNumber,
        byeType,
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateStatus();
      toast.success('Spielfrei vergeben');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

// =============================================================================
// Pairing Management
// =============================================================================

export function useChangeColor(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({ gameId, reason }: { gameId: string; reason?: string }) => {
      const response = await api.post(`/arbiter/tournaments/${tournamentId}/change-color`, {
        gameId,
        reason: reason || 'Farbwechsel durch Schiedsrichter',
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Farben getauscht');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useDeletePairing(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({ gameId, reason }: { gameId: string; reason?: string }) => {
      const reasonParam = encodeURIComponent(reason || 'Paarung gelöscht durch Schiedsrichter');
      const response = await api.delete(
        `/arbiter/tournaments/${tournamentId}/pairings/${gameId}?reason=${reasonParam}`
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Paarung gelöscht');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useSwapPlayers(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({
      roundNumber,
      game1Id,
      player1Id,
      game2Id,
      player2Id,
      reason,
    }: {
      roundNumber: number;
      game1Id: string;
      player1Id: string;
      game2Id: string;
      player2Id: string;
      reason?: string;
    }) => {
      const response = await api.post(`/arbiter/tournaments/${tournamentId}/swap-pairing`, {
        roundNumber,
        game1Id,
        player1Id,
        game2Id,
        player2Id,
        reason: reason || 'Spielertausch durch Schiedsrichter',
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Spieler getauscht');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useCreateManualPairing(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async ({
      roundNumber,
      whiteParticipantId,
      blackParticipantId,
      boardNumber,
    }: {
      roundNumber: number;
      whiteParticipantId: string;
      blackParticipantId?: string | null;
      boardNumber?: number;
    }) => {
      const response = await api.post<CreateManualPairingResponse>(
        `/arbiter/tournaments/${tournamentId}/create-pairing`,
        {
          roundNumber,
          whiteParticipantId,
          blackParticipantId,
          boardNumber,
        }
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: (data) => {
      invalidateAll();
      // Show warnings if any
      if (data?.warnings?.length) {
        data.warnings.forEach((w) => {
          toast.warning(w.message);
        });
      }
      toast.success('Paarung erstellt');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

// =============================================================================
// Round Control
// =============================================================================

export function useGeneratePairings(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async (roundNumber: number) => {
      const response = await api.post(
        `/arbiter/tournaments/${tournamentId}/rounds/${roundNumber}/pair`
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Paarungen erstellt');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useStartRound(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async (roundNumber: number) => {
      const response = await api.post(
        `/arbiter/tournaments/${tournamentId}/rounds/${roundNumber}/start`
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Runde gestartet');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

export function useCompleteRound(tournamentId: string) {
  const { invalidateAll } = useInvalidateArbiterQueries(tournamentId);

  return useMutation({
    mutationFn: async (roundNumber: number) => {
      const response = await api.post(
        `/arbiter/tournaments/${tournamentId}/rounds/${roundNumber}/complete`
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      invalidateAll();
      toast.success('Runde beendet');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}

// =============================================================================
// Announcements
// =============================================================================

export function useCreateAnnouncement(tournamentId: string) {
  return useMutation({
    mutationFn: async ({
      title,
      message,
      priority,
    }: {
      title: string;
      message: string;
      priority: AnnouncementPriority;
    }) => {
      const response = await api.post(`/arbiter/tournaments/${tournamentId}/announcements`, {
        title,
        message,
        priority,
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      toast.success('Ankündigung erstellt');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });
}
