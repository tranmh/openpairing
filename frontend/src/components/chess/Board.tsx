'use client';

import { useCallback, useMemo, useState } from 'react';
import { Chessboard } from 'react-chessboard';
import { Chess, Square } from 'chess.js';
import { cn } from '@/lib/utils';

interface BoardProps {
  fen: string;
  orientation?: 'white' | 'black';
  onMove?: (move: { from: string; to: string; promotion?: string }) => void;
  allowMoves?: boolean;
  lastMove?: { from: string; to: string };
  highlightSquares?: string[];
  inCheck?: boolean;
  className?: string;
}

export function Board({
  fen,
  orientation = 'white',
  onMove,
  allowMoves = true,
  lastMove,
  highlightSquares = [],
  inCheck = false,
  className,
}: BoardProps) {
  const [moveFrom, setMoveFrom] = useState<Square | null>(null);
  const [possibleMoves, setPossibleMoves] = useState<string[]>([]);

  // Create a chess instance to validate moves
  const game = useMemo(() => {
    const chess = new Chess();
    try {
      chess.load(fen);
    } catch {
      // Invalid FEN, use starting position
      chess.reset();
    }
    return chess;
  }, [fen]);

  // Get king position if in check
  const kingInCheckSquare = useMemo(() => {
    if (!inCheck) return null;
    const turn = game.turn();
    const board = game.board();
    for (let row = 0; row < 8; row++) {
      for (let col = 0; col < 8; col++) {
        const piece = board[row][col];
        if (piece && piece.type === 'k' && piece.color === turn) {
          const files = 'abcdefgh';
          const ranks = '87654321';
          return files[col] + ranks[row];
        }
      }
    }
    return null;
  }, [game, inCheck]);

  // Calculate custom square styles
  const customSquareStyles = useMemo(() => {
    const styles: { [square: string]: React.CSSProperties } = {};

    // Highlight last move
    if (lastMove) {
      styles[lastMove.from] = {
        backgroundColor: 'rgba(255, 255, 0, 0.4)',
      };
      styles[lastMove.to] = {
        backgroundColor: 'rgba(255, 255, 0, 0.4)',
      };
    }

    // Highlight possible moves
    possibleMoves.forEach((square) => {
      const hasPiece = game.get(square as Square);
      styles[square] = {
        background: hasPiece
          ? 'radial-gradient(circle, rgba(0, 0, 0, 0.1) 85%, transparent 85%)'
          : 'radial-gradient(circle, rgba(0, 0, 0, 0.2) 25%, transparent 25%)',
        borderRadius: hasPiece ? '50%' : undefined,
      };
    });

    // Highlight selected square
    if (moveFrom) {
      styles[moveFrom] = {
        backgroundColor: 'rgba(255, 255, 0, 0.5)',
      };
    }

    // Highlight king in check
    if (kingInCheckSquare) {
      styles[kingInCheckSquare] = {
        ...styles[kingInCheckSquare],
        background: 'radial-gradient(circle, rgba(255, 0, 0, 0.8) 0%, rgba(255, 0, 0, 0.4) 50%, transparent 70%)',
      };
    }

    // Highlight custom squares
    highlightSquares.forEach((square) => {
      styles[square] = {
        ...styles[square],
        boxShadow: 'inset 0 0 0 3px rgba(0, 0, 255, 0.5)',
      };
    });

    return styles;
  }, [lastMove, possibleMoves, moveFrom, kingInCheckSquare, highlightSquares, game]);

  const getMoveOptions = useCallback(
    (square: Square) => {
      const moves = game.moves({
        square,
        verbose: true,
      });
      return moves.map((move) => move.to);
    },
    [game]
  );

  const handleSquareClick = useCallback(
    (square: Square) => {
      if (!allowMoves || !onMove) return;

      // If we already have a piece selected
      if (moveFrom) {
        // Check if this is a valid destination
        const moves = game.moves({ square: moveFrom, verbose: true });
        const validMove = moves.find((move) => move.to === square);

        if (validMove) {
          // Check if it's a promotion move
          if (validMove.flags.includes('p')) {
            // Default to queen promotion
            onMove({
              from: moveFrom,
              to: square,
              promotion: 'q',
            });
          } else {
            onMove({
              from: moveFrom,
              to: square,
            });
          }
        }

        // Clear selection
        setMoveFrom(null);
        setPossibleMoves([]);
        return;
      }

      // Check if there's a piece on the clicked square that we can move
      const piece = game.get(square);
      if (piece && piece.color === game.turn()) {
        setMoveFrom(square);
        setPossibleMoves(getMoveOptions(square));
      }
    },
    [allowMoves, onMove, moveFrom, game, getMoveOptions]
  );

  const handlePieceDrop = useCallback(
    (sourceSquare: Square, targetSquare: Square, piece: string) => {
      if (!allowMoves || !onMove) return false;

      const moves = game.moves({ square: sourceSquare, verbose: true });
      const validMove = moves.find((move) => move.to === targetSquare);

      if (!validMove) return false;

      // Check if it's a promotion move
      if (validMove.flags.includes('p')) {
        // Default to queen promotion on drag-drop
        onMove({
          from: sourceSquare,
          to: targetSquare,
          promotion: 'q',
        });
      } else {
        onMove({
          from: sourceSquare,
          to: targetSquare,
        });
      }

      // Clear selection
      setMoveFrom(null);
      setPossibleMoves([]);

      return true;
    },
    [allowMoves, onMove, game]
  );

  const handlePieceDragBegin = useCallback(
    (_piece: string, sourceSquare: Square) => {
      if (!allowMoves) return;
      setMoveFrom(sourceSquare);
      setPossibleMoves(getMoveOptions(sourceSquare));
    },
    [allowMoves, getMoveOptions]
  );

  const handlePieceDragEnd = useCallback(() => {
    setMoveFrom(null);
    setPossibleMoves([]);
  }, []);

  return (
    <div className={cn('w-full max-w-[600px]', className)}>
      <Chessboard
        position={fen}
        boardOrientation={orientation}
        onSquareClick={handleSquareClick}
        onPieceDrop={handlePieceDrop}
        onPieceDragBegin={handlePieceDragBegin}
        onPieceDragEnd={handlePieceDragEnd}
        customSquareStyles={customSquareStyles}
        arePiecesDraggable={allowMoves}
        animationDuration={200}
        customBoardStyle={{
          borderRadius: '4px',
          boxShadow: '0 2px 10px rgba(0, 0, 0, 0.3)',
        }}
        customDarkSquareStyle={{ backgroundColor: '#779556' }}
        customLightSquareStyle={{ backgroundColor: '#ebecd0' }}
      />
    </div>
  );
}
