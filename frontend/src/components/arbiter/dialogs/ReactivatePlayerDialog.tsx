'use client';

import { useState, useEffect } from 'react';
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
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { UserPlus, Loader2 } from 'lucide-react';
import { useReactivatePlayer } from '../hooks/useArbiterMutations';
import type { ReactivatePlayerDialogProps } from '../types';

/**
 * Dialog for reactivating a withdrawn player
 */
export function ReactivatePlayerDialog({
  tournamentId,
  participantId,
  playerName,
}: ReactivatePlayerDialogProps) {
  const [open, setOpen] = useState(false);
  const [missedRoundPoints, setMissedRoundPoints] = useState('0');
  const [reason, setReason] = useState('');
  const mutation = useReactivatePlayer(tournamentId);

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      setMissedRoundPoints('0');
      setReason('');
    }
  }, [open]);

  const handleSubmit = () => {
    mutation.mutate(
      {
        participantId,
        missedRoundPoints: parseFloat(missedRoundPoints) || 0,
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
        <Button variant="ghost" size="sm" aria-label="Spieler reaktivieren">
          <UserPlus className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Spieler reaktivieren</DialogTitle>
          <DialogDescription>
            {playerName} wird wieder am Turnier teilnehmen.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="missed-points">Punkte für verpasste Runden</Label>
            <Input
              id="missed-points"
              type="number"
              step="0.5"
              min="0"
              value={missedRoundPoints}
              onChange={(e) => setMissedRoundPoints(e.target.value)}
              placeholder="0"
            />
            <p className="text-xs text-muted-foreground">
              Optional: Punkte, die für verpasste Runden gutgeschrieben werden
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="reactivate-reason">Begründung</Label>
            <Textarea
              id="reactivate-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für die Reaktivierung (optional)"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={mutation.isPending}>
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Reaktivieren
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
