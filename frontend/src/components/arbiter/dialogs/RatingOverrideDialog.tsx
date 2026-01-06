'use client';

import { useState, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
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
import { Label } from '@/components/ui/label';
import { Star, Loader2 } from 'lucide-react';
import { tournamentsApi } from '@/lib/api';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';
import { arbiterStatusKey } from '../hooks/useArbiterStatus';
import type { RatingOverrideDialogProps } from '../types';

/**
 * Dialog for setting or clearing a rating override for a participant
 *
 * BUG-2 FIX: Now properly receives hasOverride prop from parent
 * BUG-10 FIX: Added proper validation for rating input
 */
export function RatingOverrideDialog({
  tournamentId,
  participantId,
  playerName,
  currentRating,
  hasOverride,
}: RatingOverrideDialogProps) {
  const [open, setOpen] = useState(false);
  const [rating, setRating] = useState(String(currentRating));
  const [validationError, setValidationError] = useState('');
  const queryClient = useQueryClient();

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setRating(String(currentRating));
      setValidationError('');
    }
  }, [open, currentRating]);

  // Validate rating input
  const validateRating = (value: string): boolean => {
    const num = parseInt(value);
    if (isNaN(num)) {
      setValidationError('Bitte geben Sie eine gültige Zahl ein');
      return false;
    }
    if (num < 0) {
      setValidationError('Rating kann nicht negativ sein');
      return false;
    }
    if (num > 3000) {
      setValidationError('Rating kann nicht über 3000 sein');
      return false;
    }
    setValidationError('');
    return true;
  };

  const handleRatingChange = (value: string) => {
    setRating(value);
    if (value) validateRating(value);
    else setValidationError('');
  };

  const setMutation = useMutation({
    mutationFn: async () => {
      if (!validateRating(rating)) {
        throw new Error(validationError || 'Ungültiger Rating-Wert');
      }
      const response = await tournamentsApi.setRatingOverride(
        tournamentId,
        participantId,
        parseInt(rating)
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: arbiterStatusKey(tournamentId) });
      toast.success('Rating überschrieben');
      setOpen(false);
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Überschreiben');
    },
  });

  const clearMutation = useMutation({
    mutationFn: async () => {
      const response = await tournamentsApi.clearRatingOverride(tournamentId, participantId);
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: arbiterStatusKey(tournamentId) });
      toast.success('Rating-Überschreibung entfernt');
      setOpen(false);
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Entfernen');
    },
  });

  const isPending = setMutation.isPending || clearMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          aria-label="Rating überschreiben"
          className={cn(hasOverride && 'text-orange-500')}
        >
          <Star className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rating überschreiben</DialogTitle>
          <DialogDescription>
            {playerName} - Aktuelles Rating: {currentRating}
            {hasOverride && ' (überschrieben)'}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="new-rating">Neues Rating</Label>
            <Input
              id="new-rating"
              type="number"
              min="0"
              max="3000"
              step="1"
              value={rating}
              onChange={(e) => handleRatingChange(e.target.value)}
              className={cn(validationError && 'border-destructive')}
            />
            {validationError ? (
              <p className="text-xs text-destructive">{validationError}</p>
            ) : (
              <p className="text-xs text-muted-foreground">
                Dieses Rating wird für die Paarungen verwendet.
              </p>
            )}
          </div>
        </div>
        <DialogFooter className="flex-col sm:flex-row gap-2">
          {hasOverride && (
            <Button
              variant="outline"
              onClick={() => clearMutation.mutate()}
              disabled={isPending}
              className="sm:mr-auto"
            >
              {clearMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Zurücksetzen
            </Button>
          )}
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={() => setMutation.mutate()}
            disabled={!rating || !!validationError || isPending}
          >
            {setMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Speichern
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
