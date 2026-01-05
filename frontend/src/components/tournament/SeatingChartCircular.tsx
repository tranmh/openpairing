'use client';

import { useMemo } from 'react';
import { cn } from '@/lib/utils';
import type { SeatingChartData, SeatingPosition } from '@/types/roundrobin';

interface SeatingChartCircularProps {
  data: SeatingChartData;
  highlightSeat?: number;
  onSeatClick?: (seat: SeatingPosition) => void;
  showPairings?: boolean;
  className?: string;
  size?: number;
}

export function SeatingChartCircular({
  data,
  highlightSeat,
  onSeatClick,
  showPairings = true,
  className,
  size = 500,
}: SeatingChartCircularProps) {
  const { seats, pairings, numSeats } = data;

  // Calculate positions for seats arranged in a circle
  const seatPositions = useMemo(() => {
    const centerX = size / 2;
    const centerY = size / 2;
    const radius = (size - 100) / 2; // Leave room for labels

    return seats.map((seat, index) => {
      // Start at top (12 o'clock) and go clockwise
      const angle = (index / numSeats) * 2 * Math.PI - Math.PI / 2;
      const x = centerX + radius * Math.cos(angle);
      const y = centerY + radius * Math.sin(angle);
      return { ...seat, x, y, angle };
    });
  }, [seats, numSeats, size]);

  // Build seat lookup by seat number
  const seatByNumber = useMemo(() => {
    const map = new Map<number, (typeof seatPositions)[0]>();
    for (const pos of seatPositions) {
      map.set(pos.seatNumber, pos);
    }
    return map;
  }, [seatPositions]);

  const centerX = size / 2;
  const centerY = size / 2;
  const seatRadius = Math.max(20, Math.min(30, 400 / numSeats));

  return (
    <div className={cn('flex justify-center', className)}>
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        className="overflow-visible"
      >
        {/* Pairing lines connecting opponents */}
        {showPairings &&
          pairings
            .filter((p) => !p.isBye)
            .map((pairing) => {
              const whiteSeat = seatByNumber.get(pairing.whiteSeat);
              const blackSeat = seatByNumber.get(pairing.blackSeat);
              if (!whiteSeat || !blackSeat) return null;

              return (
                <line
                  key={`pairing-${pairing.board}`}
                  x1={whiteSeat.x}
                  y1={whiteSeat.y}
                  x2={blackSeat.x}
                  y2={blackSeat.y}
                  stroke="currentColor"
                  strokeOpacity={0.2}
                  strokeWidth={2}
                  strokeDasharray="4 4"
                />
              );
            })}

        {/* Center text showing round number */}
        <text
          x={centerX}
          y={centerY}
          textAnchor="middle"
          dominantBaseline="middle"
          className="fill-muted-foreground text-sm font-medium"
        >
          Runde {data.roundNumber}
        </text>

        {/* Seats */}
        {seatPositions.map((seat) => {
          const isHighlighted = seat.seatNumber === highlightSeat;
          const isFixed = seat.isFixed;
          const isBye = seat.isBye;

          return (
            <g
              key={seat.seatNumber}
              transform={`translate(${seat.x}, ${seat.y})`}
              onClick={() => onSeatClick?.(seat)}
              className={cn(
                'cursor-pointer transition-transform hover:scale-110',
                onSeatClick && 'cursor-pointer'
              )}
            >
              {/* Seat circle */}
              <circle
                r={seatRadius}
                className={cn(
                  'stroke-2 transition-colors',
                  isBye
                    ? 'fill-muted stroke-muted-foreground stroke-dashed'
                    : isFixed
                      ? 'fill-yellow-100 stroke-yellow-500 dark:fill-yellow-900/30'
                      : isHighlighted
                        ? 'fill-primary/20 stroke-primary'
                        : 'fill-background stroke-border'
                )}
                strokeDasharray={isBye ? '4 4' : undefined}
              />

              {/* Seat number */}
              <text
                y={-seatRadius - 8}
                textAnchor="middle"
                className="fill-muted-foreground text-xs"
              >
                {seat.seatNumber}
              </text>

              {/* Start number (inside circle) */}
              <text
                y={isBye ? 5 : -2}
                textAnchor="middle"
                dominantBaseline="middle"
                className={cn(
                  'font-bold',
                  isBye
                    ? 'fill-muted-foreground text-xs'
                    : 'fill-foreground text-sm'
                )}
              >
                {isBye ? 'BYE' : seat.startNumber}
              </text>

              {/* Player name (below number, truncated) */}
              {!isBye && seat.playerName && (
                <text
                  y={12}
                  textAnchor="middle"
                  className="fill-muted-foreground text-[10px]"
                >
                  {seat.playerName.length > 10
                    ? seat.playerName.substring(0, 10) + '...'
                    : seat.playerName}
                </text>
              )}

              {/* Fixed player star */}
              {isFixed && (
                <text
                  x={seatRadius - 5}
                  y={-seatRadius + 5}
                  className="fill-yellow-500 text-xs"
                >
                  ★
                </text>
              )}
            </g>
          );
        })}
      </svg>
    </div>
  );
}
