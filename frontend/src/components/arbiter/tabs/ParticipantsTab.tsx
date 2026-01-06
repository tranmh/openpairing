'use client';

import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { UserMinus, Loader2 } from 'lucide-react';
import { useWithdrawPlayer } from '../hooks/useArbiterMutations';
import { RatingOverrideDialog } from '../dialogs/RatingOverrideDialog';
import { ReactivatePlayerDialog } from '../dialogs/ReactivatePlayerDialog';
import type { ParticipantsTabProps } from '../types';

/**
 * Tab showing active and withdrawn participants with management actions
 */
export function ParticipantsTab({ tournamentId, participants }: ParticipantsTabProps) {
  const [withdrawing, setWithdrawing] = useState<string | null>(null);
  const withdrawMutation = useWithdrawPlayer(tournamentId);

  const handleWithdraw = (participantId: string) => {
    withdrawMutation.mutate(
      { participantId },
      {
        onSuccess: () => setWithdrawing(null),
      }
    );
  };

  const activeParticipants = participants.filter((p) => p.isActive);
  const withdrawnParticipants = participants.filter((p) => !p.isActive);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Aktive Teilnehmer ({activeParticipants.length})</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b">
                  <th className="text-center py-2 px-2">Nr.</th>
                  <th className="text-left py-2 px-2">Spieler</th>
                  <th className="text-right py-2 px-2">Rating</th>
                  <th className="text-right py-2 px-2">Punkte</th>
                  <th className="w-24 text-center py-2 px-2">Aktionen</th>
                </tr>
              </thead>
              <tbody>
                {activeParticipants.map((p) => {
                  const displayName = p.playerName || (p.playerLastName && p.playerFirstName ? `${p.playerLastName}, ${p.playerFirstName}` : p.playerId);
                  return (
                  <tr key={p.id} className="border-b">
                    <td className="text-center py-2 px-2">{p.startRank}</td>
                    <td className="py-2 px-2">{displayName}</td>
                    <td className="text-right py-2 px-2">{p.startingRating}</td>
                    <td className="text-right py-2 px-2">{p.points}</td>
                    <td className="py-2 px-2 flex justify-center gap-1">
                      {/* BUG-2 FIX: Pass hasOverride prop */}
                      <RatingOverrideDialog
                        tournamentId={tournamentId}
                        participantId={p.id}
                        playerName={displayName}
                        currentRating={p.startingRating}
                        hasOverride={p.manualRatingOverride !== null && p.manualRatingOverride !== undefined}
                      />
                      <Dialog
                        open={withdrawing === p.id}
                        onOpenChange={(open) => setWithdrawing(open ? p.id : null)}
                      >
                        <DialogTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            aria-label="Spieler zurückziehen"
                          >
                            <UserMinus className="h-4 w-4" />
                          </Button>
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>Spieler zurückziehen?</DialogTitle>
                            <DialogDescription>
                              Der Spieler wird aus dem Turnier zurückgezogen. Laufende Partien werden
                              als Niederlage gewertet.
                            </DialogDescription>
                          </DialogHeader>
                          <DialogFooter>
                            <Button variant="outline" onClick={() => setWithdrawing(null)}>
                              Abbrechen
                            </Button>
                            <Button
                              variant="destructive"
                              onClick={() => handleWithdraw(p.id)}
                              disabled={withdrawMutation.isPending}
                            >
                              {withdrawMutation.isPending && (
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                              )}
                              Zurückziehen
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                    </td>
                  </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {withdrawnParticipants.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-muted-foreground">
              Zurückgezogene Spieler ({withdrawnParticipants.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-center py-2 px-2">Nr.</th>
                    <th className="text-left py-2 px-2">Spieler</th>
                    <th className="text-right py-2 px-2">Rating</th>
                    <th className="text-right py-2 px-2">Punkte</th>
                    <th className="w-20"></th>
                  </tr>
                </thead>
                <tbody>
                  {withdrawnParticipants.map((p) => {
                    const displayName = p.playerName || (p.playerLastName && p.playerFirstName ? `${p.playerLastName}, ${p.playerFirstName}` : p.playerId);
                    return (
                    <tr key={p.id} className="border-b opacity-60 hover:opacity-100">
                      <td className="text-center py-2 px-2">{p.startRank}</td>
                      <td className="py-2 px-2">{displayName}</td>
                      <td className="text-right py-2 px-2">{p.startingRating}</td>
                      <td className="text-right py-2 px-2">{p.points}</td>
                      <td className="py-2 px-2">
                        <ReactivatePlayerDialog
                          tournamentId={tournamentId}
                          participantId={p.id}
                          playerName={displayName}
                        />
                      </td>
                    </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
