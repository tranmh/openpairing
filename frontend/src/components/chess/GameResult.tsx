'use client';

import { Trophy, Minus, X, Clock, Flag, Handshake, AlertTriangle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useLanguageStore } from '@/store/language';

interface GameResultProps {
  result: string;
  termination: string;
  playerColor: 'white' | 'black' | null;
  whitePlayerName?: string;
  blackPlayerName?: string;
  onClose?: () => void;
  onRematch?: () => void;
  onNewGame?: () => void;
  className?: string;
}

const terminationIcons: Record<string, typeof Trophy> = {
  checkmate: Trophy,
  resignation: Flag,
  timeout: Clock,
  draw_agreement: Handshake,
  stalemate: Minus,
  insufficient_material: AlertTriangle,
  threefold_repetition: AlertTriangle,
  fifty_move_rule: AlertTriangle,
  aborted: X,
};

const terminationKeys: Record<string, string> = {
  checkmate: 'game.termination.checkmate',
  resignation: 'game.termination.resignation',
  timeout: 'game.termination.timeout',
  draw_agreement: 'game.termination.drawByAgreement',
  stalemate: 'game.termination.stalemate',
  insufficient_material: 'game.termination.insufficientMaterial',
  threefold_repetition: 'game.termination.threefoldRepetition',
  fifty_move_rule: 'game.termination.fiftyMoveRule',
  aborted: 'game.termination.aborted',
};

export function GameResult({
  result,
  termination,
  playerColor,
  whitePlayerName,
  blackPlayerName,
  onClose,
  onRematch,
  onNewGame,
  className,
}: GameResultProps) {
  const { t } = useLanguageStore();

  const whiteName = whitePlayerName || t('chess.white');
  const blackName = blackPlayerName || t('chess.black');

  const getResultText = () => {
    if (result === '1-0') {
      return { winner: whiteName, loser: blackName, winnerColor: 'white' };
    }
    if (result === '0-1') {
      return { winner: blackName, loser: whiteName, winnerColor: 'black' };
    }
    return { winner: null, loser: null, winnerColor: null };
  };

  const { winner, winnerColor } = getResultText();
  const isDraw = result === '1/2-1/2';
  const isAborted = termination === 'aborted';

  const playerWon = playerColor && winnerColor === playerColor;
  const playerLost = playerColor && !isDraw && !isAborted && winnerColor !== playerColor;
  const isSpectator = playerColor === null;

  const terminationKey = terminationKeys[termination] || termination;
  const terminationLabel = t(terminationKey);
  const TerminationIcon = terminationIcons[termination] || AlertTriangle;

  return (
    <div
      className={cn(
        'fixed inset-0 z-50 flex items-center justify-center bg-black/70',
        className
      )}
    >
      <div className="mx-4 w-full max-w-md overflow-hidden rounded-xl bg-gray-800 shadow-2xl">
        {/* Header */}
        <div
          className={cn(
            'p-6 text-center',
            playerWon && 'bg-green-600',
            playerLost && 'bg-red-600',
            isDraw && 'bg-yellow-600',
            isAborted && 'bg-gray-600',
            isSpectator && 'bg-blue-600'
          )}
        >
          {!isAborted && !isDraw && winner && (
            <>
              <Trophy className="mx-auto mb-2 h-12 w-12" />
              <h2 className="text-2xl font-bold text-white">
                {playerWon
                  ? t('game.result.youWon')
                  : playerLost
                    ? t('game.result.youLost')
                    : `${winner} wins!`}
              </h2>
            </>
          )}

          {isDraw && (
            <>
              <Handshake className="mx-auto mb-2 h-12 w-12" />
              <h2 className="text-2xl font-bold text-white">{t('game.result.draw')}</h2>
            </>
          )}

          {isAborted && (
            <>
              <X className="mx-auto mb-2 h-12 w-12" />
              <h2 className="text-2xl font-bold text-white">{t('game.result.aborted')}</h2>
            </>
          )}
        </div>

        {/* Result details */}
        <div className="p-6">
          {/* Result score */}
          <div className="mb-4 flex items-center justify-center gap-4">
            <div className="text-center">
              <div className="flex items-center justify-center gap-2">
                <div className="h-4 w-4 rounded-full border border-gray-300 bg-white" />
                <span className="font-medium text-gray-300">{whiteName}</span>
              </div>
            </div>
            <div className="text-3xl font-bold text-white">{result}</div>
            <div className="text-center">
              <div className="flex items-center justify-center gap-2">
                <div className="h-4 w-4 rounded-full bg-gray-900" />
                <span className="font-medium text-gray-300">{blackName}</span>
              </div>
            </div>
          </div>

          {/* Termination reason */}
          <div className="mb-6 flex items-center justify-center gap-2 text-gray-400">
            <TerminationIcon className="h-4 w-4" />
            <span>{terminationLabel}</span>
          </div>

          {/* Action buttons */}
          <div className="flex flex-col gap-2">
            {onRematch && (
              <button
                onClick={onRematch}
                className="w-full rounded-lg bg-green-600 px-4 py-3 font-medium text-white hover:bg-green-700"
              >
                {t('chess.rematch')}
              </button>
            )}

            {onNewGame && (
              <button
                onClick={onNewGame}
                className="w-full rounded-lg bg-blue-600 px-4 py-3 font-medium text-white hover:bg-blue-700"
              >
                {t('chess.newGame')}
              </button>
            )}

            {onClose && (
              <button
                onClick={onClose}
                className="w-full rounded-lg bg-gray-700 px-4 py-3 font-medium text-gray-300 hover:bg-gray-600"
              >
                {t('common.close')}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
