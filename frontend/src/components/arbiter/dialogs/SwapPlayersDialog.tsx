'use client';

import { useState, useEffect, useMemo } from 'react';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { ArrowLeftRight, Loader2 } from 'lucide-react';
import { useSwapPlayers } from '../hooks/useArbiterMutations';
import type { SwapPlayersDialogProps } from '../types';

/**
 * Dialog for swapping players between two pairings
 *
 * BUG-3 FIX: Properly reset player selections when games change
 */
export function SwapPlayersDialog({
  tournamentId,
  roundNumber,
  pairings,
}: SwapPlayersDialogProps) {
  const [open, setOpen] = useState(false);
  const [game1Id, setGame1Id] = useState('');
  const [player1Id, setPlayer1Id] = useState('');
  const [game2Id, setGame2Id] = useState('');
  const [player2Id, setPlayer2Id] = useState('');
  const [reason, setReason] = useState('');
  const mutation = useSwapPlayers(tournamentId);

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      setGame1Id('');
      setPlayer1Id('');
      setGame2Id('');
      setPlayer2Id('');
      setReason('');
    }
  }, [open]);

  // BUG-3 FIX: Reset player selection when game changes
  useEffect(() => {
    setPlayer1Id('');
  }, [game1Id]);

  useEffect(() => {
    setPlayer2Id('');
  }, [game2Id]);

  const game1 = useMemo(() => pairings.find((p) => p.id === game1Id), [pairings, game1Id]);
  const game2 = useMemo(() => pairings.find((p) => p.id === game2Id), [pairings, game2Id]);

  const game1Players = useMemo(() => {
    if (!game1) return [];
    return [
      { id: game1.whiteParticipantId, name: game1.whiteName || 'Weiß', color: 'Weiß' },
      { id: game1.blackParticipantId, name: game1.blackName || 'Schwarz', color: 'Schwarz' },
    ].filter((p) => p.id);
  }, [game1]);

  const game2Players = useMemo(() => {
    if (!game2) return [];
    return [
      { id: game2.whiteParticipantId, name: game2.whiteName || 'Weiß', color: 'Weiß' },
      { id: game2.blackParticipantId, name: game2.blackName || 'Schwarz', color: 'Schwarz' },
    ].filter((p) => p.id);
  }, [game2]);

  // Only show games without results and with both players
  const availableGames = useMemo(
    () => pairings.filter((p) => !p.result && p.blackParticipantId),
    [pairings]
  );

  const handleSubmit = () => {
    mutation.mutate(
      {
        roundNumber,
        game1Id,
        player1Id,
        game2Id,
        player2Id,
        reason: reason || undefined,
      },
      {
        onSuccess: () => setOpen(false),
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <ArrowLeftRight className="mr-2 h-4 w-4" />
          Spieler tauschen
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Spieler zwischen Paarungen tauschen</DialogTitle>
          <DialogDescription>
            Wählen Sie zwei Partien und jeweils einen Spieler zum Tauschen.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="swap-game1">Partie 1</Label>
              <Select value={game1Id} onValueChange={setGame1Id}>
                <SelectTrigger id="swap-game1">
                  <SelectValue placeholder="Partie wählen" />
                </SelectTrigger>
                <SelectContent>
                  {availableGames
                    .filter((g) => g.id !== game2Id)
                    .map((g) => (
                      <SelectItem key={g.id} value={g.id}>
                        Brett {g.boardNumber}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              {game1Id && (
                <Select value={player1Id} onValueChange={setPlayer1Id}>
                  <SelectTrigger>
                    <SelectValue placeholder="Spieler wählen" />
                  </SelectTrigger>
                  <SelectContent>
                    {game1Players.map((p) => (
                      <SelectItem key={p.id} value={p.id!}>
                        {p.name} ({p.color})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="swap-game2">Partie 2</Label>
              <Select value={game2Id} onValueChange={setGame2Id}>
                <SelectTrigger id="swap-game2">
                  <SelectValue placeholder="Partie wählen" />
                </SelectTrigger>
                <SelectContent>
                  {availableGames
                    .filter((g) => g.id !== game1Id)
                    .map((g) => (
                      <SelectItem key={g.id} value={g.id}>
                        Brett {g.boardNumber}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              {game2Id && (
                <Select value={player2Id} onValueChange={setPlayer2Id}>
                  <SelectTrigger>
                    <SelectValue placeholder="Spieler wählen" />
                  </SelectTrigger>
                  <SelectContent>
                    {game2Players.map((p) => (
                      <SelectItem key={p.id} value={p.id!}>
                        {p.name} ({p.color})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="swap-reason">Begründung (optional)</Label>
            <Textarea
              id="swap-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für den Spielertausch"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!game1Id || !player1Id || !game2Id || !player2Id || mutation.isPending}
          >
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Spieler tauschen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
