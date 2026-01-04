'use client';

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

interface CrosstablePlayer {
  rank: number;
  participantId: string;
  playerId: string;
  name: string;
  title?: string;
  rating: number;
  points: number;
  buchholz: number;
  sonnebornBerger: number;
  results: CrosstableResult[];
}

interface CrosstableResult {
  round: number;
  opponentRank: number;
  color: string;
  score: number;
}

interface CrossTableProps {
  players: CrosstablePlayer[];
  rounds: number;
  highlightPlayerId?: string;
}

export function CrossTable({
  players,
  rounds,
  highlightPlayerId,
}: CrossTableProps) {
  return (
    <div className="overflow-x-auto">
      <Table className="text-sm">
        <TableHeader>
          <TableRow>
            <TableHead className="w-10 text-center sticky left-0 bg-background">
              #
            </TableHead>
            <TableHead className="min-w-[150px] sticky left-10 bg-background">
              Name
            </TableHead>
            <TableHead className="w-12 text-right">ELO</TableHead>
            {Array.from({ length: rounds }, (_, i) => (
              <TableHead key={i} className="w-12 text-center">
                {i + 1}
              </TableHead>
            ))}
            <TableHead className="w-14 text-center font-bold">Pkt</TableHead>
            <TableHead className="w-14 text-right">BH</TableHead>
            <TableHead className="w-14 text-right">SB</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {players.map((player, index) => (
            <TableRow
              key={player.participantId}
              className={cn(
                player.playerId === highlightPlayerId && 'bg-primary/5'
              )}
            >
              <TableCell className="text-center font-medium sticky left-0 bg-background">
                {player.rank}
              </TableCell>
              <TableCell className="sticky left-10 bg-background">
                <div className="flex items-center gap-1">
                  {player.title && (
                    <Badge variant="secondary" className="text-xs px-1">
                      {player.title}
                    </Badge>
                  )}
                  <span className="truncate">{player.name}</span>
                </div>
              </TableCell>
              <TableCell className="text-right text-muted-foreground">
                {player.rating || '-'}
              </TableCell>
              {Array.from({ length: rounds }, (_, roundIndex) => {
                const result = player.results[roundIndex];
                return (
                  <TableCell key={roundIndex} className="text-center p-1">
                    {result ? (
                      <ResultCell
                        result={result}
                        isSelf={result.opponentRank === player.rank}
                      />
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </TableCell>
                );
              })}
              <TableCell className="text-center font-bold">
                {player.points.toFixed(1)}
              </TableCell>
              <TableCell className="text-right text-muted-foreground">
                {player.buchholz.toFixed(1)}
              </TableCell>
              <TableCell className="text-right text-muted-foreground">
                {player.sonnebornBerger.toFixed(2)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function ResultCell({
  result,
  isSelf,
}: {
  result: CrosstableResult;
  isSelf: boolean;
}) {
  if (isSelf) {
    return <span className="text-muted-foreground">×</span>;
  }

  const scoreDisplay = getScoreDisplay(result.score);
  const colorIndicator = result.color === 'w' ? 'text-white' : 'text-black';

  return (
    <div
      className={cn(
        'inline-flex items-center justify-center w-10 h-6 rounded text-xs font-medium',
        result.score === 1 && 'bg-green-100 text-green-800',
        result.score === 0.5 && 'bg-gray-100 text-gray-800',
        result.score === 0 && 'bg-red-100 text-red-800'
      )}
      title={`vs ${result.opponentRank} (${result.color === 'w' ? 'Weiß' : 'Schwarz'})`}
    >
      <span className="mr-0.5">{result.opponentRank}</span>
      <span
        className={cn(
          'w-1.5 h-1.5 rounded-full border border-gray-400',
          result.color === 'w' ? 'bg-white' : 'bg-black'
        )}
      />
      <span className="ml-0.5">{scoreDisplay}</span>
    </div>
  );
}

function getScoreDisplay(score: number): string {
  if (score === 1) return '+';
  if (score === 0.5) return '=';
  if (score === 0) return '-';
  return '?';
}
