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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Plus, Loader2 } from 'lucide-react';
import { useCreateManualPairing } from '../hooks/useArbiterMutations';
import type { CreateManualPairingDialogProps } from '../types';

/**
 * Dialog for creating a manual pairing
 */
export function CreateManualPairingDialog({
  tournamentId,
  roundNumber,
  participants,
  existingPairings,
}: CreateManualPairingDialogProps) {
  const [open, setOpen] = useState(false);
  const [whiteParticipantId, setWhiteParticipantId] = useState('');
  const [blackParticipantId, setBlackParticipantId] = useState('');
  const [boardNumber, setBoardNumber] = useState('');
  const mutation = useCreateManualPairing(tournamentId);

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      setWhiteParticipantId('');
      setBlackParticipantId('');
      setBoardNumber('');
    }
  }, [open]);

  // Find participants already paired in this round
  const pairedParticipantIds = useMemo(() => {
    const ids = new Set<string>();
    existingPairings.forEach((p) => {
      if (p.whiteParticipantId) ids.add(p.whiteParticipantId);
      if (p.blackParticipantId) ids.add(p.blackParticipantId);
    });
    return ids;
  }, [existingPairings]);

  // Active participants not yet paired
  const availableParticipants = useMemo(
    () => participants.filter((p) => p.isActive && !pairedParticipantIds.has(p.id)),
    [participants, pairedParticipantIds]
  );

  const handleSubmit = () => {
    mutation.mutate(
      {
        roundNumber,
        whiteParticipantId,
        blackParticipantId: blackParticipantId === 'BYE' ? null : blackParticipantId,
        boardNumber: boardNumber ? parseInt(boardNumber) : undefined,
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
          <Plus className="mr-2 h-4 w-4" />
          Neue Paarung
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Manuelle Paarung erstellen</DialogTitle>
          <DialogDescription>
            Erstellen Sie eine neue Paarung für Runde {roundNumber}.
            {availableParticipants.length === 0 && (
              <span className="text-destructive block mt-1">
                Alle Teilnehmer sind bereits gepaart.
              </span>
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="manual-white">Weiß</Label>
            <Select value={whiteParticipantId} onValueChange={setWhiteParticipantId}>
              <SelectTrigger id="manual-white">
                <SelectValue placeholder="Spieler mit Weiß" />
              </SelectTrigger>
              <SelectContent>
                {availableParticipants
                  .filter((p) => p.id !== blackParticipantId)
                  .map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.startRank}. {p.playerName || p.playerId} ({p.startingRating})
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="manual-black">Schwarz</Label>
            <Select value={blackParticipantId} onValueChange={setBlackParticipantId}>
              <SelectTrigger id="manual-black">
                <SelectValue placeholder="Spieler mit Schwarz oder Spielfrei" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="BYE">
                  <span className="italic">Spielfrei (BYE)</span>
                </SelectItem>
                {availableParticipants
                  .filter((p) => p.id !== whiteParticipantId)
                  .map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.startRank}. {p.playerName || p.playerId} ({p.startingRating})
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="manual-board">Brettnummer (optional)</Label>
            <Input
              id="manual-board"
              type="number"
              min="1"
              value={boardNumber}
              onChange={(e) => setBoardNumber(e.target.value)}
              placeholder="Automatisch zuweisen"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!whiteParticipantId || !blackParticipantId || mutation.isPending}
          >
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Paarung erstellen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
