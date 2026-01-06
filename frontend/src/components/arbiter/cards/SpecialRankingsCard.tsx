'use client';

import { useState } from 'react';
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
import { Trophy, Plus, Edit2, Trash2, Loader2 } from 'lucide-react';
import { tournamentsApi, type SpecialRanking } from '@/lib/api';
import { toast } from 'sonner';
import type { SpecialRankingsCardProps } from '../types';

/**
 * Card for managing special rankings (age groups, rating classes, etc.)
 */
export function SpecialRankingsCard({ tournamentId }: SpecialRankingsCardProps) {
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [editingRanking, setEditingRanking] = useState<SpecialRanking | null>(null);

  // Form state
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [minBirthYear, setMinBirthYear] = useState('');
  const [maxBirthYear, setMaxBirthYear] = useState('');
  const [minRating, setMinRating] = useState('');
  const [maxRating, setMaxRating] = useState('');
  const [gender, setGender] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['special-rankings', tournamentId],
    queryFn: async () => {
      const response = await tournamentsApi.getSpecialRankings(tournamentId);
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
  });

  const resetForm = () => {
    setName('');
    setDisplayName('');
    setMinBirthYear('');
    setMaxBirthYear('');
    setMinRating('');
    setMaxRating('');
    setGender('');
    setEditingRanking(null);
  };

  const openCreateDialog = () => {
    resetForm();
    setEditOpen(true);
  };

  const openEditDialog = (ranking: SpecialRanking) => {
    setEditingRanking(ranking);
    setName(ranking.name);
    setDisplayName(ranking.displayName);
    setMinBirthYear(ranking.minBirthYear?.toString() || '');
    setMaxBirthYear(ranking.maxBirthYear?.toString() || '');
    setMinRating(ranking.minRating?.toString() || '');
    setMaxRating(ranking.maxRating?.toString() || '');
    setGender(ranking.gender || '');
    setEditOpen(true);
  };

  const createMutation = useMutation({
    mutationFn: async () => {
      const response = await tournamentsApi.createSpecialRanking(tournamentId, {
        name,
        displayName,
        minBirthYear: minBirthYear ? parseInt(minBirthYear) : undefined,
        maxBirthYear: maxBirthYear ? parseInt(maxBirthYear) : undefined,
        minRating: minRating ? parseInt(minRating) : undefined,
        maxRating: maxRating ? parseInt(maxRating) : undefined,
        gender: gender || undefined,
        sortOrder: (data?.rankings?.length || 0) + 1,
      });
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['special-rankings', tournamentId] });
      toast.success('Sonderwertung erstellt');
      setEditOpen(false);
      resetForm();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Erstellen');
    },
  });

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!editingRanking) throw new Error('Keine Wertung ausgewählt');
      const response = await tournamentsApi.updateSpecialRanking(
        tournamentId,
        editingRanking.id,
        {
          name,
          displayName,
          minBirthYear: minBirthYear ? parseInt(minBirthYear) : undefined,
          maxBirthYear: maxBirthYear ? parseInt(maxBirthYear) : undefined,
          minRating: minRating ? parseInt(minRating) : undefined,
          maxRating: maxRating ? parseInt(maxRating) : undefined,
          gender: gender || undefined,
        }
      );
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['special-rankings', tournamentId] });
      toast.success('Sonderwertung aktualisiert');
      setEditOpen(false);
      resetForm();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Aktualisieren');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (rankingId: string) => {
      const response = await tournamentsApi.deleteSpecialRanking(tournamentId, rankingId);
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['special-rankings', tournamentId] });
      toast.success('Sonderwertung gelöscht');
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Löschen');
    },
  });

  const rankings = data?.rankings || [];

  const getFilterDescription = (ranking: SpecialRanking) => {
    const parts: string[] = [];
    if (ranking.minBirthYear || ranking.maxBirthYear) {
      if (ranking.minBirthYear && ranking.maxBirthYear) {
        parts.push(`Jahrgang ${ranking.minBirthYear}-${ranking.maxBirthYear}`);
      } else if (ranking.minBirthYear) {
        parts.push(`Ab Jahrgang ${ranking.minBirthYear}`);
      } else {
        parts.push(`Bis Jahrgang ${ranking.maxBirthYear}`);
      }
    }
    if (ranking.minRating || ranking.maxRating) {
      if (ranking.minRating && ranking.maxRating) {
        parts.push(`Rating ${ranking.minRating}-${ranking.maxRating}`);
      } else if (ranking.minRating) {
        parts.push(`Rating ab ${ranking.minRating}`);
      } else {
        parts.push(`Rating bis ${ranking.maxRating}`);
      }
    }
    if (ranking.gender) {
      parts.push(ranking.gender === 'female' ? 'Frauen' : 'Männer');
    }
    return parts.length > 0 ? parts.join(', ') : 'Alle Teilnehmer';
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Trophy className="h-5 w-5" />
              Sonderwertungen
            </CardTitle>
            <CardDescription>Separate Ranglisten für bestimmte Spielergruppen</CardDescription>
          </div>
          <Button size="sm" onClick={openCreateDialog}>
            <Plus className="mr-2 h-4 w-4" />
            Neu
          </Button>
          <Dialog
            open={editOpen}
            onOpenChange={(open) => {
              setEditOpen(open);
              if (!open) resetForm();
            }}
          >
            <DialogContent>
              <DialogHeader>
                <DialogTitle>
                  {editingRanking ? 'Sonderwertung bearbeiten' : 'Neue Sonderwertung'}
                </DialogTitle>
                <DialogDescription>
                  Definieren Sie Filterkriterien für die Sonderwertung.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="sr-name">Interner Name</Label>
                    <Input
                      id="sr-name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="z.B. u14"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="sr-display">Anzeigename</Label>
                    <Input
                      id="sr-display"
                      value={displayName}
                      onChange={(e) => setDisplayName(e.target.value)}
                      placeholder="z.B. U14 Wertung"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="sr-birth-min">Jahrgang von</Label>
                    <Input
                      id="sr-birth-min"
                      type="number"
                      value={minBirthYear}
                      onChange={(e) => setMinBirthYear(e.target.value)}
                      placeholder="z.B. 2010"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="sr-birth-max">Jahrgang bis</Label>
                    <Input
                      id="sr-birth-max"
                      type="number"
                      value={maxBirthYear}
                      onChange={(e) => setMaxBirthYear(e.target.value)}
                      placeholder="z.B. 2014"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="sr-rating-min">Rating von</Label>
                    <Input
                      id="sr-rating-min"
                      type="number"
                      value={minRating}
                      onChange={(e) => setMinRating(e.target.value)}
                      placeholder="z.B. 1000"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="sr-rating-max">Rating bis</Label>
                    <Input
                      id="sr-rating-max"
                      type="number"
                      value={maxRating}
                      onChange={(e) => setMaxRating(e.target.value)}
                      placeholder="z.B. 1600"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="sr-gender">Geschlecht</Label>
                  <Select
                    value={gender || 'all'}
                    onValueChange={(v) => setGender(v === 'all' ? '' : v)}
                  >
                    <SelectTrigger id="sr-gender">
                      <SelectValue placeholder="Alle" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">Alle</SelectItem>
                      <SelectItem value="female">Frauen</SelectItem>
                      <SelectItem value="male">Männer</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setEditOpen(false)}>
                  Abbrechen
                </Button>
                <Button
                  onClick={() => (editingRanking ? updateMutation.mutate() : createMutation.mutate())}
                  disabled={!name || !displayName || isPending}
                >
                  {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {editingRanking ? 'Speichern' : 'Erstellen'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-20" />
        ) : rankings.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            Keine Sonderwertungen definiert
          </p>
        ) : (
          <div className="space-y-2">
            {rankings.map((ranking) => (
              <div key={ranking.id} className="flex items-center justify-between p-2 border rounded-md">
                <div>
                  <span className="font-medium">{ranking.displayName}</span>
                  <span className="ml-2 text-xs text-muted-foreground">
                    {getFilterDescription(ranking)}
                  </span>
                </div>
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => openEditDialog(ranking)}
                    aria-label="Bearbeiten"
                  >
                    <Edit2 className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => deleteMutation.mutate(ranking.id)}
                    disabled={deleteMutation.isPending}
                    aria-label="Löschen"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
