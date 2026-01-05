'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { StandingsTable } from '@/components/tournament/StandingsTable';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';

interface StandingResponse {
  rank: number;
  participantId: string;
  playerId: string;
  playerName: string;
  playerFideId?: string;
  playerTitle?: string;
  clubName?: string;
  rating: number;
  points: number;
  gamesPlayed: number;
  wins: number;
  draws: number;
  losses: number;
  tiebreaks: Record<string, number>;
  isActive: boolean;
  performanceRating?: number;
}

interface StandingsResponseData {
  tournamentId: string;
  tournamentName: string;
  currentRound: number;
  totalRounds: number;
  lastUpdated: string;
  standings: StandingResponse[];
  tiebreakOrder: string[];
}

interface PublicStandingsTabProps {
  tournamentId: string;
}

export function PublicStandingsTab({ tournamentId }: PublicStandingsTabProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['standings', tournamentId],
    queryFn: async () => {
      const response = await api.get<StandingsResponseData>(
        `/tournaments/${tournamentId}/standings`
      );
      return response.data;
    },
    enabled: !!tournamentId,
  });

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
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
          Die Rangliste konnte nicht geladen werden.
        </AlertDescription>
      </Alert>
    );
  }

  if (!data || data.standings.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        Noch keine Ergebnisse vorhanden.
      </div>
    );
  }

  // Transform backend response to StandingsTable format
  const standings = data.standings.map((s) => ({
    rank: s.rank,
    participantId: s.participantId,
    playerId: s.playerId,
    name: s.playerName,
    title: s.playerTitle,
    rating: s.rating,
    federation: undefined, // Not in backend response
    points: s.points,
    buchholz: s.tiebreaks?.buchholz ?? 0,
    buchholzCut1: s.tiebreaks?.buchholz_cut1 ?? 0,
    sonnebornBerger: s.tiebreaks?.sonneborn_berger ?? 0,
    wins: s.wins,
    draws: s.draws,
    losses: s.losses,
  }));

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between text-sm text-muted-foreground">
        <span>
          Runde {data.currentRound} von {data.totalRounds}
        </span>
        <span>{data.standings.length} Teilnehmer</span>
      </div>
      <StandingsTable
        standings={standings}
        tiebreaks={data.tiebreakOrder}
        showResults={true}
      />
    </div>
  );
}
