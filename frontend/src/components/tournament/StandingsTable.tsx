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

interface Standing {
  rank: number;
  participantId: string;
  playerId: string;
  name: string;
  title?: string;
  rating: number;
  federation?: string;
  points: number;
  buchholz: number;
  buchholzCut1: number;
  sonnebornBerger: number;
  wins: number;
  draws: number;
  losses: number;
}

interface StandingsTableProps {
  standings: Standing[];
  tiebreaks?: string[];
  highlightPlayerId?: string;
  showResults?: boolean;
}

const tiebreakLabels: Record<string, string> = {
  buchholz: 'BH',
  buchholz_cut1: 'BH-C1',
  buchholz_cut2: 'BH-C2',
  buchholz_median: 'BH-M',
  sonneborn_berger: 'SB',
  direct_encounter: 'DV',
  wins: 'S',
  games_black: 'PS',
  wins_black: 'SS',
  progress: 'Prog',
  aroc: 'AROC',
  koya: 'Koya',
  rating: 'ELO',
};

const titleColors: Record<string, string> = {
  GM: 'bg-amber-100 text-amber-800',
  IM: 'bg-gray-100 text-gray-800',
  FM: 'bg-green-100 text-green-800',
  CM: 'bg-blue-100 text-blue-800',
  WGM: 'bg-amber-100 text-amber-800',
  WIM: 'bg-gray-100 text-gray-800',
  WFM: 'bg-green-100 text-green-800',
  WCM: 'bg-blue-100 text-blue-800',
};

export function StandingsTable({
  standings,
  tiebreaks = ['buchholz', 'buchholz_cut1', 'sonneborn_berger'],
  highlightPlayerId,
  showResults = true,
}: StandingsTableProps) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-12 text-center">Platz</TableHead>
            <TableHead className="w-12 text-center">Nr.</TableHead>
            <TableHead>Name</TableHead>
            <TableHead className="text-right w-16">ELO</TableHead>
            <TableHead className="text-center w-20">Punkte</TableHead>
            {showResults && (
              <TableHead className="text-center w-24">+/=/−</TableHead>
            )}
            {tiebreaks.slice(0, 4).map((tb) => (
              <TableHead key={tb} className="text-right w-16">
                {tiebreakLabels[tb] || tb}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {standings.map((player, index) => {
            const isHighlighted = player.playerId === highlightPlayerId;

            return (
              <TableRow
                key={player.participantId}
                className={cn(
                  isHighlighted && 'bg-primary/5',
                  index < 3 && 'font-medium'
                )}
              >
                <TableCell className="text-center">
                  <span
                    className={cn(
                      'inline-flex items-center justify-center w-6 h-6 rounded-full text-sm',
                      index === 0 && 'bg-yellow-100 text-yellow-800',
                      index === 1 && 'bg-gray-100 text-gray-800',
                      index === 2 && 'bg-orange-100 text-orange-800'
                    )}
                  >
                    {player.rank}
                  </span>
                </TableCell>
                <TableCell className="text-center text-muted-foreground">
                  {index + 1}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    {player.title && (
                      <Badge
                        variant="secondary"
                        className={cn('text-xs', titleColors[player.title])}
                      >
                        {player.title}
                      </Badge>
                    )}
                    <span>{player.name}</span>
                    {player.federation && (
                      <span className="text-xs text-muted-foreground">
                        ({player.federation})
                      </span>
                    )}
                  </div>
                </TableCell>
                <TableCell className="text-right text-muted-foreground">
                  {player.rating || '-'}
                </TableCell>
                <TableCell className="text-center font-semibold">
                  {player.points.toFixed(1)}
                </TableCell>
                {showResults && (
                  <TableCell className="text-center text-sm text-muted-foreground">
                    {player.wins}/{player.draws}/{player.losses}
                  </TableCell>
                )}
                {tiebreaks.slice(0, 4).map((tb) => (
                  <TableCell key={tb} className="text-right text-muted-foreground">
                    {getTiebreakValue(player, tb)}
                  </TableCell>
                ))}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

function getTiebreakValue(player: Standing, tiebreak: string): string {
  const value = (player as any)[tiebreak] ||
    (player as any)[tiebreak.replace('_', '')] ||
    (tiebreak === 'buchholz' && player.buchholz) ||
    (tiebreak === 'buchholz_cut1' && player.buchholzCut1) ||
    (tiebreak === 'sonneborn_berger' && player.sonnebornBerger) ||
    (tiebreak === 'wins' && player.wins);

  if (typeof value === 'number') {
    return value % 1 === 0 ? value.toString() : value.toFixed(1);
  }
  return '-';
}
