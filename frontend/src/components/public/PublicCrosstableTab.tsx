'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { CrossTable } from '@/components/tournament/CrossTable';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';

interface CrosstableEntry {
  opponentRank: number;
  result?: string; // "1", "0", "=", "+", "-"
  color: string; // "w", "b", "-"
}

interface CrosstableRow {
  rank: number;
  playerName: string;
  rating: number;
  results: CrosstableEntry[];
  points: number;
}

interface CrosstableResponseData {
  tournamentId: string;
  tournamentName: string;
  currentRound: number;
  totalRounds: number;
  rows: CrosstableRow[];
}

// Also fetch standings to get additional player info
interface StandingResponse {
  rank: number;
  participantId: string;
  playerId: string;
  playerName: string;
  playerTitle?: string;
  tiebreaks: Record<string, number>;
}

interface StandingsResponseData {
  standings: StandingResponse[];
}

interface PublicCrosstableTabProps {
  tournamentId: string;
}

export function PublicCrosstableTab({ tournamentId }: PublicCrosstableTabProps) {
  // Fetch crosstable
  const {
    data: crosstableData,
    isLoading: crosstableLoading,
    error: crosstableError,
  } = useQuery({
    queryKey: ['crosstable', tournamentId],
    queryFn: async () => {
      const response = await api.get<CrosstableResponseData>(
        `/tournaments/${tournamentId}/crosstable`
      );
      return response.data;
    },
    enabled: !!tournamentId,
  });

  // Fetch standings to enrich with participantId, playerId, title, tiebreaks
  const { data: standingsData } = useQuery({
    queryKey: ['standings', tournamentId],
    queryFn: async () => {
      const response = await api.get<StandingsResponseData>(
        `/tournaments/${tournamentId}/standings`
      );
      return response.data;
    },
    enabled: !!tournamentId,
  });

  const isLoading = crosstableLoading;
  const error = crosstableError;

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
          Die Kreuztabelle konnte nicht geladen werden.
        </AlertDescription>
      </Alert>
    );
  }

  if (!crosstableData || crosstableData.rows.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        Noch keine Ergebnisse für die Kreuztabelle vorhanden.
      </div>
    );
  }

  // Create a map of standings by rank for enrichment
  const standingsByRank = new Map(
    standingsData?.standings.map((s) => [s.rank, s]) || []
  );

  // Transform backend response to CrossTable format
  const players = crosstableData.rows.map((row) => {
    const standing = standingsByRank.get(row.rank);

    return {
      rank: row.rank,
      participantId: standing?.participantId || `participant-${row.rank}`,
      playerId: standing?.playerId || `player-${row.rank}`,
      name: row.playerName,
      title: standing?.playerTitle,
      rating: row.rating,
      points: row.points,
      buchholz: standing?.tiebreaks?.buchholz ?? 0,
      sonnebornBerger: standing?.tiebreaks?.sonneborn_berger ?? 0,
      results: row.results.map((entry, index) => ({
        round: index + 1,
        opponentRank: entry.opponentRank,
        color: entry.color,
        score: resultToScore(entry.result),
      })),
    };
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between text-sm text-muted-foreground">
        <span>
          Runde {crosstableData.currentRound} von {crosstableData.totalRounds}
        </span>
        <span>{crosstableData.rows.length} Teilnehmer</span>
      </div>
      <CrossTable players={players} rounds={crosstableData.totalRounds} />
    </div>
  );
}

function resultToScore(result?: string): number {
  if (!result) return 0;
  switch (result) {
    case '1':
    case '+':
      return 1;
    case '=':
      return 0.5;
    case '0':
    case '-':
      return 0;
    default:
      return 0;
  }
}
