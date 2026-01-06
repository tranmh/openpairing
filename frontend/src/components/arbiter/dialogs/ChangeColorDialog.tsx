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
import { ArrowLeftRight, Loader2 } from 'lucide-react';
import { useChangeColor } from '../hooks/useArbiterMutations';
import type { ChangeColorDialogProps } from '../types';

/**
 * Dialog for swapping player colors in a game
 */
export function ChangeColorDialog({
  tournamentId,
  gameId,
  whiteName,
  blackName,
}: ChangeColorDialogProps) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const mutation = useChangeColor(tournamentId);

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
        <Button variant="ghost" size="sm" aria-label="Farben tauschen">
          <ArrowLeftRight className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Farben tauschen</DialogTitle>
          <DialogDescription>
            {whiteName} und {blackName} tauschen die Farben.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="reason">Begründung (optional)</Label>
            <Textarea
              id="reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für den Farbwechsel"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={mutation.isPending}>
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Farben tauschen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
