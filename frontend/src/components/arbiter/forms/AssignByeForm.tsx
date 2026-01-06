'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Loader2 } from 'lucide-react';
import { useAssignBye } from '../hooks/useArbiterMutations';
import type { AssignByeFormProps } from '../types';
import type { ByeType } from '@/types/arbiter';

/**
 * Form for assigning a bye to a player
 */
export function AssignByeForm({
  tournamentId,
  roundNumber,
  participants,
}: AssignByeFormProps) {
  const [participantId, setParticipantId] = useState('');
  const [byeType, setByeType] = useState<ByeType>('half');
  const mutation = useAssignBye(tournamentId);

  const handleSubmit = () => {
    mutation.mutate(
      { participantId, roundNumber, byeType },
      {
        onSuccess: () => setParticipantId(''),
      }
    );
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="bye-participant">Spieler</Label>
        <Select value={participantId} onValueChange={setParticipantId}>
          <SelectTrigger id="bye-participant">
            <SelectValue placeholder="Spieler auswählen" />
          </SelectTrigger>
          <SelectContent>
            {participants.map((p) => (
              <SelectItem key={p.id} value={p.id}>
                {p.startRank}. {p.playerName || p.playerId} ({p.startingRating})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label htmlFor="bye-type">Typ</Label>
        <Select value={byeType} onValueChange={(v) => setByeType(v as ByeType)}>
          <SelectTrigger id="bye-type">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="half">Halber Punkt (0.5)</SelectItem>
            <SelectItem value="full">Voller Punkt (1.0)</SelectItem>
            <SelectItem value="zero">Null Punkte (0)</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <Button
        onClick={handleSubmit}
        disabled={!participantId || mutation.isPending}
        className="w-full"
      >
        {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
        Spielfrei vergeben
      </Button>
    </div>
  );
}
