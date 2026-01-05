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
import type { SeatingChartData, SeatingPosition } from '@/types/roundrobin';

interface SeatingChartListProps {
  data: SeatingChartData;
  sortBy?: 'seat' | 'startNo' | 'name';
  className?: string;
}

export function SeatingChartList({
  data,
  sortBy = 'seat',
  className,
}: SeatingChartListProps) {
  // Sort seats based on sortBy parameter
  const sortedSeats = [...data.seats].sort((a, b) => {
    if (sortBy === 'startNo') {
      return a.startNumber - b.startNumber;
    }
    if (sortBy === 'name') {
      return (a.playerName || '').localeCompare(b.playerName || '');
    }
    return a.seatNumber - b.seatNumber;
  });

  // Build a lookup for pairings by seat
  const seatToPairing = new Map<number, { board: number; color: 'white' | 'black' }>();
  for (const pairing of data.pairings) {
    if (!pairing.isBye) {
      seatToPairing.set(pairing.whiteSeat, { board: pairing.board, color: 'white' });
      seatToPairing.set(pairing.blackSeat, { board: pairing.board, color: 'black' });
    }
  }

  return (
    <div className={cn('overflow-x-auto', className)}>
      <Table className="text-sm">
        <TableHeader>
          <TableRow>
            <TableHead className="w-16 text-center">Platz</TableHead>
            <TableHead className="w-16 text-center">StNr</TableHead>
            <TableHead className="min-w-[180px]">Spieler</TableHead>
            <TableHead className="w-16 text-right">ELO</TableHead>
            <TableHead className="w-20 text-center">Tisch</TableHead>
            <TableHead className="w-20 text-center">Farbe</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedSeats.map((seat) => {
            const pairingInfo = seatToPairing.get(seat.seatNumber);

            return (
              <TableRow
                key={seat.seatNumber}
                className={cn(
                  seat.isFixed && 'bg-primary/5',
                  seat.isBye && 'bg-muted/50 text-muted-foreground'
                )}
              >
                <TableCell className="text-center font-medium">
                  <div className="flex items-center justify-center gap-1">
                    {seat.seatNumber}
                    {seat.isFixed && (
                      <span className="text-yellow-500" title="Fixiert">
                        ★
                      </span>
                    )}
                  </div>
                </TableCell>
                <TableCell className="text-center">
                  {seat.isBye ? '-' : seat.startNumber}
                </TableCell>
                <TableCell>
                  {seat.isBye ? (
                    <span className="text-muted-foreground italic">
                      [Spielfrei]
                    </span>
                  ) : (
                    seat.playerName || `Spieler ${seat.startNumber}`
                  )}
                </TableCell>
                <TableCell className="text-right">
                  {seat.isBye ? '-' : (seat.rating || '-')}
                </TableCell>
                <TableCell className="text-center">
                  {pairingInfo ? pairingInfo.board : (seat.isBye ? '-' : '-')}
                </TableCell>
                <TableCell className="text-center">
                  {pairingInfo ? (
                    <Badge
                      variant={pairingInfo.color === 'white' ? 'outline' : 'default'}
                      className={cn(
                        'text-xs',
                        pairingInfo.color === 'white'
                          ? 'bg-white text-black border-gray-300'
                          : 'bg-gray-800 text-white'
                      )}
                    >
                      {pairingInfo.color === 'white' ? 'Weiß' : 'Schwarz'}
                    </Badge>
                  ) : seat.isBye ? (
                    <Badge variant="secondary" className="text-xs">
                      Frei
                    </Badge>
                  ) : (
                    '-'
                  )}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
