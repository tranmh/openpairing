'use client';

import { useState, useEffect } from 'react';
import { useMutation } from '@tanstack/react-query';
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
import { Copy, Loader2 } from 'lucide-react';
import { tournamentsApi } from '@/lib/api';
import { toast } from 'sonner';
import type { CloneTournamentDialogProps } from '../types';

/**
 * Dialog for cloning a tournament
 */
export function CloneTournamentDialog({
  tournamentId,
  tournamentName,
}: CloneTournamentDialogProps) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState(`Kopie von ${tournamentName}`);
  const [description, setDescription] = useState('');
  const [startDate, setStartDate] = useState('');

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      setName(`Kopie von ${tournamentName}`);
      setDescription('');
      setStartDate('');
    }
  }, [open, tournamentName]);

  const mutation = useMutation({
    mutationFn: async () => {
      const response = await tournamentsApi.cloneTournament(tournamentId, {
        name,
        description: description || undefined,
        startDate: startDate || undefined,
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: (data) => {
      toast.success('Turnier wurde geklont');
      setOpen(false);
      if (data?.tournament?.id) {
        window.open(`/tournaments/${data.tournament.id}`, '_blank');
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Klonen');
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Copy className="mr-2 h-4 w-4" />
          Turnier klonen
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Turnier klonen</DialogTitle>
          <DialogDescription>
            Erstellen Sie eine Kopie des Turniers mit allen Einstellungen. Teilnehmer werden nicht
            kopiert.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="clone-name">Name des neuen Turniers</Label>
            <Input
              id="clone-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Turniername"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="clone-description">Beschreibung (optional)</Label>
            <Textarea
              id="clone-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Beschreibung des neuen Turniers"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="clone-date">Startdatum (optional)</Label>
            <Input
              id="clone-date"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Standard: 1 Jahr nach dem Originalturnier
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button onClick={() => mutation.mutate()} disabled={!name || mutation.isPending}>
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Klonen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
