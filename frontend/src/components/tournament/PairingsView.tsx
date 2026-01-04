'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
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
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { Edit2 } from 'lucide-react';
import { api } from '@/lib/api';
import { toast } from 'sonner';

interface Game {
  id: string;
  boardNumber: number;
  whiteParticipantId?: string;
  blackParticipantId?: string;
  whiteName?: string;
  blackName?: string;
  whiteRating?: number;
  blackRating?: number;
  result?: string;
  status: string;
}

interface PairingsViewProps {
  tournamentId: string;
  games: Game[];
  roundNumber: number;
  roundStatus: string;
  canEdit?: boolean;
}

const resultLabels: Record<string, string> = {
  '1-0': '1 - 0',
  '0-1': '0 - 1',
  '1/2-1/2': '½ - ½',
  '1-0F': '1 - 0 (k)',
  '0-1F': '0 - 1 (k)',
  '0-0F': '0 - 0',
  BYE: 'BYE',
};

export function PairingsView({
  tournamentId,
  games,
  roundNumber,
  roundStatus,
  canEdit = false,
}: PairingsViewProps) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-16 text-center">Brett</TableHead>
            <TableHead className="text-right">Weiß</TableHead>
            <TableHead className="w-24 text-center">Ergebnis</TableHead>
            <TableHead>Schwarz</TableHead>
            {canEdit && <TableHead className="w-16"></TableHead>}
          </TableRow>
        </TableHeader>
        <TableBody>
          {games.map((game) => (
            <PairingRow
              key={game.id}
              game={game}
              tournamentId={tournamentId}
              canEdit={canEdit && roundStatus === 'ongoing'}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function PairingRow({
  game,
  tournamentId,
  canEdit,
}: {
  game: Game;
  tournamentId: string;
  canEdit: boolean;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedResult, setSelectedResult] = useState(game.result || '');
  const queryClient = useQueryClient();

  const resultMutation = useMutation({
    mutationFn: (result: string) =>
      api.put(`/tournaments/${tournamentId}/games/${game.id}/result`, { result }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pairings', tournamentId] });
      queryClient.invalidateQueries({ queryKey: ['standings', tournamentId] });
      toast.success('Ergebnis eingetragen');
      setIsOpen(false);
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Speichern');
    },
  });

  const isBye = !game.blackParticipantId || game.result === 'BYE';

  return (
    <TableRow className={cn(game.result && 'bg-muted/30')}>
      <TableCell className="text-center font-medium">{game.boardNumber}</TableCell>
      <TableCell className="text-right">
        <PlayerCell
          name={game.whiteName || 'Spieler ' + game.whiteParticipantId}
          rating={game.whiteRating}
          isWinner={game.result === '1-0' || game.result === '1-0F'}
        />
      </TableCell>
      <TableCell className="text-center">
        {game.result ? (
          <Badge
            variant={game.result === '1/2-1/2' ? 'secondary' : 'default'}
            className={cn(
              'font-mono',
              game.result === '1-0' && 'bg-green-100 text-green-800',
              game.result === '0-1' && 'bg-red-100 text-red-800'
            )}
          >
            {resultLabels[game.result] || game.result}
          </Badge>
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell>
        {isBye ? (
          <span className="text-muted-foreground italic">spielfrei</span>
        ) : (
          <PlayerCell
            name={game.blackName || 'Spieler ' + game.blackParticipantId}
            rating={game.blackRating}
            isWinner={game.result === '0-1' || game.result === '0-1F'}
          />
        )}
      </TableCell>
      {canEdit && !isBye && (
        <TableCell>
          <Dialog open={isOpen} onOpenChange={setIsOpen}>
            <DialogTrigger asChild>
              <Button variant="ghost" size="sm">
                <Edit2 className="h-4 w-4" />
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Ergebnis eintragen</DialogTitle>
                <DialogDescription>
                  Brett {game.boardNumber}: {game.whiteName} vs {game.blackName}
                </DialogDescription>
              </DialogHeader>
              <div className="py-4">
                <Select value={selectedResult} onValueChange={setSelectedResult}>
                  <SelectTrigger>
                    <SelectValue placeholder="Ergebnis wählen" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1-0">1 - 0 (Weiß gewinnt)</SelectItem>
                    <SelectItem value="0-1">0 - 1 (Schwarz gewinnt)</SelectItem>
                    <SelectItem value="1/2-1/2">½ - ½ (Remis)</SelectItem>
                    <SelectItem value="1-0F">1 - 0 (kampflos)</SelectItem>
                    <SelectItem value="0-1F">0 - 1 (kampflos)</SelectItem>
                    <SelectItem value="0-0F">0 - 0 (beide nicht erschienen)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsOpen(false)}>
                  Abbrechen
                </Button>
                <Button
                  onClick={() => resultMutation.mutate(selectedResult)}
                  disabled={!selectedResult || resultMutation.isPending}
                >
                  {resultMutation.isPending ? 'Speichern...' : 'Speichern'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </TableCell>
      )}
    </TableRow>
  );
}

function PlayerCell({
  name,
  rating,
  isWinner,
}: {
  name: string;
  rating?: number;
  isWinner?: boolean;
}) {
  return (
    <div className={cn('flex items-center gap-2', isWinner && 'font-semibold')}>
      <span className={cn(isWinner && 'text-green-700')}>{name}</span>
      {rating && (
        <span className="text-xs text-muted-foreground">({rating})</span>
      )}
    </div>
  );
}
