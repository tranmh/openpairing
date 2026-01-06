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
import { Edit2, Loader2 } from 'lucide-react';
import { useEnterResult } from '../hooks/useArbiterMutations';
import type { ResultEntryDialogProps } from '../types';

/**
 * Dialog for entering game results
 *
 * BUG-1 FIX: Reset result state when dialog closes
 */
export function ResultEntryDialog({
  tournamentId,
  gameId,
  boardNumber,
  whiteName,
  blackName,
}: ResultEntryDialogProps) {
  const [open, setOpen] = useState(false);
  const [result, setResult] = useState('');
  const mutation = useEnterResult(tournamentId);

  // BUG-1 FIX: Reset state when dialog closes
  useEffect(() => {
    if (!open) {
      setResult('');
    }
  }, [open]);

  const handleSubmit = () => {
    mutation.mutate(
      { gameId, result },
      {
        onSuccess: () => setOpen(false),
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" aria-label="Ergebnis eintragen">
          <Edit2 className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Ergebnis eintragen</DialogTitle>
          <DialogDescription>
            Brett {boardNumber}: {whiteName} vs {blackName}
          </DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-3 gap-2 py-4">
          {['1-0', '1/2-1/2', '0-1'].map((r) => (
            <Button
              key={r}
              variant={result === r ? 'default' : 'outline'}
              onClick={() => setResult(r)}
              className="text-lg"
            >
              {r === '1/2-1/2' ? '½-½' : r}
            </Button>
          ))}
        </div>
        <div className="grid grid-cols-3 gap-2">
          {['1-0F', '0-0F', '0-1F'].map((r) => (
            <Button
              key={r}
              variant={result === r ? 'secondary' : 'ghost'}
              onClick={() => setResult(r)}
              size="sm"
            >
              {r} (k)
            </Button>
          ))}
        </div>
        <DialogFooter className="mt-4">
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={!result || mutation.isPending}>
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Speichern
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
