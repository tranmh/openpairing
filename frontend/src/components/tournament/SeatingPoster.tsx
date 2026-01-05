'use client';

import { cn } from '@/lib/utils';
import type { SeatingChartData, SeatingPosition } from '@/types/roundrobin';

interface SeatingPosterProps {
  data: SeatingChartData;
  className?: string;
}

export function SeatingPoster({ data, className }: SeatingPosterProps) {
  const numSeats = data.numSeats;
  const halfSeats = Math.ceil(numSeats / 2);

  // Split seats into left and right columns for poster layout
  const leftSeats = data.seats.filter((s) => s.seatNumber <= halfSeats);
  const rightSeats = data.seats
    .filter((s) => s.seatNumber > halfSeats)
    .reverse();

  // Find the fixed player seat
  const fixedSeat = data.seats.find(
    (s) => s.seatNumber === data.fixedSeat
  );

  return (
    <div
      className={cn(
        'seating-poster bg-white p-8 max-w-4xl mx-auto',
        className
      )}
    >
      {/* Header */}
      <div className="text-center mb-8 border-b-2 pb-4">
        <h1 className="text-3xl font-bold mb-2">{data.tournamentName}</h1>
        <h2 className="text-2xl font-semibold">
          Sitzplan - Runde {data.roundNumber}
        </h2>
        <p className="text-lg text-muted-foreground mt-2">
          {data.numPlayers} Spieler • {numSeats} Plätze
        </p>
      </div>

      {/* Two-column seating layout */}
      <div className="grid grid-cols-2 gap-8 mb-8">
        {/* Left column */}
        <div className="space-y-2">
          <div className="text-center font-bold text-lg mb-4 border-b pb-2">
            Plätze 1 - {halfSeats}
          </div>
          {leftSeats.map((seat) => (
            <SeatRow
              key={seat.seatNumber}
              seat={seat}
              isFixed={seat.seatNumber === data.fixedSeat}
              isBye={seat.seatNumber === data.byeSeat}
            />
          ))}
        </div>

        {/* Right column */}
        <div className="space-y-2">
          <div className="text-center font-bold text-lg mb-4 border-b pb-2">
            Plätze {halfSeats + 1} - {numSeats}
          </div>
          {rightSeats.map((seat) => (
            <SeatRow
              key={seat.seatNumber}
              seat={seat}
              isFixed={seat.seatNumber === data.fixedSeat}
              isBye={seat.seatNumber === data.byeSeat}
            />
          ))}
        </div>
      </div>

      {/* Pairings section */}
      <div className="border-t-2 pt-4">
        <h3 className="text-xl font-bold text-center mb-4">
          Paarungen Runde {data.roundNumber}
        </h3>
        <div className="grid grid-cols-2 gap-4">
          {data.pairings.map((pairing) => (
            <div
              key={pairing.board}
              className={cn(
                'flex items-center justify-between p-3 border rounded',
                pairing.isBye ? 'bg-gray-100' : 'bg-white'
              )}
            >
              <span className="font-bold text-lg w-8">{pairing.board}.</span>
              {pairing.isBye ? (
                <span className="flex-1 text-center italic text-muted-foreground">
                  Spielfrei: #{pairing.whiteStartNo > data.numPlayers ? pairing.blackStartNo : pairing.whiteStartNo}
                </span>
              ) : (
                <>
                  <span className="flex-1 text-right pr-2">
                    <span className="inline-block w-3 h-3 rounded-full bg-white border border-gray-400 mr-1" />
                    {pairing.whiteName || `#${pairing.whiteStartNo}`}
                  </span>
                  <span className="text-muted-foreground px-2">—</span>
                  <span className="flex-1 text-left pl-2">
                    <span className="inline-block w-3 h-3 rounded-full bg-gray-800 mr-1" />
                    {pairing.blackName || `#${pairing.blackStartNo}`}
                  </span>
                </>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="mt-8 pt-4 border-t text-sm text-muted-foreground">
        <div className="flex justify-center gap-8">
          <div className="flex items-center gap-2">
            <span className="text-yellow-500 text-lg">★</span>
            <span>Fixierter Spieler (bleibt am Platz)</span>
          </div>
          {data.byeSeat > 0 && (
            <div className="flex items-center gap-2">
              <span className="w-4 h-4 border-2 border-dashed border-gray-400 rounded-full inline-block" />
              <span>Spielfrei</span>
            </div>
          )}
        </div>
        <p className="text-center mt-2">
          Rutsch-System: Alle Spieler außer #{fixedSeat?.startNumber || 1} rutschen
          nach jeder Runde einen Platz im Uhrzeigersinn.
        </p>
      </div>
    </div>
  );
}

function SeatRow({
  seat,
  isFixed,
  isBye,
}: {
  seat: SeatingPosition;
  isFixed: boolean;
  isBye: boolean;
}) {
  return (
    <div
      className={cn(
        'flex items-center p-3 rounded border',
        isBye
          ? 'bg-gray-100 border-dashed'
          : isFixed
            ? 'bg-yellow-50 border-yellow-300'
            : 'bg-white'
      )}
    >
      {/* Seat number */}
      <div className="w-12 text-center">
        <span className="inline-flex items-center justify-center w-8 h-8 rounded-full bg-gray-200 font-bold">
          {seat.seatNumber}
        </span>
      </div>

      {/* Player info */}
      <div className="flex-1 ml-4">
        {isBye ? (
          <span className="italic text-muted-foreground">Spielfrei</span>
        ) : (
          <>
            <span className="font-bold text-lg mr-2">#{seat.startNumber}</span>
            <span>{seat.playerName || 'N.N.'}</span>
            {seat.rating && seat.rating > 0 && (
              <span className="ml-2 text-muted-foreground">
                ({seat.rating})
              </span>
            )}
          </>
        )}
      </div>

      {/* Fixed indicator */}
      {isFixed && !isBye && (
        <span className="text-yellow-500 text-xl">★</span>
      )}
    </div>
  );
}
