'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { CheckCircle2 } from 'lucide-react';
import { ResultEntryDialog } from '../dialogs/ResultEntryDialog';
import type { ActiveGamesTabProps } from '../types';

/**
 * Tab showing active games with result entry capability
 */
export function ActiveGamesTab({ tournamentId, games, roundStatus }: ActiveGamesTabProps) {
  if (games.length === 0) {
    return (
      <Card>
        <CardContent className="py-10 text-center">
          <CheckCircle2 className="mx-auto h-12 w-12 text-green-500 mb-4" />
          <p className="text-muted-foreground">Alle Partien beendet oder keine Runde aktiv</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Aktive Partien ({games.length})</CardTitle>
        <CardDescription>Klicken Sie auf eine Partie, um das Ergebnis einzutragen</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-center py-2 px-2">Brett</th>
                <th className="text-left py-2 px-2">Weiß</th>
                <th className="text-center py-2 px-2">Status</th>
                <th className="text-right py-2 px-2">Schwarz</th>
                <th className="text-center py-2 px-2">Zeit</th>
                <th className="w-20"></th>
              </tr>
            </thead>
            <tbody>
              {games.map((game) => (
                <tr key={game.game.id} className="border-b">
                  <td className="text-center py-3 px-2 font-medium">{game.game.boardNumber}</td>
                  <td className="py-3 px-2">
                    {game.whitePlayer ? (
                      <div>
                        <span>{game.whitePlayer.name || game.whitePlayer.participantId}</span>
                        <span className="text-xs text-muted-foreground ml-2">
                          ({game.whitePlayer.rating})
                        </span>
                      </div>
                    ) : (
                      '-'
                    )}
                  </td>
                  <td className="text-center py-3 px-2">
                    {game.game.result ? (
                      <Badge>{game.game.result}</Badge>
                    ) : (
                      <Badge variant="outline">läuft</Badge>
                    )}
                  </td>
                  <td className="text-right py-3 px-2">
                    {game.blackPlayer ? (
                      <div>
                        <span className="text-xs text-muted-foreground mr-2">
                          ({game.blackPlayer.rating})
                        </span>
                        <span>{game.blackPlayer.name || game.blackPlayer.participantId}</span>
                      </div>
                    ) : (
                      <span className="italic text-muted-foreground">spielfrei</span>
                    )}
                  </td>
                  <td className="text-center py-3 px-2 text-muted-foreground">
                    {game.elapsedMinutes} min
                  </td>
                  <td className="py-3 px-2">
                    {!game.game.result && roundStatus === 'ongoing' && (
                      <ResultEntryDialog
                        tournamentId={tournamentId}
                        gameId={game.game.id}
                        boardNumber={game.game.boardNumber}
                        whiteName={game.whitePlayer?.name || 'Weiß'}
                        blackName={game.blackPlayer?.name || 'Schwarz'}
                      />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}
