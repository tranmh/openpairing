'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, Copy, Share2 } from 'lucide-react';
import { Board } from '@/components/chess/Board';
import { GameClocks } from '@/components/chess/Clock';
import { MoveList } from '@/components/chess/MoveList';
import { GameControls } from '@/components/chess/GameControls';
import { GameResult } from '@/components/chess/GameResult';
import { useGame } from '@/hooks/useGame';
import { useAuth } from '@/hooks/useAuth';
import { cn } from '@/lib/utils';

export default function GamePage() {
  const params = useParams();
  const router = useRouter();
  const { user } = useAuth();
  const gameId = params.gameId as string;

  const [showResult, setShowResult] = useState(false);
  const [lastMove, setLastMove] = useState<{ from: string; to: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleMove = useCallback((move: string, state: any) => {
    // Extract last move from UCI
    if (move.length >= 4) {
      setLastMove({
        from: move.slice(0, 2),
        to: move.slice(2, 4),
      });
    }
  }, []);

  const handleGameOver = useCallback((result: string, termination: string) => {
    setShowResult(true);
  }, []);

  const handleDrawOffered = useCallback((byUserId: string) => {
    // The GameControls component handles displaying the draw offer
  }, []);

  const handleError = useCallback((errorMessage: string) => {
    setError(errorMessage);
    // Auto-clear error after 5 seconds
    setTimeout(() => setError(null), 5000);
  }, []);

  const {
    gameState,
    isConnected,
    spectatorCount,
    opponentConnected,
    playerColor,
    isMyTurn,
    isSpectator,
    canAbort,
    hasDrawOffer,
    isDrawOfferMine,
    makeMove,
    resign,
    offerDraw,
    acceptDraw,
    declineDraw,
    abort,
  } = useGame({
    gameId,
    onMove: handleMove,
    onGameOver: handleGameOver,
    onDrawOffered: handleDrawOffered,
    onError: handleError,
  });

  // Set last move from game state on initial load
  useEffect(() => {
    if (gameState?.moves && gameState.moves.length > 0 && !lastMove) {
      const lastUci = gameState.moves[gameState.moves.length - 1];
      if (lastUci.length >= 4) {
        setLastMove({
          from: lastUci.slice(0, 2),
          to: lastUci.slice(2, 4),
        });
      }
    }
  }, [gameState?.moves, lastMove]);

  // Show result overlay when game is completed
  useEffect(() => {
    if (gameState?.status === 'completed' || gameState?.status === 'aborted') {
      setShowResult(true);
    }
  }, [gameState?.status]);

  // Board orientation based on player color
  const boardOrientation = useMemo((): 'white' | 'black' => {
    if (playerColor === 'white' || playerColor === 'black') return playerColor;
    return 'white'; // Spectators see from white's perspective
  }, [playerColor]);

  // Handle move from board
  const handleBoardMove = useCallback(
    (move: { from: string; to: string; promotion?: string }) => {
      // Convert to UCI format
      let uci = move.from + move.to;
      if (move.promotion) {
        uci += move.promotion;
      }
      makeMove(uci);
    },
    [makeMove]
  );

  // Can player make moves?
  const canMove = useMemo(() => {
    return (
      isConnected &&
      !isSpectator &&
      isMyTurn &&
      gameState?.status === 'ongoing'
    );
  }, [isConnected, isSpectator, isMyTurn, gameState?.status]);

  // Copy game link
  const copyGameLink = useCallback(() => {
    const url = window.location.href;
    navigator.clipboard.writeText(url);
    // TODO: Show toast
  }, []);

  if (!gameState) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-900">
        <div className="text-center">
          <div className="mb-4 h-12 w-12 animate-spin rounded-full border-4 border-gray-600 border-t-blue-500 mx-auto" />
          <p className="text-gray-400">
            {isConnected ? 'Loading game...' : 'Connecting...'}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      {/* Header */}
      <header className="border-b border-gray-800 bg-gray-900/95 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <button
            onClick={() => router.back()}
            className="flex items-center gap-2 text-gray-400 hover:text-white"
          >
            <ArrowLeft className="h-5 w-5" />
            <span className="hidden sm:inline">Back</span>
          </button>

          <div className="flex items-center gap-2">
            <button
              onClick={copyGameLink}
              className="flex items-center gap-2 rounded-lg bg-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-700"
            >
              <Copy className="h-4 w-4" />
              <span className="hidden sm:inline">Copy Link</span>
            </button>
            <button
              onClick={copyGameLink}
              className="flex items-center gap-2 rounded-lg bg-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-700"
            >
              <Share2 className="h-4 w-4" />
              <span className="hidden sm:inline">Share</span>
            </button>
          </div>
        </div>
      </header>

      {/* Error banner */}
      {error && (
        <div className="bg-red-600 px-4 py-2 text-center text-sm text-white">
          {error}
        </div>
      )}

      {/* Main content */}
      <main className="mx-auto max-w-7xl p-4">
        <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
          {/* Left column - Board */}
          <div className="flex flex-col items-center gap-4">
            {/* Top clock */}
            <div className="w-full max-w-[600px]">
              <GameClocks
                whiteClockMs={gameState.whiteClockMs}
                blackClockMs={gameState.blackClockMs}
                isWhiteTurn={gameState.turn === 'white'}
                isRunning={gameState.status === 'ongoing'}
                orientation={boardOrientation}
                whitePlayerName="White"
                blackPlayerName="Black"
              />
            </div>

            {/* Chess board */}
            <Board
              fen={gameState.fen}
              orientation={boardOrientation}
              onMove={handleBoardMove}
              allowMoves={canMove}
              lastMove={lastMove || undefined}
              inCheck={gameState.inCheck}
            />
          </div>

          {/* Right column - Controls & Move list */}
          <div className="flex flex-col gap-4">
            {/* Game controls */}
            <GameControls
              isSpectator={isSpectator}
              isMyTurn={isMyTurn}
              canAbort={canAbort}
              canResign={gameState.status === 'ongoing' && !isSpectator}
              canOfferDraw={gameState.status === 'ongoing' && !isSpectator}
              hasDrawOffer={hasDrawOffer}
              isDrawOfferMine={isDrawOfferMine}
              isConnected={isConnected}
              opponentConnected={opponentConnected}
              spectatorCount={spectatorCount}
              onResign={resign}
              onOfferDraw={offerDraw}
              onAcceptDraw={acceptDraw}
              onDeclineDraw={declineDraw}
              onAbort={abort}
            />

            {/* Move list */}
            <MoveList
              moves={gameState.moves}
              className="flex-1 min-h-[300px] max-h-[500px]"
            />

            {/* Game info */}
            <div className="rounded-lg bg-gray-800 p-4">
              <h3 className="mb-2 font-medium text-gray-300">Game Info</h3>
              <dl className="space-y-1 text-sm">
                <div className="flex justify-between">
                  <dt className="text-gray-500">Status</dt>
                  <dd className={cn(
                    'font-medium',
                    gameState.status === 'ongoing' && 'text-green-500',
                    gameState.status === 'completed' && 'text-blue-500',
                    gameState.status === 'aborted' && 'text-yellow-500',
                    gameState.status === 'waiting' && 'text-gray-400'
                  )}>
                    {gameState.status}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-500">Moves</dt>
                  <dd className="text-gray-300">{gameState.moveCount}</dd>
                </div>
                {gameState.result && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500">Result</dt>
                    <dd className="font-medium text-white">{gameState.result}</dd>
                  </div>
                )}
              </dl>
            </div>
          </div>
        </div>
      </main>

      {/* Game result overlay */}
      {showResult && gameState.result && (
        <GameResult
          result={gameState.result}
          termination={gameState.termination}
          playerColor={playerColor === 'white' || playerColor === 'black' ? playerColor : null}
          whitePlayerName="White"
          blackPlayerName="Black"
          onClose={() => setShowResult(false)}
          onNewGame={() => router.push('/play')}
        />
      )}
    </div>
  );
}
