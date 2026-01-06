'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { StatusCardsProps } from '../types';

/**
 * Status cards showing tournament overview statistics
 */
export function StatusCards({
  currentRound,
  totalRounds,
  statistics,
  activeGamesCount,
  pendingResults,
}: StatusCardsProps) {
  const roundStatusLabel =
    currentRound?.status === 'ongoing'
      ? 'Läuft'
      : currentRound?.status === 'paired'
        ? 'Gepaart'
        : 'Ausstehend';

  return (
    <div className="grid gap-4 md:grid-cols-4">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Aktuelle Runde
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {currentRound?.roundNumber || '-'} / {totalRounds}
          </div>
          <p className="text-xs text-muted-foreground">{roundStatusLabel}</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Teilnehmer
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{statistics.activeParticipants}</div>
          <p className="text-xs text-muted-foreground">
            {statistics.withdrawnCount} zurückgezogen
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Offene Ergebnisse
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{pendingResults}</div>
          <p className="text-xs text-muted-foreground">{activeGamesCount} aktive Partien</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Durchschnittsrating
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{Math.round(statistics.averageRating)}</div>
          <p className="text-xs text-muted-foreground">Elo/DWZ</p>
        </CardContent>
      </Card>
    </div>
  );
}
