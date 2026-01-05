'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

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
  isActive: boolean;
}

interface StandingsResponseData {
  standings: StandingResponse[];
}

interface PublicParticipantsTabProps {
  tournamentId: string;
  participantCount?: number;
}

const titleColors: Record<string, string> = {
  GM: 'bg-amber-100 text-amber-800',
  IM: 'bg-gray-100 text-gray-800',
  FM: 'bg-green-100 text-green-800',
  CM: 'bg-blue-100 text-blue-800',
  WGM: 'bg-amber-100 text-amber-800',
  WIM: 'bg-gray-100 text-gray-800',
  WFM: 'bg-green-100 text-green-800',
  WCM: 'bg-blue-100 text-blue-800',
};

export function PublicParticipantsTab({
  tournamentId,
  participantCount,
}: PublicParticipantsTabProps) {
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
    // If standings not available, show participant count if we have it
    if (participantCount !== undefined) {
      return (
        <div className="text-center py-8 text-muted-foreground">
          {participantCount} Teilnehmer angemeldet
        </div>
      );
    }
    return (
      <Alert variant="destructive">
        <AlertTitle>Fehler</AlertTitle>
        <AlertDescription>
          Die Teilnehmerliste konnte nicht geladen werden.
        </AlertDescription>
      </Alert>
    );
  }

  if (!data || data.standings.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        Noch keine Teilnehmer vorhanden.
      </div>
    );
  }

  // Sort by rating (descending) for participant list
  const participants = [...data.standings].sort((a, b) => b.rating - a.rating);

  return (
    <div className="space-y-4">
      <div className="text-sm text-muted-foreground">
        {participants.length} Teilnehmer
      </div>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-12 text-center">Nr.</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Verein</TableHead>
              <TableHead className="text-right w-16">ELO</TableHead>
              <TableHead className="text-center w-20">Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {participants.map((player, index) => (
              <TableRow key={player.participantId}>
                <TableCell className="text-center text-muted-foreground">
                  {index + 1}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    {player.playerTitle && (
                      <Badge
                        variant="secondary"
                        className={cn('text-xs', titleColors[player.playerTitle])}
                      >
                        {player.playerTitle}
                      </Badge>
                    )}
                    <span>{player.playerName}</span>
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {player.clubName || '-'}
                </TableCell>
                <TableCell className="text-right">
                  {player.rating || '-'}
                </TableCell>
                <TableCell className="text-center">
                  {player.isActive ? (
                    <Badge variant="outline" className="text-green-600 border-green-600">
                      Aktiv
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-gray-500 border-gray-500">
                      Inaktiv
                    </Badge>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
