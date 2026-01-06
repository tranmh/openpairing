'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import { Skeleton } from '@/components/ui/skeleton';
import { Ban, Plus, Trash2, Loader2 } from 'lucide-react';
import { tournamentsApi } from '@/lib/api';
import { toast } from 'sonner';
import type { ForbiddenPairsCardProps } from '../types';

/**
 * Card for managing forbidden player pairs
 */
export function ForbiddenPairsCard({ tournamentId, participants }: ForbiddenPairsCardProps) {
  const queryClient = useQueryClient();
  const [addOpen, setAddOpen] = useState(false);
  const [player1, setPlayer1] = useState('');
  const [player2, setPlayer2] = useState('');
  const [reason, setReason] = useState('');

  // Reset form when dialog closes
  useEffect(() => {
    if (!addOpen) {
      setPlayer1('');
      setPlayer2('');
      setReason('');
    }
  }, [addOpen]);

  const { data, isLoading } = useQuery({
    queryKey: ['forbidden-pairs', tournamentId],
    queryFn: async () => {
      const response = await tournamentsApi.getForbiddenPairs(tournamentId);
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
  });

  const addMutation = useMutation({
    mutationFn: async () => {
      if (!player1 || !player2) throw new Error('Beide Spieler auswählen');
      if (player1 === player2) throw new Error('Spieler müssen unterschiedlich sein');
      const response = await tournamentsApi.addForbiddenPair(tournamentId, {
        player1Id: player1,
        player2Id: player2,
        reason: reason || undefined,
      });
      if (response.error) throw new Error(response.error.message);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['forbidden-pairs', tournamentId] });
      toast.success('Verbotene Paarung hinzugefügt');
      setAddOpen(false);
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Hinzufügen');
    },
  });

  const removeMutation = useMutation({
    mutationFn: async (pairId: string) => {
      const response = await tournamentsApi.removeForbiddenPair(tournamentId, pairId);
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['forbidden-pairs', tournamentId] });
      toast.success('Verbotene Paarung entfernt');
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Entfernen');
    },
  });

  const forbiddenPairs = data?.forbiddenPairs || [];
  const activeParticipants = participants.filter((p) => p.isActive);

  const getParticipantName = (id: string) => {
    const p = participants.find((p) => p.id === id || p.playerId === id);
    return p?.playerName || p?.playerId || id;
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Ban className="h-5 w-5" />
              Verbotene Paarungen
            </CardTitle>
            <CardDescription>
              Diese Spieler dürfen nicht gegeneinander gepaart werden
            </CardDescription>
          </div>
          <Dialog open={addOpen} onOpenChange={setAddOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="mr-2 h-4 w-4" />
                Hinzufügen
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Verbotene Paarung hinzufügen</DialogTitle>
                <DialogDescription>
                  Wählen Sie zwei Spieler, die nicht gegeneinander gepaart werden sollen.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="fp-player1">Spieler 1</Label>
                  <Select value={player1} onValueChange={setPlayer1}>
                    <SelectTrigger id="fp-player1">
                      <SelectValue placeholder="Spieler auswählen" />
                    </SelectTrigger>
                    <SelectContent>
                      {activeParticipants.map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          {p.playerName || p.playerId} ({p.startingRating})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="fp-player2">Spieler 2</Label>
                  <Select value={player2} onValueChange={setPlayer2}>
                    <SelectTrigger id="fp-player2">
                      <SelectValue placeholder="Spieler auswählen" />
                    </SelectTrigger>
                    <SelectContent>
                      {activeParticipants
                        .filter((p) => p.id !== player1)
                        .map((p) => (
                          <SelectItem key={p.id} value={p.id}>
                            {p.playerName || p.playerId} ({p.startingRating})
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="fp-reason">Grund (optional)</Label>
                  <Input
                    id="fp-reason"
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    placeholder="z.B. Teammitglieder, Verwandte"
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setAddOpen(false)}>
                  Abbrechen
                </Button>
                <Button
                  onClick={() => addMutation.mutate()}
                  disabled={!player1 || !player2 || addMutation.isPending}
                >
                  {addMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Hinzufügen
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-20" />
        ) : forbiddenPairs.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            Keine verbotenen Paarungen definiert
          </p>
        ) : (
          <div className="space-y-2">
            {forbiddenPairs.map((pair) => (
              <div
                key={pair.id}
                className="flex items-center justify-between p-2 border rounded-md"
              >
                <div>
                  <span className="font-medium">
                    {pair.player1Name || getParticipantName(pair.player1Id)}
                  </span>
                  <span className="mx-2 text-muted-foreground">vs</span>
                  <span className="font-medium">
                    {pair.player2Name || getParticipantName(pair.player2Id)}
                  </span>
                  {pair.reason && (
                    <span className="ml-2 text-xs text-muted-foreground">({pair.reason})</span>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => removeMutation.mutate(pair.id)}
                  disabled={removeMutation.isPending}
                  aria-label="Paarung entfernen"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
