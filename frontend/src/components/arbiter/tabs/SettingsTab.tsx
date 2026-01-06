'use client';

import { CloneTournamentDialog } from '../dialogs/CloneTournamentDialog';
import { ForbiddenPairsCard } from '../cards/ForbiddenPairsCard';
import { SpecialRankingsCard } from '../cards/SpecialRankingsCard';
import type { SettingsTabProps } from '../types';

/**
 * Tab for tournament settings (clone, forbidden pairs, special rankings)
 */
export function SettingsTab({
  tournamentId,
  tournamentName,
  participants,
}: SettingsTabProps) {
  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-lg font-semibold">Turniereinstellungen</h2>
        <CloneTournamentDialog tournamentId={tournamentId} tournamentName={tournamentName} />
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <ForbiddenPairsCard tournamentId={tournamentId} participants={participants} />
        <SpecialRankingsCard tournamentId={tournamentId} />
      </div>
    </div>
  );
}
