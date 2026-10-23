import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}));

import { api } from '@/lib/api';
import { toast } from 'sonner';
import {
  useEnterResult,
  useCorrectResult,
  useWithdrawPlayer,
  useReactivatePlayer,
  useAssignBye,
  useChangeColor,
  useDeletePairing,
  useSwapPlayers,
  useCreateManualPairing,
  useGeneratePairings,
  useStartRound,
  useCompleteRound,
  useCreateAnnouncement,
} from './useArbiterMutations';

const mockPost = api.post as unknown as ReturnType<typeof vi.fn>;
const mockDelete = api.delete as unknown as ReturnType<typeof vi.fn>;
const mockToast = toast as unknown as {
  success: ReturnType<typeof vi.fn>;
  error: ReturnType<typeof vi.fn>;
  warning: ReturnType<typeof vi.fn>;
};

const T = 'tour-1';
const STATUS_KEY = ['arbiter-status', T];
const PAIRINGS_KEY = ['pairings', T];

function setup<R>(hook: () => R) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const rendered = renderHook(hook, { wrapper });
  return { ...rendered, invalidateSpy, queryClient };
}

function invalidatedKeys(spy: ReturnType<typeof vi.spyOn>) {
  return spy.mock.calls.map((c) => (c[0] as { queryKey: unknown[] }).queryKey);
}

const ok = (data: unknown = { ok: true }) => ({ data, status: 200 });
const fail = (message: string, status = 400) => ({ error: { message }, status });

