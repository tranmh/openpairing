'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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
import { UserPlus, Search, Loader2 } from 'lucide-react';
import { useDebounce } from 'use-debounce';
import { federationApi, tournamentsApi, type FederationPlayer } from '@/lib/api';
import { toast } from 'sonner';
import { arbiterStatusKey } from '../hooks/useArbiterStatus';
import type { LateEntryDialogProps } from '../types';

/**
 * Dialog for adding a late entry player to the tournament
 */
export function LateEntryDialog({
  tournamentId,
  currentRoundNumber,
  tournamentRounds: _tournamentRounds,
}: LateEntryDialogProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [debouncedSearch] = useDebounce(search, 300);
  const [selectedPlayer, setSelectedPlayer] = useState<FederationPlayer | null>(null);
  const [entryRound, setEntryRound] = useState(String(Math.max(2, currentRoundNumber)));
  const [startingRating, setStartingRating] = useState('');
  const queryClient = useQueryClient();

  const { data: searchResults, isLoading: searchLoading } = useQuery({
    queryKey: ['federation', 'search', debouncedSearch],
    queryFn: async () => {
      const response = await federationApi.search(debouncedSearch, 'dsb', 10);
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    enabled: debouncedSearch.length >= 2,
  });

  const mutation = useMutation({
    mutationFn: async () => {
      if (!selectedPlayer) throw new Error('Kein Spieler ausgewählt');
      const response = await tournamentsApi.addLateEntry(tournamentId, {
        dsbId: selectedPlayer.id,
        entryRound: parseInt(entryRound),
        startingRating: startingRating ? parseInt(startingRating) : undefined,
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: arbiterStatusKey(tournamentId) });
      toast.success('Nachnennung erfolgreich');
      setOpen(false);
      resetForm();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler bei der Nachnennung');
    },
  });

  const resetForm = () => {
    setSearch('');
    setSelectedPlayer(null);
    setEntryRound(String(Math.max(2, currentRoundNumber)));
    setStartingRating('');
  };

  const handleSelectPlayer = (player: FederationPlayer) => {
    setSelectedPlayer(player);
    setSearch(`${player.firstName || ''} ${player.name}`.trim());
    setStartingRating(String(player.rating || player.fideRating || ''));
  };

  const handleOpenChange = (isOpen: boolean) => {
    setOpen(isOpen);
    if (!isOpen) resetForm();
  };

  // Generate available entry rounds (from 2 to current round)
  const availableRounds = currentRoundNumber >= 2
    ? Array.from({ length: currentRoundNumber - 1 }, (_, i) => i + 2)
    : [];

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <UserPlus className="mr-2 h-4 w-4" />
          Nachnennung
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Nachnennung</DialogTitle>
          <DialogDescription>
            Fügen Sie einen Spieler nachträglich zum laufenden Turnier hinzu.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="player-search">Spieler suchen (Portal64)</Label>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                id="player-search"
                className="pl-10"
                placeholder="Name eingeben..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  if (selectedPlayer) setSelectedPlayer(null);
                }}
              />
            </div>
            {searchLoading && <p className="text-sm text-muted-foreground">Suche...</p>}
            {!selectedPlayer && searchResults?.results && searchResults.results.length > 0 && (
              <div className="border rounded-md max-h-48 overflow-y-auto">
                {searchResults.results.map((player) => (
                  <button
                    key={player.id}
                    type="button"
                    className="w-full px-3 py-2 text-left hover:bg-muted text-sm border-b last:border-b-0"
                    onClick={() => handleSelectPlayer(player)}
                  >
                    <div className="font-medium">
                      {player.firstName} {player.name}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {player.club && `${player.club} · `}
                      DWZ: {player.rating || '-'}
                      {player.fideRating && ` · FIDE: ${player.fideRating}`}
                    </div>
                  </button>
                ))}
              </div>
            )}
            {selectedPlayer && (
              <div className="p-3 bg-muted rounded-md">
                <div className="font-medium">
                  {selectedPlayer.firstName} {selectedPlayer.name}
                </div>
                <div className="text-sm text-muted-foreground">
                  {selectedPlayer.club && `${selectedPlayer.club} · `}
                  DWZ: {selectedPlayer.rating || '-'}
                </div>
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="entry-round">Einstiegsrunde</Label>
            {currentRoundNumber < 2 ? (
              <p className="text-sm text-muted-foreground py-2">
                Nachnennung erst ab Runde 2 möglich.
              </p>
            ) : (
              <>
                <Select value={entryRound} onValueChange={setEntryRound}>
                  <SelectTrigger id="entry-round">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {availableRounds.map((r) => (
                      <SelectItem key={r} value={String(r)}>
                        Runde {r}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Für Runden vor dem Einstieg werden Freilose vergeben.
                </p>
              </>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="starting-rating">Startrating</Label>
            <Input
              id="starting-rating"
              type="number"
              min="0"
              max="3000"
              value={startingRating}
              onChange={(e) => setStartingRating(e.target.value)}
              placeholder="Wird aus Portal64 übernommen"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!selectedPlayer || currentRoundNumber < 2 || mutation.isPending}
          >
            {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Hinzufügen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
