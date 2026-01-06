'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { AssignByeForm } from '../forms/AssignByeForm';
import { CorrectResultForm } from '../forms/CorrectResultForm';
import { LateEntryDialog } from '../dialogs/LateEntryDialog';
import type { ActionsTabProps } from '../types';

/**
 * Tab for special arbiter actions (assign bye, correct result, late entry)
 */
export function ActionsTab({
  tournamentId,
  currentRound,
  participants,
  tournamentRounds,
}: ActionsTabProps) {
  const activeParticipants = participants.filter((p) => p.isActive);

  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Spielfrei vergeben</CardTitle>
          <CardDescription>
            Vergeben Sie einem Spieler ein Spielfrei für die aktuelle Runde
          </CardDescription>
        </CardHeader>
        <CardContent>
          <AssignByeForm
            tournamentId={tournamentId}
            roundNumber={currentRound?.roundNumber || 1}
            participants={activeParticipants}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Ergebnis korrigieren</CardTitle>
          <CardDescription>Korrigieren Sie ein fehlerhaft eingetragenes Ergebnis</CardDescription>
        </CardHeader>
        <CardContent>
          <CorrectResultForm tournamentId={tournamentId} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Nachnennung</CardTitle>
          <CardDescription>
            Fügen Sie einen Spieler nachträglich zum laufenden Turnier hinzu
          </CardDescription>
        </CardHeader>
        <CardContent>
          <LateEntryDialog
            tournamentId={tournamentId}
            currentRoundNumber={currentRound?.roundNumber || 1}
            tournamentRounds={tournamentRounds}
          />
        </CardContent>
      </Card>
    </div>
  );
}
