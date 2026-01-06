'use client';

import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Loader2 } from 'lucide-react';
import { api } from '@/lib/api';
import { useCorrectResult } from '../hooks/useArbiterMutations';
import type { CorrectResultFormProps } from '../types';
import type { GameResult, PairingGame, RoundInfo } from '@/types/arbiter';

/**
 * Form for correcting a game result
 *
 * BUG-4 FIX: Added game selector instead of requiring manual UUID input
 */
export function CorrectResultForm({ tournamentId }: CorrectResultFormProps) {
  const [selectedRound, setSelectedRound] = useState('');
  const [gameId, setGameId] = useState('');
  const [newResult, setNewResult] = useState<GameResult | ''>('');
  const [reason, setReason] = useState('');
  const mutation = useCorrectResult(tournamentId);

  // Fetch rounds
  const { data: roundsData, isLoading: roundsLoading } = useQuery({
    queryKey: ['rounds', tournamentId],
    queryFn: async () => {
      const response = await api.get<{ rounds: RoundInfo[] }>(
        `/tournaments/${tournamentId}/rounds`
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
  });

  // Fetch games for selected round
  const { data: gamesData, isLoading: gamesLoading } = useQuery({
    queryKey: ['pairings', tournamentId, selectedRound],
    queryFn: async () => {
      const response = await api.get<{ games: PairingGame[] }>(
        `/tournaments/${tournamentId}/rounds/${selectedRound}/pairings`
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    enabled: !!selectedRound,
  });

  // Filter to only show completed games (ones with results)
  const completedGames = useMemo(() => {
    if (!gamesData?.games) return [];
    return gamesData.games.filter((g) => g.result);
  }, [gamesData?.games]);

  // Get selected game details for display
  const selectedGame = useMemo(() => {
    if (!gameId || !completedGames.length) return null;
    return completedGames.find((g) => g.id === gameId);
  }, [gameId, completedGames]);

  const handleRoundChange = (round: string) => {
    setSelectedRound(round);
    setGameId('');
  };

  const handleSubmit = () => {
    if (!gameId || !newResult || !reason) return;

    mutation.mutate(
      { gameId, newResult, reason },
      {
        onSuccess: () => {
          setGameId('');
          setNewResult('');
          setReason('');
        },
      }
    );
  };

  if (roundsLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-20 w-full" />
      </div>
    );
  }

  const rounds = roundsData?.rounds || [];
  // Only show rounds that have completed games
  const roundsWithGames = rounds.filter(
    (r) => r.status === 'completed' || r.status === 'ongoing'
  );

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="correct-round">Runde</Label>
        <Select value={selectedRound} onValueChange={handleRoundChange}>
          <SelectTrigger id="correct-round">
            <SelectValue placeholder="Runde wählen" />
          </SelectTrigger>
          <SelectContent>
            {roundsWithGames.map((r) => (
              <SelectItem key={r.id} value={r.roundNumber.toString()}>
                Runde {r.roundNumber}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {selectedRound && (
        <div className="space-y-2">
          <Label htmlFor="correct-game">Partie</Label>
          {gamesLoading ? (
            <Skeleton className="h-10 w-full" />
          ) : completedGames.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Keine Partien mit Ergebnis in dieser Runde
            </p>
          ) : (
            <Select value={gameId} onValueChange={setGameId}>
              <SelectTrigger id="correct-game">
                <SelectValue placeholder="Partie auswählen" />
              </SelectTrigger>
              <SelectContent>
                {completedGames.map((g) => (
                  <SelectItem key={g.id} value={g.id}>
                    Brett {g.boardNumber}: {g.whiteName || 'Weiß'} - {g.blackName || 'Schwarz'}{' '}
                    ({g.result})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>
      )}

      {selectedGame && (
        <div className="rounded-md bg-muted p-3 text-sm">
          <p className="font-medium">Aktuelles Ergebnis: {selectedGame.result}</p>
          <p className="text-muted-foreground">
            {selectedGame.whiteName || 'Weiß'} vs {selectedGame.blackName || 'Schwarz'}
          </p>
        </div>
      )}

      <div className="space-y-2">
        <Label htmlFor="correct-result">Neues Ergebnis</Label>
        <Select
          value={newResult}
          onValueChange={(v) => setNewResult(v as GameResult)}
          disabled={!gameId}
        >
          <SelectTrigger id="correct-result">
            <SelectValue placeholder="Ergebnis wählen" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="1-0">1 - 0</SelectItem>
            <SelectItem value="0-1">0 - 1</SelectItem>
            <SelectItem value="1/2-1/2">½ - ½</SelectItem>
            <SelectItem value="1-0F">1 - 0 (kampflos)</SelectItem>
            <SelectItem value="0-1F">0 - 1 (kampflos)</SelectItem>
            <SelectItem value="0-0F">0 - 0</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label htmlFor="correct-reason">
          Begründung <span className="text-destructive">*</span>
        </Label>
        <Textarea
          id="correct-reason"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Grund für die Korrektur (Pflichtfeld)"
          disabled={!gameId}
        />
      </div>

      <Button
        onClick={handleSubmit}
        disabled={!gameId || !newResult || !reason || mutation.isPending}
        className="w-full"
      >
        {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
        Ergebnis korrigieren
      </Button>
    </div>
  );
}