describe('useArbiterMutations', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('useEnterResult', () => {
    it('posts the result, invalidates status and pairings and shows a success toast', async () => {
      mockPost.mockResolvedValue(ok({ id: 'r1' }));
      const { result, invalidateSpy } = setup(() => useEnterResult(T));

      let data: unknown;
      await act(async () => {
        data = await result.current.mutateAsync({ gameId: 'g1', result: '1-0' });
      });

      expect(data).toEqual({ id: 'r1' });
      expect(mockPost).toHaveBeenCalledWith('/results', { gameId: 'g1', result: '1-0' });
      expect(mockToast.success).toHaveBeenCalledWith('Ergebnis eingetragen');
      expect(mockToast.error).not.toHaveBeenCalled();
      const keys = invalidatedKeys(invalidateSpy);
      expect(keys).toContainEqual(STATUS_KEY);
      expect(keys).toContainEqual(PAIRINGS_KEY);
    });

    it('turns an API error response into a failed mutation with an error toast', async () => {
      mockPost.mockResolvedValue(fail('Spiel bereits beendet'));
      const { result, invalidateSpy } = setup(() => useEnterResult(T));

      act(() => {
        result.current.mutate({ gameId: 'g1', result: '1-0' });
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(result.current.error?.message).toBe('Spiel bereits beendet');
      expect(mockToast.error).toHaveBeenCalledWith('Spiel bereits beendet');
      expect(mockToast.success).not.toHaveBeenCalled();
      expect(invalidateSpy).not.toHaveBeenCalled();
    });

    it('uses the server error text from a rejected response object', async () => {
      mockPost.mockRejectedValue({ response: { data: { error: 'Nicht berechtigt' } } });
      const { result } = setup(() => useEnterResult(T));

      act(() => {
        result.current.mutate({ gameId: 'g1', result: '0-1' });
      });

      await waitFor(() => expect(mockToast.error).toHaveBeenCalledWith('Nicht berechtigt'));
    });

    it('uses the message field of a rejected plain object', async () => {
      mockPost.mockRejectedValue({ message: 'Timeout' });
      const { result } = setup(() => useEnterResult(T));

      act(() => {
        result.current.mutate({ gameId: 'g1', result: '0-1' });
      });

      await waitFor(() => expect(mockToast.error).toHaveBeenCalledWith('Timeout'));
    });

    it('falls back to a generic message for unknown rejection values', async () => {
      mockPost.mockRejectedValue('boom');
      const { result } = setup(() => useEnterResult(T));

      act(() => {
        result.current.mutate({ gameId: 'g1', result: '0-1' });
      });

      await waitFor(() =>
        expect(mockToast.error).toHaveBeenCalledWith('Ein Fehler ist aufgetreten')
      );
    });
  });

  describe('useCorrectResult', () => {
    it('posts the correction to the tournament endpoint', async () => {
      mockPost.mockResolvedValue(ok());
      const { result, invalidateSpy } = setup(() => useCorrectResult(T));

      await act(async () => {
        await result.current.mutateAsync({ gameId: 'g1', newResult: '0-1', reason: 'Tippfehler' });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/correct-result`, {
        gameId: 'g1',
        newResult: '0-1',
        reason: 'Tippfehler',
      });
      expect(mockToast.success).toHaveBeenCalledWith('Ergebnis korrigiert');
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([STATUS_KEY, PAIRINGS_KEY])
      );
    });

    it('reports errors', async () => {
      mockPost.mockResolvedValue(fail('Runde abgeschlossen'));
      const { result } = setup(() => useCorrectResult(T));

      await act(async () => {
        await expect(
          result.current.mutateAsync({ gameId: 'g1', newResult: '1-0', reason: 'x' })
        ).rejects.toThrow('Runde abgeschlossen');
      });
      expect(mockToast.error).toHaveBeenCalledWith('Runde abgeschlossen');
    });
  });

  describe('useWithdrawPlayer', () => {
    it('uses a default reason and only invalidates the status', async () => {
      mockPost.mockResolvedValue(ok());
      const { result, invalidateSpy } = setup(() => useWithdrawPlayer(T));

      await act(async () => {
        await result.current.mutateAsync({ participantId: 'p1' });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/withdraw-player`, {
        participantId: 'p1',
        reason: 'Abmeldung durch Schiedsrichter',
      });
      expect(mockToast.success).toHaveBeenCalledWith('Spieler zurückgezogen');
      expect(invalidatedKeys(invalidateSpy)).toEqual([STATUS_KEY]);
    });

    it('passes a custom reason', async () => {
      mockPost.mockResolvedValue(ok());
      const { result } = setup(() => useWithdrawPlayer(T));

      await act(async () => {
        await result.current.mutateAsync({ participantId: 'p1', reason: 'Krank' });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/withdraw-player`, {
        participantId: 'p1',
        reason: 'Krank',
      });
    });

    it('treats an empty reason like a missing one', async () => {
      mockPost.mockResolvedValue(ok());
      const { result } = setup(() => useWithdrawPlayer(T));

      await act(async () => {
        await result.current.mutateAsync({ participantId: 'p1', reason: '' });
      });

      expect(mockPost.mock.calls[0][1]).toMatchObject({
        reason: 'Abmeldung durch Schiedsrichter',
      });
    });

    it('does not invalidate anything on failure', async () => {
      mockPost.mockResolvedValue(fail('Spieler nicht gefunden', 404));
      const { result, invalidateSpy } = setup(() => useWithdrawPlayer(T));

      act(() => result.current.mutate({ participantId: 'nope' }));

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(mockToast.error).toHaveBeenCalledWith('Spieler nicht gefunden');
      expect(invalidateSpy).not.toHaveBeenCalled();
    });
  });

  describe('useReactivatePlayer', () => {
    it('sends missed round points and a default reason', async () => {
      mockPost.mockResolvedValue(ok());
      const { result, invalidateSpy } = setup(() => useReactivatePlayer(T));

      await act(async () => {
        await result.current.mutateAsync({ participantId: 'p1', missedRoundPoints: 0 });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/reactivate-player`, {
        participantId: 'p1',
        missedRoundPoints: 0,
        reason: 'Reaktivierung durch Schiedsrichter',
      });
      expect(mockToast.success).toHaveBeenCalledWith('Spieler reaktiviert');
      expect(invalidatedKeys(invalidateSpy)).toEqual([STATUS_KEY]);
    });
  });

  describe('useAssignBye', () => {
    it('posts the bye and invalidates the status', async () => {
      mockPost.mockResolvedValue(ok());
      const { result, invalidateSpy } = setup(() => useAssignBye(T));

      await act(async () => {
        await result.current.mutateAsync({ participantId: 'p1', roundNumber: 3, byeType: 'half' });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/assign-bye`, {
        participantId: 'p1',
        roundNumber: 3,
        byeType: 'half',
      });
      expect(mockToast.success).toHaveBeenCalledWith('Spielfrei vergeben');
      expect(invalidatedKeys(invalidateSpy)).toEqual([STATUS_KEY]);
    });
  });

  describe('useChangeColor', () => {
    it('uses a default reason', async () => {
      mockPost.mockResolvedValue(ok());
      const { result, invalidateSpy } = setup(() => useChangeColor(T));

      await act(async () => {
        await result.current.mutateAsync({ gameId: 'g7' });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/change-color`, {
        gameId: 'g7',
        reason: 'Farbwechsel durch Schiedsrichter',
      });
      expect(mockToast.success).toHaveBeenCalledWith('Farben getauscht');
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([STATUS_KEY, PAIRINGS_KEY])
      );
    });
  });

  describe('useDeletePairing', () => {
    it('sends a DELETE with the URL-encoded default reason', async () => {
      mockDelete.mockResolvedValue({ status: 204 });
      const { result, invalidateSpy } = setup(() => useDeletePairing(T));

      await act(async () => {
        await result.current.mutateAsync({ gameId: 'g9' });
      });

      expect(mockDelete).toHaveBeenCalledWith(
        `/arbiter/tournaments/${T}/pairings/g9?reason=Paarung%20gel%C3%B6scht%20durch%20Schiedsrichter`
      );
      expect(mockToast.success).toHaveBeenCalledWith('Paarung gelöscht');
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([STATUS_KEY, PAIRINGS_KEY])
      );
    });

    it('URL-encodes special characters of a custom reason', async () => {
      mockDelete.mockResolvedValue({ status: 204 });
      const { result } = setup(() => useDeletePairing(T));

      await act(async () => {
        await result.current.mutateAsync({ gameId: 'g9', reason: 'a&b=c?' });
      });

      expect(mockDelete).toHaveBeenCalledWith(
        `/arbiter/tournaments/${T}/pairings/g9?reason=a%26b%3Dc%3F`
      );
    });

    it('reports errors', async () => {
      mockDelete.mockResolvedValue(fail('Paarung hat Ergebnis'));
      const { result } = setup(() => useDeletePairing(T));

      act(() => result.current.mutate({ gameId: 'g9' }));

      await waitFor(() =>
        expect(mockToast.error).toHaveBeenCalledWith('Paarung hat Ergebnis')
      );
    });
  });

  describe('useSwapPlayers', () => {
    it('posts the swap with a default reason', async () => {
      mockPost.mockResolvedValue(ok());
      const { result } = setup(() => useSwapPlayers(T));

      await act(async () => {
        await result.current.mutateAsync({
          roundNumber: 2,
          game1Id: 'g1',
          player1Id: 'p1',
          game2Id: 'g2',
          player2Id: 'p2',
        });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/swap-pairing`, {
        roundNumber: 2,
        game1Id: 'g1',
        player1Id: 'p1',
        game2Id: 'g2',
        player2Id: 'p2',
        reason: 'Spielertausch durch Schiedsrichter',
      });
      expect(mockToast.success).toHaveBeenCalledWith('Spieler getauscht');
    });
  });

  describe('useCreateManualPairing', () => {
    it('shows one warning toast per warning plus the success toast', async () => {
      mockPost.mockResolvedValue(
        ok({
          gameId: 'new',
          warnings: [
            { type: 'repeat', message: 'Spieler haben bereits gespielt' },
            { type: 'color', message: 'Dritte Weißpartie in Folge' },
          ],
        })
      );
      const { result, invalidateSpy } = setup(() => useCreateManualPairing(T));

      await act(async () => {
        await result.current.mutateAsync({
          roundNumber: 1,
          whiteParticipantId: 'w',
          blackParticipantId: 'b',
          boardNumber: 4,
        });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/create-pairing`, {
        roundNumber: 1,
        whiteParticipantId: 'w',
        blackParticipantId: 'b',
        boardNumber: 4,
      });
      expect(mockToast.warning).toHaveBeenCalledTimes(2);
      expect(mockToast.warning).toHaveBeenNthCalledWith(1, 'Spieler haben bereits gespielt');
      expect(mockToast.warning).toHaveBeenNthCalledWith(2, 'Dritte Weißpartie in Folge');
      expect(mockToast.success).toHaveBeenCalledWith('Paarung erstellt');
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([STATUS_KEY, PAIRINGS_KEY])
      );
    });

    it('shows no warnings for an empty warnings list (bye pairing)', async () => {
      mockPost.mockResolvedValue(ok({ gameId: 'new', warnings: [] }));
      const { result } = setup(() => useCreateManualPairing(T));

      await act(async () => {
        await result.current.mutateAsync({
          roundNumber: 1,
          whiteParticipantId: 'w',
          blackParticipantId: null,
        });
      });

      expect(mockPost.mock.calls[0][1]).toMatchObject({ blackParticipantId: null });
      expect(mockToast.warning).not.toHaveBeenCalled();
      expect(mockToast.success).toHaveBeenCalledWith('Paarung erstellt');
    });

    it('reports errors', async () => {
      mockPost.mockResolvedValue(fail('Spieler bereits gepaart', 409));
      const { result } = setup(() => useCreateManualPairing(T));

      act(() => result.current.mutate({ roundNumber: 1, whiteParticipantId: 'w' }));

      await waitFor(() =>
        expect(mockToast.error).toHaveBeenCalledWith('Spieler bereits gepaart')
      );
      expect(mockToast.success).not.toHaveBeenCalled();
    });
  });

  describe('round control', () => {
    it.each([
      ['useGeneratePairings', useGeneratePairings, 'pair', 'Paarungen erstellt'],
      ['useStartRound', useStartRound, 'start', 'Runde gestartet'],
      ['useCompleteRound', useCompleteRound, 'complete', 'Runde beendet'],
    ] as const)('%s posts to the round endpoint', async (_name, hook, action, message) => {
      mockPost.mockResolvedValue(ok());
      const { result, invalidateSpy } = setup(() => hook(T));

      await act(async () => {
        await result.current.mutateAsync(5);
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/rounds/5/${action}`);
      expect(mockToast.success).toHaveBeenCalledWith(message);
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([STATUS_KEY, PAIRINGS_KEY])
      );
    });

    it.each([
      ['useGeneratePairings', useGeneratePairings],
      ['useStartRound', useStartRound],
      ['useCompleteRound', useCompleteRound],
    ] as const)('%s reports API errors', async (_name, hook) => {
      mockPost.mockResolvedValue(fail('Vorherige Runde nicht abgeschlossen'));
      const { result, invalidateSpy } = setup(() => hook(T));

      act(() => result.current.mutate(2));

      await waitFor(() =>
        expect(mockToast.error).toHaveBeenCalledWith('Vorherige Runde nicht abgeschlossen')
      );
      expect(invalidateSpy).not.toHaveBeenCalled();
    });
  });

  describe('useCreateAnnouncement', () => {
    it('posts the announcement without invalidating arbiter queries', async () => {
      mockPost.mockResolvedValue(ok({ id: 'a1' }));
      const { result, invalidateSpy } = setup(() => useCreateAnnouncement(T));

      await act(async () => {
        await result.current.mutateAsync({
          title: 'Pause',
          message: '15 Minuten Pause',
          priority: 'info',
        });
      });

      expect(mockPost).toHaveBeenCalledWith(`/arbiter/tournaments/${T}/announcements`, {
        title: 'Pause',
        message: '15 Minuten Pause',
        priority: 'info',
      });
      expect(mockToast.success).toHaveBeenCalledWith('Ankündigung erstellt');
      expect(invalidateSpy).not.toHaveBeenCalled();
    });

    it('reports errors', async () => {
      mockPost.mockResolvedValue(fail('Titel fehlt'));
      const { result } = setup(() => useCreateAnnouncement(T));

      act(() => result.current.mutate({ title: '', message: 'x', priority: 'urgent' }));

      await waitFor(() => expect(mockToast.error).toHaveBeenCalledWith('Titel fehlt'));
    });
  });
});
