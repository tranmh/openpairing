'use client';

import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { usePairings } from '../hooks/useArbiterStatus';
import { ChangeColorDialog } from '../dialogs/ChangeColorDialog';
import { DeletePairingDialog } from '../dialogs/DeletePairingDialog';
import { SwapPlayersDialog } from '../dialogs/SwapPlayersDialog';
import { CreateManualPairingDialog } from '../dialogs/CreateManualPairingDialog';
import type { PairingsManagementTabProps } from '../types';

/**
 * Tab for managing pairings (view, edit colors, delete, swap players, create manual)
 */
export function PairingsManagementTab({
  tournamentId,
  rounds,
  participants,
}: PairingsManagementTabProps) {
  const [selectedRound, setSelectedRound] = useState<string>(
    rounds.length > 0 ? rounds[rounds.length - 1].roundNumber.toString() : '1'
  );

  const selectedRoundData = rounds.find((r) => r.roundNumber === parseInt(selectedRound));
  const { data: pairingsData, isLoading } = usePairings(tournamentId, selectedRound);
  const pairings = pairingsData?.games || [];
  const canEdit = selectedRoundData?.status === 'paired';

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Paarungsverwaltung</CardTitle>
              <CardDescription>
                Bearbeiten Sie die Paarungen der ausgewählten Runde
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <Select value={selectedRound} onValueChange={setSelectedRound}>
                <SelectTrigger className="w-[150px]">
                  <SelectValue placeholder="Runde" />
                </SelectTrigger>
                <SelectContent>
                  {rounds.map((r) => (
                    <SelectItem key={r.id} value={r.roundNumber.toString()}>
                      Runde {r.roundNumber}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {canEdit && (
                <>
                  <SwapPlayersDialog
                    tournamentId={tournamentId}
                    roundNumber={parseInt(selectedRound)}
                    pairings={pairings}
                  />
                  <CreateManualPairingDialog
                    tournamentId={tournamentId}
                    roundNumber={parseInt(selectedRound)}
                    participants={participants}
                    existingPairings={pairings}
                  />
                </>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : pairings.length === 0 ? (
            <div className="text-center py-10">
              <p className="text-muted-foreground">Keine Paarungen für diese Runde vorhanden</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-center py-2 px-2">Brett</th>
                    <th className="text-left py-2 px-2">Weiß</th>
                    <th className="text-center py-2 px-2">Ergebnis</th>
                    <th className="text-right py-2 px-2">Schwarz</th>
                    {canEdit && <th className="w-24 text-center">Aktionen</th>}
                  </tr>
                </thead>
                <tbody>
                  {pairings.map((game) => (
                    <tr key={game.id} className="border-b">
                      <td className="text-center py-2 px-2 font-medium">{game.boardNumber}</td>
                      <td className="py-2 px-2">
                        {game.whiteName || game.whiteParticipantId || '-'}
                        {game.whiteRating && (
                          <span className="text-xs text-muted-foreground ml-1">
                            ({game.whiteRating})
                          </span>
                        )}
                      </td>
                      <td className="text-center py-2 px-2">
                        {game.result ? (
                          <Badge>{game.result}</Badge>
                        ) : (
                          <Badge variant="outline">-</Badge>
                        )}
                      </td>
                      <td className="text-right py-2 px-2">
                        {game.blackParticipantId ? (
                          <>
                            {game.blackRating && (
                              <span className="text-xs text-muted-foreground mr-1">
                                ({game.blackRating})
                              </span>
                            )}
                            {game.blackName || game.blackParticipantId}
                          </>
                        ) : (
                          <span className="italic text-muted-foreground">spielfrei</span>
                        )}
                      </td>
                      {canEdit && (
                        <td className="py-2 px-2">
                          <div className="flex justify-center gap-1">
                            {game.blackParticipantId && !game.result && (
                              <ChangeColorDialog
                                tournamentId={tournamentId}
                                gameId={game.id}
                                whiteName={game.whiteName || 'Weiß'}
                                blackName={game.blackName || 'Schwarz'}
                              />
                            )}
                            {!game.result && (
                              <DeletePairingDialog
                                tournamentId={tournamentId}
                                gameId={game.id}
                                boardNumber={game.boardNumber}
                              />
                            )}
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {!canEdit && selectedRoundData && (
        <p className="text-sm text-muted-foreground text-center">
          Paarungen können nur bearbeitet werden, wenn die Runde den Status &quot;Gepaart&quot; hat.
          Aktueller Status: <Badge variant="outline">{selectedRoundData.status}</Badge>
        </p>
      )}
    </div>
  );
}
