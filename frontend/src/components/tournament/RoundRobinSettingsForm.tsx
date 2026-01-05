'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { tournamentsApi } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2, Save, Info } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { RoundRobinSettings, UpdateRoundRobinSettingsRequest } from '@/types/roundrobin';

interface RoundRobinSettingsFormProps {
  tournamentId: string;
  initialSettings?: RoundRobinSettings;
  numPlayers: number;
  onSave?: (settings: RoundRobinSettings) => void;
  className?: string;
}

export function RoundRobinSettingsForm({
  tournamentId,
  initialSettings,
  numPlayers,
  onSave,
  className,
}: RoundRobinSettingsFormProps) {
  const queryClient = useQueryClient();

  const defaultSettings: RoundRobinSettings = {
    byePosition: 0,
    fixedPlayer: 1,
    showSeatingChart: true,
  };

  const [settings, setSettings] = useState<RoundRobinSettings>(
    initialSettings || defaultSettings
  );

  const mutation = useMutation({
    mutationFn: async (data: UpdateRoundRobinSettingsRequest) => {
      const response = await tournamentsApi.updateRoundRobinSettings(
        tournamentId,
        data
      );
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data?.settings;
    },
    onSuccess: (newSettings) => {
      queryClient.invalidateQueries({ queryKey: ['tournament', tournamentId] });
      queryClient.invalidateQueries({
        queryKey: ['seatingChart', tournamentId],
      });
      if (newSettings) {
        onSave?.(newSettings);
      }
    },
  });

  const handleSave = () => {
    mutation.mutate({
      byePosition: settings.byePosition,
      fixedPlayer: settings.fixedPlayer,
      showSeatingChart: settings.showSeatingChart,
    });
  };

  // Calculate bye position options
  const numSeats = numPlayers % 2 === 1 ? numPlayers + 1 : numPlayers;
  const defaultByePosition = Math.floor(numSeats / 2) + 1;

  // Generate player options for fixed player
  const playerOptions = Array.from({ length: numPlayers }, (_, i) => i + 1);

  // Generate bye position options (only relevant for odd number of players)
  const byePositionOptions = Array.from({ length: numSeats }, (_, i) => i + 1);

  const isOddPlayers = numPlayers % 2 === 1;

  return (
    <div className={cn('space-y-6', className)}>
      <div className="space-y-4">
        {/* Show Seating Chart Toggle */}
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label htmlFor="showSeatingChart">Sitzplan anzeigen</Label>
            <p className="text-sm text-muted-foreground">
              Zeigt den Sitzplan für Spieler und Publikum
            </p>
          </div>
          <Switch
            id="showSeatingChart"
            checked={settings.showSeatingChart}
            onCheckedChange={(checked: boolean) =>
              setSettings({ ...settings, showSeatingChart: checked })
            }
          />
        </div>

        {/* Fixed Player Selection */}
        <div className="space-y-2">
          <Label htmlFor="fixedPlayer">Fixierter Spieler</Label>
          <p className="text-sm text-muted-foreground">
            Dieser Spieler bleibt an seinem Platz, alle anderen rutschen
          </p>
          <Select
            value={String(settings.fixedPlayer)}
            onValueChange={(v) =>
              setSettings({ ...settings, fixedPlayer: Number(v) })
            }
          >
            <SelectTrigger id="fixedPlayer" className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {playerOptions.map((num) => (
                <SelectItem key={num} value={String(num)}>
                  Spieler #{num}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Bye Position Selection (only for odd number of players) */}
        {isOddPlayers && (
          <div className="space-y-2">
            <Label htmlFor="byePosition">Spielfrei-Position</Label>
            <p className="text-sm text-muted-foreground">
              Welcher Platz im Kreis ist spielfrei (Standard: gegenüber vom
              fixierten Spieler)
            </p>
            <Select
              value={String(settings.byePosition)}
              onValueChange={(v) =>
                setSettings({ ...settings, byePosition: Number(v) })
              }
            >
              <SelectTrigger id="byePosition" className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">
                  Automatisch (Platz {defaultByePosition})
                </SelectItem>
                {byePositionOptions.map((num) => (
                  <SelectItem key={num} value={String(num)}>
                    Platz {num}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

      {/* Info about rotation */}
      <Alert>
        <Info className="h-4 w-4" />
        <AlertDescription>
          Im Rutsch-System sitzt Spieler #{settings.fixedPlayer} immer am
          gleichen Platz. Alle anderen Spieler rutschen nach jeder Runde einen
          Platz im Uhrzeigersinn weiter.
          {isOddPlayers && (
            <>
              {' '}
              Bei {numPlayers} Spielern gibt es einen Platz für Spielfrei
              (Position{' '}
              {settings.byePosition === 0
                ? defaultByePosition
                : settings.byePosition}
              ).
            </>
          )}
        </AlertDescription>
      </Alert>

      {/* Error message */}
      {mutation.isError && (
        <Alert variant="destructive">
          <AlertDescription>
            {mutation.error instanceof Error
              ? mutation.error.message
              : 'Fehler beim Speichern'}
          </AlertDescription>
        </Alert>
      )}

      {/* Save button */}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={mutation.isPending}>
          {mutation.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Speichern...
            </>
          ) : (
            <>
              <Save className="mr-2 h-4 w-4" />
              Speichern
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
