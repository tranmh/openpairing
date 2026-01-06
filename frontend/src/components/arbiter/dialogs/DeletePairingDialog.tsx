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
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Trash2, Loader2 } from 'lucide-react';
import { useDeletePairing } from '../hooks/useArbiterMutations';
import type { DeletePairingDialogProps } from '../types';

/**
 * Dialog for deleting a pairing
 */
export function DeletePairingDialog({
  tournamentId,
  gameId,
  boardNumber,
}: DeletePairingDialogProps) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const mutation = useDeletePairing(tournamentId);

  // Reset state when dialog closes
  useEffect(() => {
    if (!open) {
      setReason('');
    }
  }, [open]);

  const handleSubmit = () => {
    mutation.mutate(
      { gameId, reason },
      {
        onSuccess: () => setOpen(false),
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" aria-label="Paarung löschen">
          <Trash2 className="h-4 w-4 text-destructive" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Paarung löschen</DialogTitle>
          <DialogDescription>
            Die Paarung auf Brett {boardNumber} wird gelöscht.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="delete-reason">Begründung (optional)</Label>
            <Textarea
              id="delete-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für das Löschen"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button variant="destructive" onClick={handleSubmit} disabled={mutation.isPending}>
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Löschen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
