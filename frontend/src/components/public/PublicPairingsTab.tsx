'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { PairingsView } from '@/components/tournament/PairingsView';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface Round {
  roundNumber: number;
  status: string;
  startTime?: string;
  endTime?: string;
  gamesCount: number;
}

interface RoundsResponse {
  rounds: Round[];
  currentRound: number;
}

interface Game {
  id: string;
  boardNumber: number;
  whiteParticipantId?: string;
  blackParticipantId?: string;
  whitePlayerName?: string;
  blackPlayerName?: string;
  whiteRating?: number;
  blackRating?: number;
  result?: string;
  status: string;
}

interface PairingsResponse {
  roundNumber: number;
  status: string;
  games: Game[];
}

interface PublicPairingsTabProps {
  tournamentId: string;
  currentRound?: number;
  totalRounds?: number;
}

export function PublicPairingsTab({
  tournamentId,
  currentRound = 1,
  totalRounds = 1,
}: PublicPairingsTabProps) {
  const [selectedRound, setSelectedRound] = useState(currentRound);

  // Fetch rounds info
  const { data: roundsData } = useQuery({
    queryKey: ['rounds', tournamentId],
    queryFn: async () => {
      const response = await api.get<RoundsResponse>(
        `/tournaments/${tournamentId}/rounds`
      );
      return response.data;
    },
    enabled: !!tournamentId,
  });

  // Fetch pairings for selected round
  const {
    data: pairingsData,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['pairings', tournamentId, selectedRound],
    queryFn: async () => {
      const response = await api.get<PairingsResponse>(
        `/tournaments/${tournamentId}/rounds/${selectedRound}/pairings`
      );
      return response.data;
    },
    enabled: !!tournamentId && selectedRound > 0,
  });

  const rounds = roundsData?.rounds || [];
  const pairedRounds = rounds.filter((r) => r.status !== 'pending');
  const maxRound = Math.max(
    pairedRounds.length > 0 ? Math.max(...pairedRounds.map((r) => r.roundNumber)) : 0,
    currentRound
  );

  // Transform games to match PairingsView expected format (whiteName/blackName)
  const transformedGames = pairingsData?.games.map((game) => ({
    ...game,
    whiteName: game.whitePlayerName,
    blackName: game.blackPlayerName,
  })) || [];

  const canGoPrev = selectedRound > 1;
  const canGoNext = selectedRound < maxRound;

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Fehler</AlertTitle>
        <AlertDescription>
          Die Paarungen konnten nicht geladen werden.
        </AlertDescription>
      </Alert>
    );
  }

  if (!pairingsData || transformedGames.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        Noch keine Paarungen für diese Runde vorhanden.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Round Navigation */}
      <div className="flex items-center justify-between">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setSelectedRound((r) => r - 1)}
          disabled={!canGoPrev}
        >
          <ChevronLeft className="h-4 w-4 mr-1" />
          Vorherige
        </Button>

        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">Runde {selectedRound}</span>
          <span className="text-sm text-muted-foreground">
            von {totalRounds}
          </span>
          {pairingsData.status === 'ongoing' && (
            <span className="text-xs text-green-600 font-medium">
              (läuft)
            </span>
          )}
          {pairingsData.status === 'completed' && (
            <span className="text-xs text-muted-foreground">
              (beendet)
            </span>
          )}
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => setSelectedRound((r) => r + 1)}
          disabled={!canGoNext}
        >
          Nächste
          <ChevronRight className="h-4 w-4 ml-1" />
        </Button>
      </div>

      {/* Round Dots for quick navigation */}
      {totalRounds <= 15 && (
        <div className="flex justify-center gap-1">
          {Array.from({ length: maxRound }, (_, i) => i + 1).map((round) => {
            const roundInfo = rounds.find((r) => r.roundNumber === round);
            const isPaired = roundInfo && roundInfo.status !== 'pending';
            return (
              <button
                key={round}
                onClick={() => isPaired && setSelectedRound(round)}
                disabled={!isPaired}
                className={`w-8 h-8 rounded-full text-xs font-medium transition-colors ${
                  round === selectedRound
                    ? 'bg-primary text-primary-foreground'
                    : isPaired
                    ? 'bg-muted hover:bg-muted-foreground/20'
                    : 'bg-muted/50 text-muted-foreground cursor-not-allowed'
                }`}
              >
                {round}
              </button>
            );
          })}
        </div>
      )}

      {/* Games count */}
      <div className="text-sm text-muted-foreground text-center">
        {transformedGames.length} Paarungen
      </div>

      {/* Pairings Table */}
      <PairingsView
        tournamentId={tournamentId}
        games={transformedGames}
        roundNumber={selectedRound}
        roundStatus={pairingsData.status}
        canEdit={false}
      />
    </div>
  );
}
