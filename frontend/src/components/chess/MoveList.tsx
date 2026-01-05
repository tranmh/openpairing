'use client';

import { useEffect, useMemo, useRef } from 'react';
import { Chess } from 'chess.js';
import { cn } from '@/lib/utils';
import { useLanguageStore } from '@/store/language';

interface MoveListProps {
  moves: string[];
  currentMoveIndex?: number;
  onMoveClick?: (index: number) => void;
  className?: string;
}

interface MoveDisplay {
  moveNumber: number;
  whiteMove: { uci: string; san: string; index: number } | null;
  blackMove: { uci: string; san: string; index: number } | null;
}

export function MoveList({
  moves,
  currentMoveIndex,
  onMoveClick,
  className,
}: MoveListProps) {
  const { t } = useLanguageStore();
  const listRef = useRef<HTMLDivElement>(null);

  // Convert UCI moves to SAN notation
  const displayMoves = useMemo(() => {
    const chess = new Chess();
    const result: MoveDisplay[] = [];

    for (let i = 0; i < moves.length; i += 2) {
      const moveNumber = Math.floor(i / 2) + 1;
      let whiteMove: MoveDisplay['whiteMove'] = null;
      let blackMove: MoveDisplay['blackMove'] = null;

      // White's move
      if (i < moves.length) {
        const uci = moves[i];
        try {
          const move = chess.move({
            from: uci.slice(0, 2),
            to: uci.slice(2, 4),
            promotion: uci.length > 4 ? uci[4] : undefined,
          });
          whiteMove = { uci, san: move.san, index: i };
        } catch {
          whiteMove = { uci, san: uci, index: i };
        }
      }

      // Black's move
      if (i + 1 < moves.length) {
        const uci = moves[i + 1];
        try {
          const move = chess.move({
            from: uci.slice(0, 2),
            to: uci.slice(2, 4),
            promotion: uci.length > 4 ? uci[4] : undefined,
          });
          blackMove = { uci, san: move.san, index: i + 1 };
        } catch {
          blackMove = { uci, san: uci, index: i + 1 };
        }
      }

      result.push({ moveNumber, whiteMove, blackMove });
    }

    return result;
  }, [moves]);

  // Auto-scroll to latest move
  useEffect(() => {
    if (listRef.current && currentMoveIndex === undefined) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [moves.length, currentMoveIndex]);

  const renderMove = (
    move: { uci: string; san: string; index: number } | null,
    isBlack: boolean
  ) => {
    if (!move) {
      return <span className="w-16 text-gray-600">...</span>;
    }

    const isCurrentMove =
      currentMoveIndex !== undefined && move.index === currentMoveIndex;

    return (
      <button
        onClick={() => onMoveClick?.(move.index)}
        className={cn(
          'w-16 rounded px-1 py-0.5 text-left transition-colors',
          onMoveClick && 'hover:bg-gray-700 cursor-pointer',
          isCurrentMove && 'bg-blue-600 text-white',
          !isCurrentMove && (isBlack ? 'text-gray-300' : 'text-gray-200')
        )}
        disabled={!onMoveClick}
      >
        {move.san}
      </button>
    );
  };

  if (moves.length === 0) {
    return (
      <div className={cn('flex items-center justify-center p-4 text-gray-500', className)}>
        {t('chess.noMovesYet')}
      </div>
    );
  }

  return (
    <div
      ref={listRef}
      className={cn(
        'overflow-y-auto rounded-lg bg-gray-800 p-2',
        className
      )}
    >
      <div className="space-y-0.5">
        {displayMoves.map((row) => (
          <div
            key={row.moveNumber}
            className="flex items-center gap-1 text-sm font-mono"
          >
            <span className="w-8 text-gray-500 text-right pr-1">
              {row.moveNumber}.
            </span>
            {renderMove(row.whiteMove, false)}
            {renderMove(row.blackMove, true)}
          </div>
        ))}
      </div>
    </div>
  );
}
