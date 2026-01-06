'use client';

import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Shuffle, PlayCircle, CheckCircle2, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  useGeneratePairings,
  useStartRound,
  useCompleteRound,
} from '../hooks/useArbiterMutations';
import type { RoundControlTabProps } from '../types';
import type { RoundStatus } from '@/types/arbiter';

/**
 * Tab for controlling round lifecycle (generate pairings, start, complete)
 */
export function RoundControlTab({
  tournamentId,
  rounds,
  currentRound,
  pendingResults,
}: RoundControlTabProps) {
  const [selectedRound, setSelectedRound] = useState<string>(
    currentRound?.roundNumber.toString() || '1'
  );

  const generatePairingsMutation = useGeneratePairings(tournamentId);
  const startRoundMutation = useStartRound(tournamentId);
  const completeRoundMutation = useCompleteRound(tournamentId);

  const selectedRoundData = rounds.find((r) => r.roundNumber === parseInt(selectedRound));
  const isCurrentRound = currentRound?.roundNumber === parseInt(selectedRound);

  const getStatusBadge = (status: RoundStatus) => {
    switch (status) {
      case 'pending':
        return <Badge variant="outline">Ausstehend</Badge>;
      case 'paired':
        return <Badge variant="secondary">Gepaart</Badge>;
      case 'ongoing':
        return <Badge>Läuft</Badge>;
      case 'completed':
        return (
          <Badge variant="default" className="bg-green-600">
            Beendet
          </Badge>
        );
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  const isPending =
    generatePairingsMutation.isPending ||
    startRoundMutation.isPending ||
    completeRoundMutation.isPending;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Rundensteuerung</CardTitle>
        <CardDescription>Verwalten Sie den Ablauf der Turnierrunden</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center gap-4">
          <div className="space-y-2">
            <Label htmlFor="round-select">Runde auswählen</Label>
            <Select value={selectedRound} onValueChange={setSelectedRound}>
              <SelectTrigger id="round-select" className="w-[180px]">
                <SelectValue placeholder="Runde wählen" />
              </SelectTrigger>
              <SelectContent>
                {rounds.map((r) => (
                  <SelectItem key={r.id} value={r.roundNumber.toString()}>
                    Runde {r.roundNumber}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Status</Label>
            <div className="h-10 flex items-center">
              {selectedRoundData && getStatusBadge(selectedRoundData.status)}
            </div>
          </div>
        </div>

        {selectedRoundData && (
          <div className="flex flex-wrap gap-3">
            {selectedRoundData.status === 'pending' && (
              <Button
                onClick={() => generatePairingsMutation.mutate(parseInt(selectedRound))}
                disabled={isPending}
              >
                {generatePairingsMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Shuffle className="mr-2 h-4 w-4" />
                )}
                Paarungen erstellen
              </Button>
            )}

            {selectedRoundData.status === 'paired' && (
              <Button
                onClick={() => startRoundMutation.mutate(parseInt(selectedRound))}
                disabled={isPending}
              >
                {startRoundMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <PlayCircle className="mr-2 h-4 w-4" />
                )}
                Runde starten
              </Button>
            )}

            {selectedRoundData.status === 'ongoing' && (
              <Button
                onClick={() => completeRoundMutation.mutate(parseInt(selectedRound))}
                disabled={isPending || (isCurrentRound && pendingResults > 0)}
                title={
                  isCurrentRound && pendingResults > 0
                    ? `Noch ${pendingResults} offene Ergebnisse`
                    : ''
                }
              >
                {completeRoundMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <CheckCircle2 className="mr-2 h-4 w-4" />
                )}
                Runde beenden
                {isCurrentRound && pendingResults > 0 && (
                  <span className="ml-2 text-xs">({pendingResults} offen)</span>
                )}
              </Button>
            )}

            {selectedRoundData.status === 'completed' && (
              <p className="text-sm text-muted-foreground flex items-center">
                <CheckCircle2 className="mr-2 h-4 w-4 text-green-600" />
                Diese Runde ist abgeschlossen.
              </p>
            )}
          </div>
        )}

        <div className="border-t pt-4">
          <h4 className="font-medium mb-2">Rundenübersicht</h4>
          <div className="flex flex-wrap gap-2">
            {rounds.map((r) => (
              <Button
                key={r.id}
                variant={r.roundNumber === parseInt(selectedRound) ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedRound(r.roundNumber.toString())}
                className={cn(
                  r.status === 'completed' && 'border-green-600',
                  r.status === 'ongoing' && 'border-blue-600'
                )}
              >
                R{r.roundNumber}
                {r.status === 'completed' && (
                  <CheckCircle2 className="ml-1 h-3 w-3 text-green-600" />
                )}
              </Button>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
