'use client';

import { useMemo } from 'react';
import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';
import type { SeatingChartData, SeatingPosition } from '@/types/roundrobin';

interface SeatingChartColumnsProps {
  data: SeatingChartData;
  highlightStartNo?: number;
  className?: string;
}

/**
 * Two-column layout for mobile devices.
 * Left column: Fixed player (#1) at top, descending to middle of circle
 * Right column: Bye position at top, continuing seats
 * Horizontal lines connect paired opponents
 */
export function SeatingChartColumns({
  data,
  highlightStartNo,
  className,
}: SeatingChartColumnsProps) {
  const { seats, pairings, numSeats, byeSeat } = data;

  // Split seats into two columns (left = first half, right = second half)
  // This represents the two "sides" of the circular table
  const { leftColumn, rightColumn, pairingMap } = useMemo(() => {
    const midpoint = Math.ceil(numSeats / 2);

    // Left column: seats 1 to midpoint
    const left = seats.filter((s) => s.seatNumber <= midpoint);

    // Right column: seats from midpoint+1 to numSeats, but we show them
    // aligned with left column so seat 1 pairs with seat numSeats
    // Reverse the right column so pairings align horizontally
    const right = seats
      .filter((s) => s.seatNumber > midpoint)
      .sort((a, b) => b.seatNumber - a.seatNumber);

    // Build pairing map: startNo -> opponent startNo
    const pMap = new Map<number, number>();
    for (const p of pairings) {
      if (!p.isBye) {
        pMap.set(p.whiteStartNo, p.blackStartNo);
        pMap.set(p.blackStartNo, p.whiteStartNo);
      }
    }

    return { leftColumn: left, rightColumn: right, pairingMap: pMap };
  }, [seats, pairings, numSeats]);

  // Create rows by pairing left and right column entries
  const rows = useMemo(() => {
    const result: Array<{
      left: SeatingPosition | null;
      right: SeatingPosition | null;
      isPaired: boolean;
    }> = [];

    const maxRows = Math.max(leftColumn.length, rightColumn.length);

    for (let i = 0; i < maxRows; i++) {
      const left = leftColumn[i] || null;
      const right = rightColumn[i] || null;

      // Check if these two are actually paired
      let isPaired = false;
      if (left && right && !left.isBye && !right.isBye) {
        isPaired =
          pairingMap.get(left.startNumber) === right.startNumber ||
          pairingMap.get(right.startNumber) === left.startNumber;
      }

      result.push({ left, right, isPaired });
    }

    return result;
  }, [leftColumn, rightColumn, pairingMap]);

  const SeatCard = ({
    seat,
    side,
  }: {
    seat: SeatingPosition | null;
    side: 'left' | 'right';
  }) => {
    if (!seat) {
      return <div className="flex-1" />;
    }

    const isHighlighted = seat.startNumber === highlightStartNo;

    return (
      <div
        className={cn(
          'flex items-center gap-2 p-2 rounded-md border',
          side === 'left' ? 'flex-row' : 'flex-row-reverse',
          seat.isBye
            ? 'bg-muted/50 border-dashed border-muted-foreground/30'
            : seat.isFixed
              ? 'bg-yellow-50 border-yellow-300 dark:bg-yellow-900/20 dark:border-yellow-700'
              : isHighlighted
                ? 'bg-primary/10 border-primary'
                : 'bg-background border-border'
        )}
      >
        {/* Start number badge */}
        <div
          className={cn(
            'w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm',
            seat.isBye
              ? 'bg-muted text-muted-foreground'
              : seat.isFixed
                ? 'bg-yellow-200 text-yellow-800 dark:bg-yellow-800 dark:text-yellow-200'
                : 'bg-primary/10 text-primary'
          )}
        >
          {seat.isBye ? '-' : seat.startNumber}
        </div>

        {/* Player info */}
        <div className={cn('flex-1 min-w-0', side === 'right' && 'text-right')}>
          <div className="font-medium text-sm truncate">
            {seat.isBye ? (
              <span className="text-muted-foreground italic">[Spielfrei]</span>
            ) : (
              seat.playerName || `Spieler ${seat.startNumber}`
            )}
          </div>
          {!seat.isBye && seat.rating && (
            <div className="text-xs text-muted-foreground">{seat.rating}</div>
          )}
        </div>

        {/* Fixed indicator */}
        {seat.isFixed && (
          <span className="text-yellow-500 text-sm" title="Fixiert">
            ★
          </span>
        )}
      </div>
    );
  };

  return (
    <div className={cn('space-y-1', className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-2 py-1 text-xs text-muted-foreground font-medium">
        <span>Linke Seite</span>
        <Badge variant="outline" className="text-xs">
          Runde {data.roundNumber}
        </Badge>
        <span>Rechte Seite</span>
      </div>

      {/* Rows */}
      <div className="space-y-1">
        {rows.map((row, index) => (
          <div key={index} className="flex items-center gap-2">
            {/* Left seat */}
            <div className="flex-1">
              <SeatCard seat={row.left} side="left" />
            </div>

            {/* Connector line */}
            <div className="w-8 flex items-center justify-center">
              {row.isPaired ? (
                <div className="w-full h-0.5 bg-border" />
              ) : (
                <div className="w-full h-0.5 bg-transparent" />
              )}
            </div>

            {/* Right seat */}
            <div className="flex-1">
              <SeatCard seat={row.right} side="right" />
            </div>
          </div>
        ))}
      </div>

      {/* Legend */}
      <div className="flex items-center justify-center gap-4 pt-2 text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <span className="text-yellow-500">★</span>
          <span>Fixiert</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-4 h-0.5 bg-border" />
          <span>Paarung</span>
        </div>
      </div>
    </div>
  );
}
